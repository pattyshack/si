package allocator

import (
	"fmt"
	"math"
	"strings"

	arch "github.com/pattyshack/chickadee/architecture"
	"github.com/pattyshack/chickadee/ast"
)

type defLoc struct {
	def *ast.VariableDefinition
	loc *arch.DataLocation

	pseudoDefVal ast.Value // global ref or immediate
}

type defAlloc struct {
	definition   *ast.VariableDefinition
	numRegisters int

	nextUseDist int // 0 indicates the definition is dead after this instruction
	constraints []*arch.LocationConstraint

	// numRequired + 1 >= numPreferred >= numRequired >= numClobbered
	//
	// The extra copy in numPreferred is used to ensure the definition outlives
	// the current instruction.
	numPreferred int
	numRequired  int
	numClobbered int

	numActual         int
	hasFixedStackCopy bool
}

type operationsScheduler struct {
	*BlockState

	instruction ast.Instruction
	constraints *arch.InstructionConstraints

	srcs map[*arch.LocationConstraint]*defLoc

	// A temp stack destination allocated during the set up phase.
	tempDest defLoc

	// Note: the set up phase will create a placeholder destination location
	// entry which won't be allocated until the tear down phase.
	finalDest defLoc
}

func newOperationsScheduler(
	state *BlockState,
	inst ast.Instruction,
	constraints *arch.InstructionConstraints,
) *operationsScheduler {
	srcs := map[*arch.LocationConstraint]*defLoc{}
	for idx, value := range inst.Sources() {
		entry := &defLoc{}
		ref, ok := value.(*ast.VariableReference)
		if ok {
			entry.def = ref.UseDef
		} else {
			entry.def = &ast.VariableDefinition{
				StartEndPos:       value.StartEnd(),
				Name:              fmt.Sprintf("%%constant-source-%d", idx),
				Type:              value.Type(),
				ParentInstruction: inst,
			}
			entry.pseudoDefVal = value
		}
		srcs[constraints.Sources[idx]] = entry
	}

	return &operationsScheduler{
		BlockState:  state,
		instruction: inst,
		constraints: constraints,
		srcs:        srcs,
		finalDest: defLoc{
			def: inst.Destination(),
		},
	}
}

func (scheduler *operationsScheduler) ScheduleOperations() {
	_, ok := scheduler.instruction.(*ast.CopyOperation)
	if ok {
		// TODO schedule copy
		return
	}

	scheduler.setUpTempStack()

	// Note: To maximize degrees of freedom / simplify accounting, register
	// pressure is computed after setting up temp stack, which will likely free
	// up some registers.
	pressure, defAllocs := scheduler.computeRegisterPressure()
	scheduler.reduceRegisterPressure(pressure, defAllocs)
	scheduler.setUpRegisters(defAllocs)

	scheduler.executeInstruction()
	scheduler.tearDownInstruction()
}

func (scheduler *operationsScheduler) executeInstruction() {
	srcLocs := []*arch.DataLocation{}
	for _, constraint := range scheduler.constraints.Sources {
		entry, ok := scheduler.srcs[constraint]
		if !ok {
			panic("should never happen")
		}
		srcLocs = append(srcLocs, entry.loc)
	}

	destLoc := scheduler.finalDest.loc
	if scheduler.tempDest.loc != nil {
		destLoc = scheduler.tempDest.loc
	}

	scheduler.BlockState.ExecuteInstruction(
		scheduler.instruction,
		srcLocs,
		destLoc)
}

func (scheduler *operationsScheduler) tearDownInstruction() {
	// NOTE: temp register must be selected before freeing any location since
	// the freed location may have the destination's data.
	tempRegister := scheduler.selectTempRegister()

	// Free all clobbered source locations
	for _, constraint := range scheduler.constraints.Sources {
		if !constraint.ClobberedByInstruction() {
			continue
		}

		entry, ok := scheduler.srcs[constraint]
		if !ok {
			panic("should never happen")
		}
		scheduler.FreeLocation(entry.loc)
	}

	// Free all dead definitions.
	//
	// Note: free location operations are not in deterministic order.  This is
	// ok since free location won't emit any real instruction.
	for def, locs := range scheduler.ValueLocations.Values {
		liveRange, ok := scheduler.LiveRanges[def]
		if !ok {
			panic("should never happen")
		}

		if len(liveRange.NextUses) > 0 {
			continue
		}

		for loc, _ := range locs {
			scheduler.FreeLocation(loc)
		}
	}

	if scheduler.finalDest.def == nil {
		return
	}

	if scheduler.finalDest.loc == nil {
		panic("should never happen")
	} else if scheduler.finalDest.loc.OnTempStack {
		panic("should never happen")
	} else if scheduler.finalDest.loc.OnFixedStack {
		// value must be on temp tack
		if scheduler.tempDest.loc == nil {
			panic("should never happen")
		}

		scheduler.finalDest.loc = scheduler.AllocateFixedStackLocation(
			scheduler.finalDest.def)
	} else {
		// Since sources and destination may share registers, register destination
		// must be allocated after all clobbered sources are freed.
		scheduler.finalDest.loc = scheduler.AllocateRegistersLocation(
			scheduler.finalDest.def,
			scheduler.finalDest.loc.Registers...)
	}

	if scheduler.tempDest.loc != nil {
		// The destination value is on a temp stack
		scheduler.CopyLocation(
			scheduler.tempDest.loc,
			scheduler.finalDest.loc,
			tempRegister)
		scheduler.FreeLocation(scheduler.tempDest.loc)
	}
}

func (scheduler *operationsScheduler) setUpTempStack() {
	tempRegister := scheduler.selectTempRegister()

	var srcDefs []*ast.VariableDefinition
	copySrcs := map[*ast.VariableDefinition]*arch.DataLocation{}
	for _, constraint := range scheduler.constraints.SrcStackLocations {
		entry, ok := scheduler.srcs[constraint]
		if !ok {
			panic("should never happen")
		}
		srcDefs = append(srcDefs, entry.def)

		if entry.pseudoDefVal != nil {
			copySrcs[entry.def] = scheduler.selectCopySourceLocation(entry.def)
		}
	}

	if scheduler.constraints.DestStackLocation != nil {
		scheduler.tempDest.def = &ast.VariableDefinition{
			StartEndPos:       scheduler.finalDest.def.StartEnd(),
			Name:              "%temp-destination",
			Type:              scheduler.finalDest.def.Type,
			ParentInstruction: scheduler.instruction,
		}
	}

	stackSrcLocs, tempDestLoc := scheduler.AllocateTempStackLocations(
		srcDefs,
		scheduler.tempDest.def)

	for idx, constraint := range scheduler.constraints.SrcStackLocations {
		entry, ok := scheduler.srcs[constraint]
		if !ok {
			panic("should never happen")
		}

		loc := stackSrcLocs[idx]
		entry.loc = loc

		if entry.pseudoDefVal != nil {
			scheduler.SetConstantValue(entry.pseudoDefVal, loc, tempRegister)
		} else {
			copySrc, ok := copySrcs[entry.def]
			if !ok {
				panic("should never happen")
			}
			scheduler.CopyLocation(copySrc, loc, tempRegister)
		}
	}

	scheduler.tempDest.loc = tempDestLoc
	if tempDestLoc != nil {
		scheduler.InitializeZeros(tempDestLoc, tempRegister)
	}
}

func (scheduler *operationsScheduler) selectCopySourceLocation(
	def *ast.VariableDefinition,
) *arch.DataLocation {
	set, ok := scheduler.ValueLocations.Values[def]
	if !ok {
		panic("should never happen")
	}

	var selected *arch.DataLocation
	for loc, _ := range set {
		if selected == nil ||
			// Prefer register locations over stack location
			selected.OnFixedStack ||
			selected.OnTempStack ||
			// Pick a deterministic register location
			(len(selected.Registers) > 0 &&
				selected.Registers[0].Index > loc.Registers[0].Index) {

			selected = loc
		}
	}

	return selected
}

// By construction, there's always at least one unused register (this
// assumption is checked by the instruction constraints validator).  The
// function entry point is the only place where all registers could be in
// used; in this case, at least one of the register is a pseudo-source
// callee-saved register that is never used by the function.
func (scheduler *operationsScheduler) selectTempRegister() *arch.Register {
	// The common case fast path
	for _, regInfo := range scheduler.ValueLocations.Registers {
		if regInfo.UsedBy == nil {
			return regInfo.Register
		}
	}

	// Slow path to handle function entry point

	var selected *RegisterInfo
	for _, regInfo := range scheduler.ValueLocations.Registers {
		defName := regInfo.UsedBy.Name

		// Registers holding real definitions are not eligible
		if !strings.HasPrefix(defName, "%") ||
			strings.HasPrefix(defName, "%%") {
			continue
		}

		// Previous frame pointer has highest spill priority
		if defName == arch.PreviousFramePointer {
			selected = regInfo
			break
		}

		// Pick a register deterministically.  Any one will do.
		if selected == nil || selected.Index > regInfo.Index {
			selected = regInfo
		}
	}

	if selected == nil {
		panic("should never happen")
	}

	regLoc := selected.UsedBy
	def, ok := scheduler.ValueLocations.ValueNames[regLoc.Name]
	if !ok {
		panic("should never happen")
	}

	stackLoc := scheduler.AllocateFixedStackLocation(def)
	scheduler.CopyLocation(regLoc, stackLoc, nil)
	scheduler.FreeLocation(regLoc)

	return selected.Register
}

func (scheduler *operationsScheduler) reduceRegisterPressure(
	pressure int,
	defAllocs map[*ast.VariableDefinition]*defAlloc,
) {
	candidates := map[*ast.VariableDefinition]*defAlloc{}
	for def, alloc := range defAllocs {
		if alloc.numRegisters == 0 { // unit data type takes up no space
			continue
		}

		if alloc.numRequired == alloc.numPreferred &&
			alloc.numRequired >= alloc.numActual {
			continue // we can't free any more copies of this definition
		}

		candidates[def] = alloc
	}

	threshold := len(scheduler.ValueLocations.Registers) - 1
	for pressure > threshold {
		candidate := scheduler.selectFreeDefCandidate(candidates)

		shouldSpill := false
		if candidate.numPreferred >= candidate.numActual {
			if candidate.numPreferred <= candidate.numRequired {
				panic("should never happen")
			}

			candidate.numPreferred--

			// Note: if candidate is finalDest, the destination's value is copy to
			// fixed stack during instruction tear down.
			if candidate.definition != scheduler.finalDest.def {
				shouldSpill = true
			}
		}

		if shouldSpill && !candidate.hasFixedStackCopy {
			src := scheduler.selectCopySourceLocation(candidate.definition)
			dest := scheduler.AllocateFixedStackLocation(candidate.definition)
			scheduler.CopyLocation(src, dest, nil)
			candidate.hasFixedStackCopy = true
		}

		if candidate.numActual > candidate.numPreferred {
			scheduler.FreeLocation(scheduler.selectFreeLocation(candidate))
			candidate.numActual--
		}
		pressure -= candidate.numRegisters

		// Prune unfreeable candidate
		if candidate.numRequired == candidate.numPreferred &&
			candidate.numRequired >= candidate.numActual {

			delete(candidates, candidate.definition)
		}
	}
}

func (scheduler *operationsScheduler) computeRegisterPressure() (
	int,
	map[*ast.VariableDefinition]*defAlloc,
) {
	newDefAlloc := func(def *ast.VariableDefinition) *defAlloc {
		liveRange, ok := scheduler.LiveRanges[def]
		if !ok {
			panic("should never happen")
		}
		nextUseDist := 0
		if len(liveRange.NextUses) > 0 {
			nextUseDist = liveRange.NextUses[0]
		}

		return &defAlloc{
			definition:   def,
			numRegisters: arch.NumRegisters(def.Type),
			nextUseDist:  nextUseDist,
		}
	}

	defAllocs := map[*ast.VariableDefinition]*defAlloc{}
	for def, locs := range scheduler.ValueLocations.Values {
		candidate := newDefAlloc(def)
		defAllocs[def] = candidate

		for loc, _ := range locs {
			if loc.OnFixedStack {
				candidate.hasFixedStackCopy = true
			} else if loc.OnTempStack {
				// Do nothing.
			} else {
				candidate.numActual++
			}
		}
	}

	registerCandidates := map[*arch.RegisterCandidate]struct{}{}
	for constraint, defLoc := range scheduler.srcs {
		candidate, ok := defAllocs[defLoc.def]
		if !ok {
			if defLoc.pseudoDefVal == nil {
				panic("should never happen")
			}

			candidate = &defAlloc{
				definition:   defLoc.def,
				numRegisters: arch.NumRegisters(defLoc.pseudoDefVal.Type()),
			}
			defAllocs[defLoc.def] = candidate
		}

		if constraint.AnyLocation || constraint.RequireOnStack {
			if candidate.numRequired == candidate.numPreferred {
				candidate.numPreferred++
			}
		} else {
			for _, reg := range constraint.Registers {
				registerCandidates[reg] = struct{}{}
			}

			candidate.constraints = append(candidate.constraints, constraint)
			candidate.numPreferred++
			candidate.numRequired++

			if constraint.ClobberedByInstruction() {
				candidate.numClobbered++
			}
		}
	}

	registerPressure := 0
	if scheduler.constraints.Destination != nil {
		constraint := scheduler.constraints.Destination
		if constraint.AnyLocation || constraint.RequireOnStack {
			// Note: If numPreferred is 0 after pressure reduction, the final
			// destination is on fixed stack (we can skip copying to final
			// destination if nextUseDist is also 0)
			defAllocs[scheduler.finalDest.def] = newDefAlloc(scheduler.finalDest.def)
		} else {
			// Account for destination registers that do not overlap with source
			// registers
			for _, reg := range constraint.Registers {
				_, ok := registerCandidates[reg]
				if !ok {
					registerPressure++
				}
			}
		}
	}

	for _, candidate := range defAllocs {
		numCopies := candidate.numActual

		// If the definition out lives the instruction, ensure at least one copy of
		// the source definition survive.
		if candidate.nextUseDist > 0 &&
			candidate.numRequired == candidate.numClobbered &&
			candidate.numPreferred == candidate.numRequired {

			candidate.numPreferred++
		}

		// For now, assume there's enough room to keep everything on registers.
		if candidate.numPreferred > numCopies {
			numCopies = candidate.numPreferred
		}
		registerPressure += numCopies * candidate.numRegisters
	}

	return registerPressure, defAllocs
}

// TODO prioritize large objects
//
// Free preferences (from highest to lowest):
//   - most discardable extra copies (does not spill)
//   - candidate is already on fixed stack (already spill)
//   - callee-saved parameters (spill / use exactly once)
//   - spill furthest nextUseDist (this heuristic hopeful keeps register
//     pressure low for longer, but could cause instruction pipeline stall)
//
// All preferences are tie break by name to make the selection deterministic.
func (scheduler *operationsScheduler) selectFreeDefCandidate(
	candidates map[*ast.VariableDefinition]*defAlloc,
) *defAlloc {
	// Prefer to free candidate with extra discardable copies
	maxExtraCopies := 0
	var selected *defAlloc
	for _, candidate := range candidates {
		if candidate.numPreferred >= candidate.numActual {
			// The candidate cannot be unconditionally discarded
			continue
		}

		extraCopies := candidate.numActual - candidate.numClobbered
		if extraCopies > maxExtraCopies {
			maxExtraCopies = extraCopies
			selected = candidate
		} else if extraCopies == maxExtraCopies &&
			arch.CompareDefinitionNames(
				candidate.definition.Name,
				selected.definition.Name) < 0 {
			selected = candidate
		}
	}
	if selected != nil {
		return selected
	}

	// Prefer to free candidate that are already on stack
	for _, candidate := range candidates {
		if candidate.hasFixedStackCopy {
			if selected == nil {
				selected = candidate
			} else if arch.CompareDefinitionNames(
				candidate.definition.Name,
				selected.definition.Name) < 0 {

				selected = candidate
			}
		}
	}
	if selected != nil {
		return selected
	}

	// Prefer callee-saved parameters
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate.definition.Name, "%") {
			if selected == nil {
				selected = candidate
			} else if arch.CompareDefinitionNames(
				candidate.definition.Name,
				selected.definition.Name) < 0 {

				selected = candidate
			}
		}
	}
	if selected != nil {
		return selected
	}

	// Prefer further nextUseDist
	for _, candidate := range candidates {
		if selected == nil {
			selected = candidate
		} else if candidate.nextUseDist > selected.nextUseDist {
			selected = candidate
		} else if candidate.nextUseDist == selected.nextUseDist &&
			arch.CompareDefinitionNames(
				candidate.definition.Name,
				selected.definition.Name) < 0 {
			selected = candidate
		}
	}

	if selected == nil {
		panic("should never happen")
	}
	return selected
}

func (scheduler *operationsScheduler) selectFreeLocation(
	candidate *defAlloc,
) *arch.DataLocation {
	worstMatch := math.MaxInt32
	var selected *arch.DataLocation
	for loc, _ := range scheduler.ValueLocations.Values[candidate.definition] {
		if loc.OnFixedStack || loc.OnTempStack {
			continue
		}

		match := 0
		for _, constraint := range candidate.constraints {
			numSatisfied := 0
			for idx, reg := range loc.Registers {
				if constraint.Registers[idx].SatisfyBy(reg) {
					numSatisfied++
				}
			}

			if numSatisfied > match {
				match = numSatisfied
			}
		}

		if worstMatch > match {
			worstMatch = match
			selected = loc
		} else if worstMatch == match &&
			loc.Registers[0].Index > selected.Registers[0].Index {

			// (arbitrary) deterministic tie break
			selected = loc
		}
	}

	if selected == nil {
		panic("should never happen")
	}
	return selected
}

func (scheduler *operationsScheduler) setUpRegisters(
	defAllocs map[*ast.VariableDefinition]*defAlloc,
) {
	// TODO
}

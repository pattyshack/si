package allocator

import (
	"fmt"
	"strings"

	arch "github.com/pattyshack/chickadee/architecture"
	"github.com/pattyshack/chickadee/ast"
)

type defLoc struct {
	def *ast.VariableDefinition
	loc *arch.DataLocation

	pseudoDefVal ast.Value // global ref or immediate
}

type freeDefCandidate struct {
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

	numActualCopies   int
	hasFixedStackCopy bool
}

type instructionAllocator struct {
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

func newInstructionAllocator(
	state *BlockState,
	inst ast.Instruction,
	constraints *arch.InstructionConstraints,
) *instructionAllocator {
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

	return &instructionAllocator{
		BlockState:  state,
		instruction: inst,
		constraints: constraints,
		srcs:        srcs,
		finalDest: defLoc{
			def: inst.Destination(),
		},
	}
}

func (allocator *instructionAllocator) SetUpInstruction() {
	allocator.setUpTempStack()

	// Note: To maximize degrees of freedom / simplify accounting, register
	// pressure is computed after setting up temp stack, which will likely free
	// up some registers.
	allocator.reduceRegisterPressure()
	allocator.setUpRegisters()
}

func (allocator *instructionAllocator) ExecuteInstruction() {
	_, ok := allocator.instruction.(*ast.CopyOperation)
	if ok {
		return // value already copied during the set up phase
	}

	srcLocs := []*arch.DataLocation{}
	for _, constraint := range allocator.constraints.Sources {
		entry, ok := allocator.srcs[constraint]
		if !ok {
			panic("should never happen")
		}
		srcLocs = append(srcLocs, entry.loc)
	}

	destLoc := allocator.finalDest.loc
	if allocator.tempDest.loc != nil {
		destLoc = allocator.tempDest.loc
	}

	allocator.BlockState.ExecuteInstruction(
		allocator.instruction,
		srcLocs,
		destLoc)
}

func (allocator *instructionAllocator) TearDownInstruction() {
	// NOTE: temp register must be selected before freeing any location since
	// the freed location may have the destination's data.
	tempRegister := allocator.selectTempRegister()

	// Free all clobbered source locations
	for _, constraint := range allocator.constraints.Sources {
		if !constraint.ClobberedByInstruction() {
			continue
		}

		entry, ok := allocator.srcs[constraint]
		if !ok {
			panic("should never happen")
		}
		allocator.FreeLocation(entry.loc)
	}

	// Free all dead definitions.
	//
	// Note: free location operations are not in deterministic order.  This is
	// ok since free location won't emit any real instruction.
	for def, locs := range allocator.ValueLocations.Values {
		liveRange, ok := allocator.LiveRanges[def]
		if !ok {
			panic("should never happen")
		}

		if len(liveRange.NextUses) > 0 {
			continue
		}

		for loc, _ := range locs {
			allocator.FreeLocation(loc)
		}
	}

	if allocator.finalDest.def == nil {
		return
	}

	if allocator.finalDest.loc == nil {
		panic("should never happen")
	} else if allocator.finalDest.loc.OnTempStack {
		panic("should never happen")
	} else if allocator.finalDest.loc.OnFixedStack {
		// value must be on temp tack
		if allocator.tempDest.loc == nil {
			panic("should never happen")
		}

		allocator.finalDest.loc = allocator.AllocateFixedStackLocation(
			allocator.finalDest.def)
	} else {
		// Since sources and destination may share registers, register destination
		// must be allocated after all clobbered sources are freed.
		allocator.finalDest.loc = allocator.AllocateRegistersLocation(
			allocator.finalDest.def,
			allocator.finalDest.loc.Registers...)
	}

	if allocator.tempDest.loc != nil {
		// The destination value is on a temp stack
		allocator.CopyLocation(
			allocator.tempDest.loc,
			allocator.finalDest.loc,
			tempRegister)
		allocator.FreeLocation(allocator.tempDest.loc)
	}
}

func (allocator *instructionAllocator) setUpTempStack() {
	tempRegister := allocator.selectTempRegister()

	var srcDefs []*ast.VariableDefinition
	copySrcs := map[*ast.VariableDefinition]*arch.DataLocation{}
	for _, constraint := range allocator.constraints.SrcStackLocations {
		entry, ok := allocator.srcs[constraint]
		if !ok {
			panic("should never happen")
		}
		srcDefs = append(srcDefs, entry.def)

		if entry.pseudoDefVal != nil {
			copySrcs[entry.def] = allocator.selectCopySourceLocation(entry.def)
		}
	}

	if allocator.constraints.DestStackLocation != nil {
		allocator.tempDest.def = &ast.VariableDefinition{
			StartEndPos:       allocator.finalDest.def.StartEnd(),
			Name:              "%temp-destination",
			Type:              allocator.finalDest.def.Type,
			ParentInstruction: allocator.instruction,
		}
	}

	stackSrcLocs, tempDestLoc := allocator.AllocateTempStackLocations(
		srcDefs,
		allocator.tempDest.def)

	for idx, constraint := range allocator.constraints.SrcStackLocations {
		entry, ok := allocator.srcs[constraint]
		if !ok {
			panic("should never happen")
		}

		loc := stackSrcLocs[idx]
		entry.loc = loc

		if entry.pseudoDefVal != nil {
			allocator.SetConstantValue(entry.pseudoDefVal, loc, tempRegister)
		} else {
			copySrc, ok := copySrcs[entry.def]
			if !ok {
				panic("should never happen")
			}
			allocator.CopyLocation(copySrc, loc, tempRegister)
		}
	}

	allocator.tempDest.loc = tempDestLoc
	if tempDestLoc != nil {
		allocator.InitializeZeros(tempDestLoc, tempRegister)
	}
}

func (allocator *instructionAllocator) selectCopySourceLocation(
	def *ast.VariableDefinition,
) *arch.DataLocation {
	set, ok := allocator.ValueLocations.Values[def]
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
func (allocator *instructionAllocator) selectTempRegister() *arch.Register {
	// The common case fast path
	for _, regInfo := range allocator.ValueLocations.Registers {
		if regInfo.UsedBy == nil {
			return regInfo.Register
		}
	}

	// Slow path to handle function entry point

	var selected *RegisterInfo
	for _, regInfo := range allocator.ValueLocations.Registers {
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
	def, ok := allocator.ValueLocations.ValueNames[regLoc.Name]
	if !ok {
		panic("should never happen")
	}

	stackLoc := allocator.AllocateFixedStackLocation(def)
	allocator.CopyLocation(regLoc, stackLoc, nil)
	allocator.FreeLocation(regLoc)

	return selected.Register
}

func (allocator *instructionAllocator) reduceRegisterPressure() {
	pressure, freeDefCandidates := allocator.computeRegisterPressure()
	threshold := len(allocator.ValueLocations.Registers) - 1
	for pressure > threshold {
		candidate := allocator.selectFreeDefCandidate(freeDefCandidates)

		shouldSpill := false
		if candidate.numActualCopies > candidate.numPreferred {
			candidate.numActualCopies-- // Just free an extra copy
		} else {
			if candidate.numPreferred <= candidate.numRequired {
				panic("should never happen")
			}

			if candidate.numActualCopies == candidate.numPreferred {
				candidate.numActualCopies--
			}

			candidate.numPreferred--
			shouldSpill = true
		}

		if shouldSpill && !candidate.hasFixedStackCopy {
			src := allocator.selectCopySourceLocation(candidate.definition)
			dest := allocator.AllocateFixedStackLocation(candidate.definition)
			allocator.CopyLocation(src, dest, nil)
			candidate.hasFixedStackCopy = true
		}

		allocator.FreeLocation(allocator.selectFreeLocation(candidate))
		pressure -= candidate.numRegisters

		// Prune unfreeable candidate
		if candidate.numPreferred == candidate.numRequired &&
			candidate.numRequired >= candidate.numActualCopies {

			delete(freeDefCandidates, candidate.definition)
		}
	}
}

func (allocator *instructionAllocator) computeRegisterPressure() (
	int,
	map[*ast.VariableDefinition]*freeDefCandidate,
) {
	candidates := map[*ast.VariableDefinition]*freeDefCandidate{}
	for def, locs := range allocator.ValueLocations.Values {
		numRegisters := arch.NumRegisters(def.Type)
		if numRegisters == 0 {
			// Unit value type is never a candidate since it takes up no space
			continue
		}

		liveRange, ok := allocator.LiveRanges[def]
		if !ok {
			panic("should never happen")
		}
		nextUseDist := 0
		if len(liveRange.NextUses) > 0 {
			nextUseDist = liveRange.NextUses[0]
		}

		candidate := &freeDefCandidate{
			definition:   def,
			numRegisters: numRegisters,
			nextUseDist:  nextUseDist,
		}
		candidates[def] = candidate

		for loc, _ := range locs {
			if loc.OnFixedStack {
				candidate.hasFixedStackCopy = true
			} else if loc.OnTempStack {
				// Do nothing.
			} else {
				candidate.numActualCopies++
			}
		}
	}

	registerPressure := 0
	registerCandidates := map[*arch.RegisterCandidate]struct{}{}
	for constraint, defLoc := range allocator.srcs {
		if constraint.NumRegisters == 0 {
			// Unit value type is never a candidate since it takes up no space
			continue
		}

		candidate, ok := candidates[defLoc.def]
		if !ok {
			if defLoc.pseudoDefVal == nil {
				panic("should never happen")
			}

			// Make room for the constant source value
			registerPressure += constraint.NumRegisters
			continue
		}

		if constraint.RequireOnStack {
			// Do nothing.  We've already set up the temp stack
		} else if constraint.AnyLocation { // only used by copy operation
			candidate.numPreferred++
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

	// Account for destination registers that do not overlap with source registers
	if allocator.constraints.Destination != nil {
		for _, reg := range allocator.constraints.Destination.Registers {
			_, ok := registerCandidates[reg]
			if !ok {
				registerPressure++
			}
		}
	}

	for def, candidate := range candidates {
		numCopies := candidate.numActualCopies

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

		// Prune candidates that can never be freed
		if candidate.numRequired >= candidate.numActualCopies &&
			candidate.numPreferred == candidate.numRequired {

			delete(candidates, def)
		}
	}

	return registerPressure, candidates
}

// Free preferences (from highest to lowest):
//   - most discardable extra copies (does not spill)
//   - candidate is already on fixed stack (already spill)
//   - callee-saved parameters (spill / use exactly once)
//   - spill furthest nextUseDist (this heuristic hopeful keeps register
//     pressure low for longer, but could cause instruction pipeline stall)
//
// All preferences are tie break by name to make the selection deterministic.
func (allocator *instructionAllocator) selectFreeDefCandidate(
	candidates map[*ast.VariableDefinition]*freeDefCandidate,
) *freeDefCandidate {
	// Prefer to free candidate with extra discardable copies
	maxExtraCopies := 0
	var selected *freeDefCandidate
	for _, candidate := range candidates {
		if candidate.numPreferred >= candidate.numActualCopies {
			// The candidate cannot be unconditionally discarded
			continue
		}

		extraCopies := candidate.numActualCopies - candidate.numClobbered
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

func (allocator *instructionAllocator) selectFreeLocation(
	candidate *freeDefCandidate,
) *arch.DataLocation {
	// TODO
	return nil
}

func (allocator *instructionAllocator) setUpRegisters() {
	// TODO
}

package allocator

import (
	"fmt"
	"math"
	"sort"
	"strings"

	arch "github.com/pattyshack/chickadee/architecture"
	"github.com/pattyshack/chickadee/ast"
)

type constrainedLocation struct {
	def *ast.VariableDefinition

	// NOTE: loc is fully allocated iff loc != nil and misplacedChunks is empty
	loc *arch.DataLocation

	pseudoDefVal ast.Value // global ref or immediate

	constraint      *arch.LocationConstraint
	misplacedChunks []int

	// When false, loc has not been allocated, and loc.Registers is partially
	// reserved and may include nils.  When true, loc is allocated and
	// correctly initialized.
	hasAllocated bool
}

type defAlloc struct {
	definition   *ast.VariableDefinition
	numRegisters int

	// 0 indicates the definition is dead after this instruction
	nextUseDelta int // = nextUseDist - currentDist

	constrained []*constrainedLocation // only used by sources

	// numRequired + 1 >= numPreferred >= numRequired >= numClobbered
	//
	// The extra copy in numPreferred is used to ensure the definition outlives
	// the current instruction.
	numPreferred int
	numRequired  int
	numClobbered int

	numActual         int
	hasFixedStackCopy bool

	isPseudoDefVal bool
}

func (alloc *defAlloc) RegisterLocations() []*constrainedLocation {
	locs := make([]*constrainedLocation, 0, len(alloc.constrained))
	for _, loc := range alloc.constrained {
		if loc.constraint.AnyLocation || loc.constraint.RequireOnStack {
			continue
		}
		locs = append(locs, loc)
	}

	return locs
}

type operationsScheduler struct {
	*BlockState

	*RegisterSelector

	currentDist int
	instruction ast.Instruction
	constraints *arch.InstructionConstraints

	srcs []*constrainedLocation

	// A temp stack destination allocated during the set up phase.
	tempDest constrainedLocation

	// Scratch register used for copying to fixed stack finalDest.
	// Note that this register must be selected at the same time as final
	// destination since final destination is defer allocated.
	destScratchRegister *arch.Register

	// Note: the set up phase will create a placeholder destination location
	// entry which won't be allocated until the tear down phase.
	finalDest constrainedLocation
}

func newOperationsScheduler(
	state *BlockState,
	currentDist int,
) *operationsScheduler {
	return &operationsScheduler{
		BlockState:       state,
		RegisterSelector: NewRegisterSelector(state),
		currentDist:      currentDist,
	}
}

func (scheduler *operationsScheduler) initializeInstructionConstraints(
	inst ast.Instruction,
	constraints *arch.InstructionConstraints,
) {
	srcValues := inst.Sources()
	srcs := make([]*constrainedLocation, 0, len(srcValues))
	for idx, value := range srcValues {
		entry := &constrainedLocation{
			constraint: constraints.Sources[idx],
		}
		srcs = append(srcs, entry)

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
	}

	destDef := inst.Destination()
	tempDest := constrainedLocation{}
	finalDest := constrainedLocation{
		def: destDef,
	}

	if constraints.Destination != nil && constraints.Destination.RequireOnStack {
		tempDest.def = &ast.VariableDefinition{
			StartEndPos:       destDef.StartEnd(),
			Name:              "%temp-destination",
			Type:              destDef.Type,
			ParentInstruction: inst,
		}
		tempDest.constraint = constraints.Destination

		finalDest.constraint = &arch.LocationConstraint{
			NumRegisters: constraints.Destination.NumRegisters,
			AnyLocation:  true,
		}
	} else {
		finalDest.constraint = constraints.Destination
	}

	scheduler.instruction = inst
	scheduler.constraints = constraints
	scheduler.srcs = srcs
	scheduler.tempDest = tempDest
	scheduler.finalDest = finalDest
}

func (scheduler *operationsScheduler) ScheduleInstructionOperations(
	instruction ast.Instruction,
	constraints *arch.InstructionConstraints,
) {
	scheduler.initializeInstructionConstraints(instruction, constraints)

	_, ok := scheduler.instruction.(*ast.CopyOperation)
	if ok {
		scheduler.scheduleCopy()
		return
	}

	scratchRegister := scheduler.SelectScratch()
	scheduler.setUpTempStack(scratchRegister)

	// Note: To maximize degrees of freedom / simplify accounting, register
	// pressure is computed after setting up temp stack, which will likely free
	// up some registers.
	pressure, defAllocs, finalDestAlloc := scheduler.computeRegisterPressure()
	scheduler.reduceRegisterPressure(pressure, defAllocs, scratchRegister)

	// Release scratch register before register selection since it may be picked
	// by the instruction.
	scheduler.ReleaseScratch(scratchRegister)

	scheduler.setUpRegisters(defAllocs, finalDestAlloc)

	scheduler.executeInstruction()

	copyTempDest := finalDestAlloc != nil && finalDestAlloc.nextUseDelta > 0
	scheduler.tearDownInstruction(copyTempDest)
}

func (scheduler *operationsScheduler) scheduleCopy() {
	defer func() {
		// Only invoke tear down to remove source definition (if it's dead).
		scheduler.srcs = nil
		scheduler.tempDest = constrainedLocation{}
		scheduler.finalDest = constrainedLocation{}
		scheduler.tearDownInstruction(false)
	}()

	if len(scheduler.srcs) != 1 {
		panic("should never happen")
	}
	src := scheduler.srcs[0]

	if src.pseudoDefVal != nil {
		// We don't need to allocate space for global reference / immediate source,
		// but we still need to allocate space for the destination.
		scheduler.srcs = nil
	}

	pressure, defAllocs, finalDestAlloc := scheduler.computeRegisterPressure()

	// If the dest defintion is never used, we only need to clean up dead
	// source definition
	if finalDestAlloc.nextUseDelta == 0 {
		return
	}

	var srcAlloc *defAlloc
	for _, alloc := range defAllocs {
		if alloc.definition == src.def {
			srcAlloc = alloc
			break
		}
	}

	destDef := finalDestAlloc.definition

	// Src remains alive after this instruction, but there are multiple register
	// copies of src. Just transfer one of the register copies to dest.
	if srcAlloc != nil && srcAlloc.nextUseDelta > 0 && srcAlloc.numActual > 1 {
		scheduler.transferLocations(src.def, destDef, true)
		return
	}

	// Src and dest shares the same name, and src is dead after this instruction.
	// Transfer all locations from src to dest.
	if srcAlloc != nil && src.def.Name == destDef.Name {
		scheduler.transferLocations(src.def, destDef, false)
		return
	}

	// Src has register locations and is dead after this instruction.  Transfer
	// all register locations from src to dest.
	if srcAlloc != nil && srcAlloc.nextUseDelta == 0 && srcAlloc.numActual > 0 {
		scheduler.transferLocations(src.def, destDef, false)
		return
	}

	scratchRegister := scheduler.SelectScratch()
	scheduler.reduceRegisterPressure(pressure, defAllocs, scratchRegister)
	scheduler.ReleaseScratch(scratchRegister)

	scheduler.setUpFinalDestination(finalDestAlloc)
	destLoc := scheduler.finalDest.loc
	if destLoc == nil || destLoc.OnTempStack {
		panic("should never happen")
	} else if destLoc.OnFixedStack {
		destLoc = scheduler.AllocateFixedStackLocation(destDef)
	} else {
		destLoc = scheduler.AllocateRegistersLocation(destDef, destLoc.Registers...)
	}

	scratchRegister = scheduler.destScratchRegister
	if scratchRegister == nil {
		scratchRegister = scheduler.SelectScratch()
	}

	if src.pseudoDefVal != nil {
		scheduler.SetConstantValue(src.pseudoDefVal, destLoc, scratchRegister)
	} else {
		scheduler.CopyLocation(
			scheduler.selectCopySourceLocation(src.def),
			destLoc,
			scratchRegister)
	}
}

func (scheduler *operationsScheduler) transferLocations(
	srcDef *ast.VariableDefinition,
	destDef *ast.VariableDefinition,
	singleRegisterCopy bool,
) {
	locs := append(
		[]*arch.DataLocation{},
		scheduler.ValueLocations.Values[srcDef]...)
	for _, loc := range locs {
		if loc.OnTempStack {
			panic("should never happen")
		} else if loc.OnFixedStack {
			if srcDef.Name != destDef.Name {
				// We can't transfer the fixed stack location since src and dest occupy
				// different stack locations.
				continue
			}
			scheduler.FreeLocation(loc)
			scheduler.AllocateFixedStackLocation(destDef)
		} else {
			scheduler.FreeLocation(loc)
			scheduler.AllocateRegistersLocation(destDef, loc.Registers...)

			if singleRegisterCopy {
				return
			}
		}
	}

	if singleRegisterCopy {
		panic("should never reach here")
	}
}

func (scheduler *operationsScheduler) setUpTempStack(
	scratchRegister *arch.Register,
) {
	var tempStackSrcs []*constrainedLocation
	var srcDefs []*ast.VariableDefinition
	copySrcs := map[*ast.VariableDefinition]*arch.DataLocation{}
	for _, src := range scheduler.srcs {
		if !src.constraint.RequireOnStack {
			continue
		}

		tempStackSrcs = append(tempStackSrcs, src)
		srcDefs = append(srcDefs, src.def)

		if src.pseudoDefVal == nil {
			copySrcs[src.def] = scheduler.selectCopySourceLocation(src.def)
		}
	}

	stackSrcLocs, tempDestLoc := scheduler.AllocateTempStackLocations(
		srcDefs,
		scheduler.tempDest.def)

	for idx, src := range tempStackSrcs {
		loc := stackSrcLocs[idx]
		if src.pseudoDefVal != nil {
			scheduler.SetConstantValue(src.pseudoDefVal, loc, scratchRegister)
		} else {
			copySrc, ok := copySrcs[src.def]
			if !ok {
				panic("should never happen")
			}
			scheduler.CopyLocation(copySrc, loc, scratchRegister)
		}

		src.loc = loc
		src.hasAllocated = true
	}

	scheduler.tempDest.loc = tempDestLoc
	if tempDestLoc != nil {
		scheduler.InitializeZeros(tempDestLoc, scratchRegister)
		scheduler.tempDest.hasAllocated = true
	}
}

func (scheduler *operationsScheduler) computeRegisterPressure() (
	int,
	[]*defAlloc,
	*defAlloc,
) {
	newDefAlloc := func(def *ast.VariableDefinition) *defAlloc {
		nextUseDelta := 0
		liveRange, ok := scheduler.LiveRanges[def]
		if ok && len(liveRange.NextUses) > 0 {
			nextUseDelta = liveRange.NextUses[0] - scheduler.currentDist
		}

		return &defAlloc{
			definition:   def,
			numRegisters: arch.NumRegisters(def.Type),
			nextUseDelta: nextUseDelta,
		}
	}

	mappedDefAllocs := map[*ast.VariableDefinition]*defAlloc{}
	for def, locs := range scheduler.ValueLocations.Values {
		candidate := newDefAlloc(def)
		mappedDefAllocs[def] = candidate

		for _, loc := range locs {
			if loc.OnFixedStack {
				candidate.hasFixedStackCopy = true
			} else if loc.OnTempStack {
				// Do nothing.
			} else {
				candidate.numActual++
			}
		}
	}

	registerConstraints := map[*arch.RegisterConstraint]struct{}{}
	for _, src := range scheduler.srcs {
		candidate, ok := mappedDefAllocs[src.def]
		if !ok {
			if src.pseudoDefVal == nil {
				panic("should never happen")
			}

			candidate = &defAlloc{
				definition:     src.def,
				numRegisters:   arch.NumRegisters(src.pseudoDefVal.Type()),
				isPseudoDefVal: true,
			}
			mappedDefAllocs[src.def] = candidate
		}

		if src.constraint.AnyLocation || src.constraint.RequireOnStack {
			if candidate.numRequired == candidate.numPreferred {
				candidate.numPreferred++
			}
		} else {
			for _, reg := range src.constraint.Registers {
				registerConstraints[reg] = struct{}{}
			}

			candidate.constrained = append(candidate.constrained, src)
			candidate.numPreferred++
			candidate.numRequired++

			if src.constraint.ClobberedByInstruction() {
				candidate.numClobbered++
			}
		}
	}

	var finalDestAlloc *defAlloc
	registerPressure := 0
	if scheduler.constraints.Destination != nil {
		constraint := scheduler.constraints.Destination
		if constraint.AnyLocation || constraint.RequireOnStack {
			// Note: If numPreferred is 0 after pressure reduction, the final
			// destination is on fixed stack (we can skip copying to final
			// destination if nextUseDelta is also 0)
			finalDestAlloc = newDefAlloc(scheduler.finalDest.def)
			mappedDefAllocs[scheduler.finalDest.def] = finalDestAlloc
		} else {
			// Account for destination registers that do not overlap with source
			// registers
			for _, reg := range constraint.Registers {
				_, ok := registerConstraints[reg]
				if !ok {
					registerPressure++
				}
			}
		}
	}

	defAllocs := make([]*defAlloc, 0, len(mappedDefAllocs))
	for _, candidate := range mappedDefAllocs {
		defAllocs = append(defAllocs, candidate)

		numCopies := candidate.numActual

		// If the definition out lives the instruction, ensure at least one copy of
		// the source definition survive.
		if candidate.nextUseDelta > 0 &&
			candidate.numRequired == candidate.numClobbered &&
			candidate.numPreferred == candidate.numRequired {

			// Ensure we don't eagerly load value back onto registers if the value
			// only exist on stack and is not used by the instruction.
			if !candidate.hasFixedStackCopy ||
				candidate.numActual > 0 ||
				candidate.numRequired > 0 {

				candidate.numPreferred++
			}
		}

		// For now, assume there's enough room to keep everything on registers.
		if candidate.numPreferred > numCopies {
			numCopies = candidate.numPreferred
		}
		registerPressure += numCopies * candidate.numRegisters
	}

	sort.Slice(
		defAllocs,
		func(i int, j int) bool {
			return arch.CompareDefinitionNames(
				defAllocs[i].definition.Name,
				defAllocs[j].definition.Name) < 0
		})

	return registerPressure, defAllocs, finalDestAlloc
}

func (scheduler *operationsScheduler) reduceRegisterPressure(
	pressure int,
	defAllocs []*defAlloc,
	scratchRegister *arch.Register,
) {
	candidates := make([]*defAlloc, 0, len(defAllocs))
	for _, alloc := range defAllocs {
		if alloc.numRegisters == 0 { // unit data type takes up no space
			continue
		}

		if alloc.numRequired == alloc.numPreferred &&
			alloc.numRequired >= alloc.numActual {
			continue // we can't free any more copies of this definition
		}

		candidates = append(candidates, alloc)
	}

	threshold := len(scheduler.ValueLocations.Registers) - 1
	for pressure > threshold {
		idx, candidate := scheduler.selectFreeDefCandidate(candidates)

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
			scheduler.CopyLocation(src, dest, scratchRegister)
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

			// Shift all remaining candidates left to preserve ordering.
			nextIdx := idx + 1
			for nextIdx < len(candidates) {
				candidates[idx] = candidates[nextIdx]
				idx = nextIdx
				nextIdx++
			}
			candidates = candidates[:len(candidates)-1]
		}
	}
}

func (scheduler *operationsScheduler) initUnitLocations(
	alloc *defAlloc,
) {
	locs := scheduler.ValueLocations.RegisterLocations(alloc.definition)
	if len(locs) != 1 {
		panic("should never happen")
	}

	loc := locs[0]
	for _, constrained := range alloc.RegisterLocations() {
		scheduler.assignAllocatedLocation(constrained, loc)
	}
}

func (scheduler *operationsScheduler) setUpRegisters(
	allocs []*defAlloc,
	finalDestAlloc *defAlloc,
) {
	// TODO setup frame pointer for calls
	misplaced := make([]*constrainedLocation, 0, len(scheduler.srcs))
	for _, alloc := range allocs {
		if len(alloc.constrained) == 0 || alloc == finalDestAlloc {
			continue
		} else if alloc.isPseudoDefVal {
			for _, loc := range alloc.RegisterLocations() {
				scheduler.assignUnallocatedLocation(alloc, loc)
				misplaced = append(misplaced, loc)
			}
		} else if alloc.numRegisters == 0 {
			scheduler.initUnitLocations(alloc)
		} else {
			misplaced = scheduler.initSourceRegisterLocations(alloc, misplaced)
		}
	}

	for len(misplaced) > 0 {
		selectableChanged := false
		misplaced, selectableChanged = scheduler.reduceMisplacedWithoutEvication(
			misplaced)

		if len(misplaced) == 0 {
			break
		}

		if selectableChanged {
			continue
		}

		// Evict a register used by a unselected location to make room for this
		// constraint.
		loc := misplaced[0]
		regIdx := loc.misplacedChunks[0]
		loc.misplacedChunks = loc.misplacedChunks[1:]

		selected := scheduler.SelectSourceRegister(
			loc.constraint.Registers[regIdx],
			false)
		if loc.hasAllocated {
			scheduler.MoveRegister(loc.loc.Registers[regIdx], selected)
		} else {
			loc.loc.Registers[regIdx] = selected
		}

		if len(loc.misplacedChunks) == 0 {
			scheduler.finalizeSourceRegistersLocation(loc)
			misplaced = misplaced[1:]
		}
	}

	if scheduler.finalDest.def != nil {
		scheduler.setUpFinalDestination(finalDestAlloc)
	}

	// Create unconstrained preferred copies that only serve to keep definitions
	// alive.
	//
	// NOTE: We must finalize destination before creating preferred copies since
	// the destination may have specific register constraints.
	for _, alloc := range allocs {
		if alloc == finalDestAlloc || alloc.isPseudoDefVal {
			continue
		}

		numNeeded := alloc.numPreferred - alloc.numActual
		if numNeeded <= 0 {
			continue
		} else if numNeeded != 1 {
			panic("should never happen")
		}

		src := scheduler.selectCopySourceLocation(alloc.definition)

		registers := make([]*arch.Register, alloc.numRegisters)
		for idx, _ := range registers {
			registers[idx] = scheduler.SelectFreeRegister()
		}

		loc := scheduler.AllocateRegistersLocation(alloc.definition, registers...)
		alloc.numActual++

		scheduler.CopyLocation(src, loc, nil)
	}
}

func (scheduler *operationsScheduler) assignUnallocatedLocation(
	alloc *defAlloc,
	constrained *constrainedLocation,
) {
	if constrained.loc != nil {
		panic("should never happen")
	}

	misplacedChunks := make([]int, alloc.numRegisters)
	for idx := 0; idx < alloc.numRegisters; idx++ {
		misplacedChunks[idx] = idx
	}
	constrained.misplacedChunks = misplacedChunks

	constrained.loc = &arch.DataLocation{
		Registers: make([]*arch.Register, alloc.numRegisters),
	}

	alloc.numActual++
}

func (scheduler *operationsScheduler) assignAllocatedLocation(
	constrained *constrainedLocation,
	loc *arch.DataLocation,
) bool {
	if constrained.loc != nil {
		panic("should never happen")
	}

	constrained.loc = loc
	constrained.hasAllocated = true

	misplacedChunks := make([]int, 0, len(loc.Registers))
	for regIdx, reg := range loc.Registers {
		constraint := constrained.constraint.Registers[regIdx]
		if constraint.SatisfyBy(reg) {
			scheduler.ReserveSource(reg, constraint)
		} else {
			misplacedChunks = append(misplacedChunks, regIdx)
		}
	}

	if len(misplacedChunks) > 0 {
		constrained.misplacedChunks = misplacedChunks
		return true
	}
	return false
}

func (scheduler *operationsScheduler) initSourceRegisterLocations(
	alloc *defAlloc,
	misplaced []*constrainedLocation,
) []*constrainedLocation {
	constrained := alloc.RegisterLocations()
	if len(constrained) == 0 { // no required register locations
		return nil
	}

	constrained, candidates, numAssigned, misplaced := scheduler.initCompatible(
		constrained,
		scheduler.ValueLocations.RegisterLocations(alloc.definition),
		misplaced)
	if len(constrained) == 0 {
		return misplaced
	}

	maxCopies := alloc.numActual
	if alloc.numPreferred > maxCopies {
		maxCopies = alloc.numPreferred
	}

	numExtras := (alloc.numRequired + len(candidates)) - maxCopies
	for i := 0; i < numExtras; i++ {
		candidate := candidates[0]
		candidates = candidates[1:]
		if numAssigned == 0 &&
			len(constrained) > 0 &&
			alloc.numRequired == alloc.numPreferred {

			// Ensure at least one copy of register location survive to speed up
			// location copying.
			loc := constrained[0]
			constrained = constrained[1:]

			scheduler.assignAllocatedLocation(loc, candidate)
			misplaced = append(misplaced, loc)
			numAssigned++
		} else {
			scheduler.FreeLocation(candidate)
			alloc.numActual--
		}
	}

	for _, loc := range constrained {
		scheduler.assignUnallocatedLocation(alloc, loc)
		misplaced = append(misplaced, loc)
	}

	return misplaced
}

func (scheduler *operationsScheduler) initCompatible(
	constrained []*constrainedLocation,
	candidates []*arch.DataLocation,
	misplaced []*constrainedLocation,
) (
	[]*constrainedLocation,
	[]*arch.DataLocation,
	int,
	[]*constrainedLocation,
) {
	if len(candidates) == 0 {
		return constrained, candidates, 0, misplaced
	}

	type matIdx struct {
		*constrainedLocation
		*arch.DataLocation
	}

	compatibilities := make(map[matIdx]int, len(constrained)*len(candidates))
	for _, constrained := range constrained {
		for _, candidate := range candidates {
			compatibility := 0
			for regIdx, reg := range candidate.Registers {
				constraint := constrained.constraint.Registers[regIdx]
				if constraint.SatisfyBy(reg) {
					compatibility++
				}
			}
			if compatibility > 0 {
				compatibilities[matIdx{constrained, candidate}] = compatibility
			}
		}
	}

	numAssigned := 0
	for len(constrained) > 0 {
		if len(candidates) == 0 {
			break
		}

		bestCompatibility := 0
		bestConstrainedIdx := -1
		bestCandidateIdx := -1
		for constrainedIdx, constrained := range constrained {
			if constrained == nil {
				continue
			}
			for candidateIdx, candidate := range candidates {
				if candidate == nil {
					continue
				}
				compatibility := compatibilities[matIdx{constrained, candidate}]
				if compatibility > bestCompatibility {
					bestCompatibility = compatibility
					bestConstrainedIdx = constrainedIdx
					bestCandidateIdx = candidateIdx
				}
			}
		}

		if bestCompatibility < 1 {
			break
		}

		loc := constrained[bestConstrainedIdx]
		candidate := candidates[bestCandidateIdx]
		numAssigned++

		if bestCandidateIdx > 0 {
			candidates[bestCandidateIdx] = candidates[0]
		}
		candidates = candidates[1:]

		if bestConstrainedIdx > 0 {
			constrained[bestConstrainedIdx] = constrained[0]
		}
		constrained = constrained[1:]

		if scheduler.assignAllocatedLocation(loc, candidate) {
			misplaced = append(misplaced, loc)
		}
	}

	return constrained, candidates, numAssigned, misplaced
}

func (scheduler *operationsScheduler) reduceMisplacedWithoutEvication(
	misplaced []*constrainedLocation,
) (
	[]*constrainedLocation,
	bool,
) {
	updatedMisplaced := make([]*constrainedLocation, 0, len(misplaced))
	selectableChanged := false
	for _, loc := range misplaced {
		misplacedRegs := make([]int, 0, len(loc.misplacedChunks))
		for _, idx := range loc.misplacedChunks {
			constraint := loc.constraint.Registers[idx]
			existingReg := loc.loc.Registers[idx]

			// NOTE: We need to recheck the existing register since register
			// eviction may have replaced a previous register with a register that
			// satisfy the constraint.
			if loc.hasAllocated && constraint.SatisfyBy(existingReg) {
				scheduler.ReserveSource(existingReg, constraint)
				continue
			}

			selected := scheduler.SelectSourceRegister(constraint, true)
			if selected == nil {
				misplacedRegs = append(misplacedRegs, idx)
				continue
			}

			if loc.hasAllocated {
				scheduler.MoveRegister(existingReg, selected)
				selectableChanged = true
			} else {
				loc.loc.Registers[idx] = selected
			}
		}

		loc.misplacedChunks = misplacedRegs
		if len(misplacedRegs) == 0 {
			scheduler.finalizeSourceRegistersLocation(loc)
		} else {
			updatedMisplaced = append(updatedMisplaced, loc)
		}
	}

	return updatedMisplaced, selectableChanged
}

func (scheduler *operationsScheduler) finalizeSourceRegistersLocation(
	loc *constrainedLocation,
) {
	if loc.hasAllocated {
		return
	}

	var src *arch.DataLocation
	if loc.pseudoDefVal == nil {
		src = scheduler.selectCopySourceLocation(loc.def)
	}

	loc.loc = scheduler.AllocateRegistersLocation(
		loc.def,
		loc.loc.Registers...)

	if loc.pseudoDefVal == nil {
		scheduler.CopyLocation(src, loc.loc, nil)
	} else {
		scheduler.SetConstantValue(loc.pseudoDefVal, loc.loc, nil)
	}
	loc.hasAllocated = true
}

func (scheduler *operationsScheduler) setUpFinalDestination(alloc *defAlloc) {
	finalDestLoc := &arch.DataLocation{}
	scheduler.finalDest.loc = finalDestLoc

	destLocConst := scheduler.constraints.Destination
	if destLocConst.AnyLocation || destLocConst.RequireOnStack {
		if alloc.numPreferred == 0 {
			finalDestLoc.OnFixedStack = true
			scheduler.destScratchRegister = scheduler.SelectScratch()
		} else {
			for i := 0; i < alloc.numRegisters; i++ {
				finalDestLoc.Registers = append(
					finalDestLoc.Registers,
					scheduler.SelectFreeRegister())
			}
		}
	} else {
		for _, regConst := range destLocConst.Registers {
			finalDestLoc.Registers = append(
				finalDestLoc.Registers,
				scheduler.SelectDestinationRegister(regConst))
		}
	}
}

func (scheduler *operationsScheduler) executeInstruction() {
	destLoc := scheduler.finalDest.loc
	if scheduler.tempDest.loc != nil {
		destLoc = scheduler.tempDest.loc
	}

	scheduler.BlockState.ExecuteInstruction(
		scheduler.instruction,
		scheduler.srcs,
		destLoc)
}

func (scheduler *operationsScheduler) tearDownInstruction(
	mustCopyTempDest bool,
) {
	// Free all clobbered source locations
	for _, src := range scheduler.srcs {
		if src.constraint.ClobberedByInstruction() {
			scheduler.FreeLocation(src.loc)
		}
	}

	// Free all dead definitions.
	for def, locs := range scheduler.ValueLocations.Values {
		if def == scheduler.tempDest.def {
			// We need to copy the value to final destination before freeing.
			continue
		}

		liveRange, ok := scheduler.LiveRanges[def]
		if ok && len(liveRange.NextUses) > 0 {
			continue
		}

		for _, loc := range locs {
			scheduler.FreeLocation(loc)
		}
	}

	if scheduler.finalDest.def == nil {
		return
	}

	if ast.IsTerminal(scheduler.instruction) {
		if scheduler.tempDest.loc != nil {
			scheduler.FreeLocation(scheduler.tempDest.loc)
		}

		return
	}

	if scheduler.finalDest.loc == nil {
		// If the destination definition is not used, finalDest could be nil (but
		// only if the value is on temp stack).
		if scheduler.tempDest.loc == nil {
			panic("should never happen")
		}
	} else if scheduler.finalDest.loc.OnTempStack {
		panic("should never happen")
	} else if scheduler.finalDest.loc.OnFixedStack {
		// value must be on temp stack
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
		if mustCopyTempDest {
			scheduler.CopyLocation(
				scheduler.tempDest.loc,
				scheduler.finalDest.loc,
				scheduler.destScratchRegister)
		}
		scheduler.FreeLocation(scheduler.tempDest.loc)
	}
}

func (scheduler *operationsScheduler) selectCopySourceLocation(
	def *ast.VariableDefinition,
) *arch.DataLocation {
	locs, ok := scheduler.ValueLocations.Values[def]
	if !ok {
		panic("should never happen. missing definition: " + def.Name)
	}

	var selected *arch.DataLocation
	for _, loc := range locs {
		// Prefer register locations over stack location
		selected = loc
		if !selected.OnFixedStack && !selected.OnTempStack {
			break
		}
	}

	if selected == nil {
		panic("should never happen")
	}
	return selected
}

// Free preferences (from highest to lowest):
//   - most discardable extra copies (does not spill)
//   - candidate is already on fixed stack (already spill)
//   - callee-saved parameters (spill / use exactly once)
//   - spill largest (numRegisters * nextUseDelta).  This heuristic balances
//     between keeping more small values on registers, and keeping register
//     pressure low for longer (but could cause instruction pipeline stall).
//
// All preferences are tie break by name to make the selection deterministic.
func (scheduler *operationsScheduler) selectFreeDefCandidate(
	candidates []*defAlloc,
) (
	int,
	*defAlloc,
) {
	// Prefer to free candidate with extra discardable copies
	maxExtraCopies := 0
	selectedIdx := -1
	var selected *defAlloc
	for idx, candidate := range candidates {
		if candidate.numPreferred >= candidate.numActual {
			// The candidate cannot be unconditionally discarded
			continue
		}

		extraCopies := candidate.numActual - candidate.numClobbered
		if extraCopies > maxExtraCopies {
			maxExtraCopies = extraCopies
			selectedIdx = idx
			selected = candidate
		}
	}
	if selected != nil {
		return selectedIdx, selected
	}

	// Prefer to free candidate that are already on stack
	for idx, candidate := range candidates {
		if candidate.hasFixedStackCopy {
			return idx, candidate
		}
	}

	// Prefer callee-saved parameters
	for idx, candidate := range candidates {
		if strings.HasPrefix(candidate.definition.Name, "%") {
			return idx, candidate
		}
	}

	// Prefer largest (numRegisters * nextUseDelta)
	// TODO: improve heuristic
	largestHeuristic := -1
	for idx, candidate := range candidates {
		heuristic := candidate.numRegisters * candidate.nextUseDelta
		if heuristic > largestHeuristic {
			largestHeuristic = heuristic
			selectedIdx = idx
			selected = candidate
		}
	}

	if selected == nil {
		panic("should never happen")
	}
	return selectedIdx, selected
}

func (scheduler *operationsScheduler) selectFreeLocation(
	candidate *defAlloc,
) *arch.DataLocation {
	worstMatch := math.MaxInt32
	var selected *arch.DataLocation
	for _, loc := range scheduler.ValueLocations.Values[candidate.definition] {
		if loc.OnFixedStack || loc.OnTempStack {
			continue
		}

		match := 0
		for _, src := range candidate.constrained {
			numSatisfied := 0
			for idx, reg := range loc.Registers {
				if src.constraint.Registers[idx].SatisfyBy(reg) {
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
		}
	}

	if selected == nil {
		panic("should never happen")
	}
	return selected
}

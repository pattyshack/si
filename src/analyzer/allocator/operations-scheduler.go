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
	*allocation

	location *arch.DataLocation

	constraint      *arch.LocationConstraint
	misplacedChunks []int

	// When false, location has not been allocated, and location.Registers is
	// partially reserved and may include nils.  When true, location is allocated
	// and correctly initialized (misplacedChunks is empty)
	hasAllocated bool
}

type allocation struct {
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

	requireOnTempStack bool

	numActual         int
	hasFixedStackCopy bool

	immediate ast.Value // all immediates including label references

	setFramePointer bool
	evictOnly       bool
}

func (alloc *allocation) RegisterLocations() []*constrainedLocation {
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
	tempDest *constrainedLocation

	// Note: the set up phase will create a placeholder destination location
	// entry which won't be allocated until the tear down phase.
	finalDest *constrainedLocation
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

func (scheduler *operationsScheduler) ScheduleTransferBlockOperations(
	childLocationIn LocationSet,
) {
	allocs := scheduler.initializeTransferBlockConstraints(childLocationIn)

	// Spill definition to fixed stack and free all extra register location
	// copies.  Note that extra fixed stack location can be safely kept around;
	// the child block will simply ignore the location.
	for _, alloc := range allocs {
		locs := scheduler.ValueLocations.Values[alloc.definition]
		if alloc.numRequired == 0 {
			if !alloc.hasFixedStackCopy {
				src := scheduler.selectCopySourceLocation(alloc.definition)
				dest := scheduler.AllocateFixedStackLocation(alloc.definition)
				scheduler.CopyLocation(src, dest, nil)
				alloc.hasFixedStackCopy = true
			}

			for _, loc := range locs {
				if loc.OnFixedStack {
					continue
				}
				scheduler.FreeLocation(loc)
			}
		} else if alloc.numActual > 1 {
			constraints := alloc.constrained[0].constraint.Registers

			bestNumMatches := -1
			var bestLoc *arch.DataLocation
			for _, loc := range locs {
				if loc.OnFixedStack {
					continue
				}

				numMatches := 0
				for idx, reg := range loc.Registers {
					if constraints[idx].SatisfyBy(reg) {
						numMatches++
					}
				}

				if numMatches > bestNumMatches {
					bestNumMatches = numMatches
					bestLoc = loc
				}
			}

			for _, loc := range locs {
				if loc != bestLoc {
					scheduler.FreeLocation(loc)
				}
			}
		}
	}

	scheduler.setUpRegisters(allocs)
}

func (scheduler *operationsScheduler) getValueLocationAllocs(
	nextUseDelta func(*ast.VariableDefinition) int,
) map[*ast.VariableDefinition]*allocation {
	mappedAllocs := make(
		map[*ast.VariableDefinition]*allocation,
		len(scheduler.ValueLocations.Values))
	for def, locs := range scheduler.ValueLocations.Values {
		alloc := &allocation{
			definition:   def,
			numRegisters: arch.NumRegisters(def.Type),
			nextUseDelta: nextUseDelta(def),
		}
		mappedAllocs[def] = alloc

		for _, loc := range locs {
			if loc.OnFixedStack {
				alloc.hasFixedStackCopy = true
			} else if loc.OnTempStack {
				panic("should never happen")
			} else {
				alloc.numActual++
			}
		}
	}

	return mappedAllocs
}

func (scheduler *operationsScheduler) sortedAllocs(
	mappedAllocs map[*ast.VariableDefinition]*allocation,
) []*allocation {
	allocs := make([]*allocation, 0, len(mappedAllocs))
	for _, alloc := range mappedAllocs {
		allocs = append(allocs, alloc)
	}
	sort.Slice(
		allocs,
		func(i int, j int) bool {
			return arch.CompareDefinitionNames(
				allocs[i].definition.Name,
				allocs[j].definition.Name) < 0
		})

	return allocs
}

func (scheduler *operationsScheduler) initializeTransferBlockConstraints(
	childLocationIn LocationSet,
) []*allocation {
	mappedAllocs := scheduler.getValueLocationAllocs(
		func(*ast.VariableDefinition) int { return 1 })

	for def, loc := range childLocationIn {
		alloc := mappedAllocs[def]

		if loc.OnFixedStack {
			// do nothing
		} else if loc.OnTempStack {
			panic("should never happen")
		} else {
			regConstraints := make([]*arch.RegisterConstraint, len(loc.Registers))
			for idx, reg := range loc.Registers {
				scheduler.ExactMatch(reg)
				regConstraints[idx] = &arch.RegisterConstraint{
					Require: reg,
				}
			}

			alloc.constrained = []*constrainedLocation{
				&constrainedLocation{
					allocation: alloc,
					constraint: &arch.LocationConstraint{
						NumRegisters: alloc.numRegisters,
						Registers:    regConstraints,
					},
				},
			}
			alloc.numPreferred = 1
			alloc.numRequired = 1
		}
	}

	return scheduler.sortedAllocs(mappedAllocs)
}

func (scheduler *operationsScheduler) ScheduleInstructionOperations(
	instruction ast.Instruction,
	constraints *arch.InstructionConstraints,
) {
	pressure, allocations := scheduler.initializeInstructionConstraints(
		instruction,
		constraints)

	_, ok := scheduler.instruction.(*ast.CopyOperation)
	if ok {
		scheduler.scheduleCopy(pressure, allocations)
		return
	}

	scratchRegister := scheduler.SelectScratch()
	scheduler.setUpTempStack(scratchRegister)

	// Note: To maximize degrees of freedom / simplify accounting, register
	// pressure is computed after setting up temp stack, which will likely free
	// up some registers.
	scheduler.reduceRegisterPressure(pressure, allocations, scratchRegister)

	// Release scratch register before register selection since it may be picked
	// by the instruction.
	scheduler.ReleaseScratch(scratchRegister)

	scheduler.setUpRegisters(allocations)

	scheduler.executeInstruction()

	scheduler.freeDeadSources()
	scheduler.tearDownInstruction()
}

func (scheduler *operationsScheduler) nextUseDelta(
	def *ast.VariableDefinition,
) int {
	return scheduler.NextUseDelta(scheduler.currentDist, def)
}

func (scheduler *operationsScheduler) initializeInstructionConstraints(
	instruction ast.Instruction,
	constraints *arch.InstructionConstraints,
) (
	int,
	[]*allocation,
) {
	mappedSrcAllocs := scheduler.getValueLocationAllocs(scheduler.nextUseDelta)
	srcs := scheduler.generateConstrainedSources(
		mappedSrcAllocs,
		instruction,
		constraints)
	scheduler.combineConstrainedSources(mappedSrcAllocs, srcs)

	scheduler.instruction = instruction
	scheduler.constraints = constraints
	scheduler.srcs = srcs

	nonSrcExacts, overlappedWildcards, overlappableWildcards := scheduler.
		collectRegisterConstraintStats(srcs, constraints)

	scheduler.generateConstrainedPseudoExactSource(
		mappedSrcAllocs,
		instruction,
		nonSrcExacts)

	if constraints.Destination == nil {
		// do nothing
	} else {
		destDef := instruction.Destination()
		finalDestAlloc := &allocation{
			definition:   destDef,
			numRegisters: constraints.Destination.NumRegisters,
			nextUseDelta: scheduler.nextUseDelta(destDef),
		}

		scheduler.finalDest = &constrainedLocation{
			allocation: finalDestAlloc,
			constraint: constraints.Destination,
		}
		finalDestAlloc.constrained = append(
			finalDestAlloc.constrained,
			scheduler.finalDest)

		if constraints.Destination.AnyLocation {
			// already correctly setup
		} else if constraints.Destination.RequireOnStack {
			scheduler.setUpConstrainedStackDestination()
		} else { // register location
			nonSrcWildcards := scheduler.setUpConstrainedRegisterDestination(
				overlappedWildcards,
				overlappableWildcards)
			scheduler.generateConstrainedPseudoWildcardSource(
				mappedSrcAllocs,
				instruction,
				nonSrcWildcards)
		}
	}

	registerPressure := 0
	for _, alloc := range mappedSrcAllocs {
		for _, loc := range alloc.constrained {
			if loc.constraint.AnyLocation { // only copy op uses AnyLocation for src
				loc.numPreferred++
			} else if loc.constraint.RequireOnStack {
				loc.requireOnTempStack = true
			} else {
				loc.numPreferred++
				loc.numRequired++

				if loc.constraint.ClobberedByInstruction() {
					loc.numClobbered++
				}
			}
		}

		// If the destination out lives the instruction, ensure at least one copy
		// of the source definition survive.
		if alloc.nextUseDelta > 0 &&
			alloc.numPreferred == alloc.numRequired &&
			alloc.numRequired == alloc.numClobbered {

			// Ensure we don't eagerly load value back onto register if the value
			// only exist on stack and is not used by the instruction.
			if !alloc.hasFixedStackCopy ||
				alloc.numActual > 0 ||
				alloc.numRequired > 0 ||
				alloc.requireOnTempStack {

				alloc.numPreferred++
			}
		}

		numCopies := alloc.numActual
		if alloc.numPreferred > alloc.numActual {
			numCopies = alloc.numPreferred
		}

		registerPressure += numCopies * alloc.numRegisters
	}

	return registerPressure, scheduler.sortedAllocs(mappedSrcAllocs)
}

func (scheduler *operationsScheduler) generateConstrainedSources(
	mappedSrcAllocs map[*ast.VariableDefinition]*allocation,
	instruction ast.Instruction,
	constraints *arch.InstructionConstraints,
) []*constrainedLocation {
	srcValues := instruction.Sources()
	srcs := make([]*constrainedLocation, 0, len(srcValues))
	for idx, value := range srcValues {
		loc := &constrainedLocation{
			constraint: constraints.Sources[idx],
		}

		var alloc *allocation
		ref, ok := value.(*ast.VariableReference)
		if ok {
			alloc, ok = mappedSrcAllocs[ref.UseDef]
			if !ok {
				panic("should never happen")
			}
		} else { // immediates
			def := &ast.VariableDefinition{
				StartEndPos:       value.StartEnd(),
				Name:              fmt.Sprintf("%%constant-source-%d", idx),
				Type:              value.Type(),
				ParentInstruction: instruction,
			}

			alloc = &allocation{
				definition: def,
				immediate:  value,
			}
			mappedSrcAllocs[def] = alloc

			if loc.constraint.SupportEncodedImmediate &&
				scheduler.CanEncodeImmediate(value) {

				if loc.constraint.ClobberedByInstruction() { // sanity check
					panic("should never happen")
				}

				loc.location = &arch.DataLocation{
					Name:             def.Name,
					Type:             def.Type,
					EncodedImmediate: value,
				}
				loc.hasAllocated = true
			} else {
				alloc.numRegisters = arch.NumRegisters(def.Type)
			}
		}

		loc.allocation = alloc
		srcs = append(srcs, loc)
		alloc.constrained = append(alloc.constrained, loc)
	}

	if constraints.FramePointerRegister != nil {
		// TODO: populate type
		def := &ast.VariableDefinition{
			StartEndPos:       instruction.StartEnd(),
			Name:              arch.CurrentFramePointer,
			ParentInstruction: instruction,
		}

		alloc := &allocation{
			definition:      def,
			numRegisters:    1,
			setFramePointer: true,
		}
		mappedSrcAllocs[def] = alloc

		loc := &constrainedLocation{
			allocation: alloc,
			constraint: &arch.LocationConstraint{
				NumRegisters: 1,
				Registers: []*arch.RegisterConstraint{
					&arch.RegisterConstraint{
						Require: constraints.FramePointerRegister,
					},
				},
			},
		}
		alloc.constrained = []*constrainedLocation{loc}
	}

	return srcs
}

func (scheduler *operationsScheduler) combineConstrainedSources(
	mappedAllocs map[*ast.VariableDefinition]*allocation,
	srcs []*constrainedLocation,
) {
	substitution := map[*constrainedLocation]*constrainedLocation{}
	for _, alloc := range mappedAllocs {
		if len(alloc.constrained) < 2 { // nothing to merge
			continue
		}

		// Non-clobbered wildcard location constraints are mergable candidates.
		// All other location constraints are unmergable.
		selected := make([]*constrainedLocation, 0, len(alloc.constrained))
		candidates := make([]*constrainedLocation, 0, len(alloc.constrained))
		for _, loc := range alloc.constrained {
			if len(loc.constraint.Registers) == 0 ||
				loc.constraint.Registers[0].Require != nil ||
				loc.constraint.Registers[0].Clobbered {

				selected = append(selected, loc)
			} else {
				candidates = append(candidates, loc)
			}
		}

		for _, candidate := range candidates {
			var replacement *constrainedLocation
			for _, loc := range selected {
				if len(loc.constraint.Registers) == 0 {
					continue // stack location or unit data type
				}

				mergable := true
				for idx, locReg := range loc.constraint.Registers {
					candidateReg := candidate.constraint.Registers[idx]
					if locReg.Require != nil {
						if !candidateReg.SatisfyBy(locReg.Require) {
							mergable = false
							break
						}
					} else if !(candidateReg.AnyGeneral == locReg.AnyGeneral &&
						candidateReg.AnyFloat == locReg.AnyFloat) {
						// Only allow identical wildcard conditions to merge
						mergable = false
						break
					}
				}

				if mergable {
					replacement = loc
					break
				}
			}

			if replacement != nil {
				substitution[candidate] = replacement
			} else {
				selected = append(selected, candidate)
			}
		}

		alloc.constrained = selected
	}

	for idx, src := range srcs {
		replacement, ok := substitution[src]
		if ok {
			srcs[idx] = replacement
		}
	}
}

func (scheduler *operationsScheduler) collectRegisterConstraintStats(
	srcs []*constrainedLocation,
	constraints *arch.InstructionConstraints,
) (
	map[*arch.Register]struct{}, // non source clobbered registers
	map[*arch.RegisterConstraint]struct{}, // overlapped wildcard
	[]*arch.RegisterConstraint, // overlappable wildcard candidate
) {
	nonSrcExacts := map[*arch.Register]struct{}{}
	for reg, clobbered := range constraints.RequiredRegisters {
		if clobbered {
			nonSrcExacts[reg] = struct{}{}
		}
	}

	destRegs := map[*arch.RegisterConstraint]struct{}{}
	if constraints.Destination != nil {
		for _, reg := range constraints.Destination.Registers {
			destRegs[reg] = struct{}{}
		}
	}

	srcRegs := map[*arch.RegisterConstraint]struct{}{}
	overlappedWildcards := map[*arch.RegisterConstraint]struct{}{}
	overlappableWildcards := []*arch.RegisterConstraint{}
	for _, src := range srcs {
		for _, reg := range src.constraint.Registers {
			_, ok := srcRegs[reg]
			if ok { // location reused by multiple src values
				continue
			}
			srcRegs[reg] = struct{}{}

			if reg.Require != nil {
				delete(nonSrcExacts, reg.Require)
				continue
			}

			// reg is a wildcard match
			_, ok = destRegs[reg]
			if ok { // already overlapped
				overlappedWildcards[reg] = struct{}{}
			} else if reg.Clobbered || src.nextUseDelta == 0 {
				overlappableWildcards = append(overlappableWildcards, reg)
			}
		}
	}

	return nonSrcExacts, overlappedWildcards, overlappableWildcards
}

// Create a pseudo allocation source entry for exact match clobbered registers
// that aren't part of src constraints (i.e., scratch registers and
// non-overlapped destination registers).  These registers must be evicted
// before instruction execution.
func (scheduler *operationsScheduler) generateConstrainedPseudoExactSource(
	mappedSrcAllocs map[*ast.VariableDefinition]*allocation,
	instruction ast.Instruction,
	nonSrcExacts map[*arch.Register]struct{},
) {
	if len(nonSrcExacts) == 0 {
		return
	}

	// TODO: populate type
	def := &ast.VariableDefinition{
		StartEndPos:       instruction.StartEnd(),
		Name:              "%exact-non-source",
		ParentInstruction: instruction,
	}

	alloc := &allocation{
		definition:   def,
		numRegisters: len(nonSrcExacts),
		evictOnly:    true,
	}
	mappedSrcAllocs[def] = alloc

	registers := []*arch.RegisterConstraint{}
	for _, regInfo := range scheduler.ValueLocations.Registers {
		_, ok := nonSrcExacts[regInfo.Register]
		if !ok {
			continue
		}

		registers = append(
			registers,
			&arch.RegisterConstraint{
				Require:   regInfo.Register,
				Clobbered: true,
			})
	}

	loc := &constrainedLocation{
		allocation: alloc,
		constraint: &arch.LocationConstraint{
			NumRegisters: len(nonSrcExacts),
			Registers:    registers,
		},
	}
	alloc.constrained = []*constrainedLocation{loc}
}

// Create a pseudo allocation source entry for wildcard match clobbered
// registers that	aren't part of src constraints. These registers must be
// evicted before instruction execution.
func (scheduler *operationsScheduler) generateConstrainedPseudoWildcardSource(
	mappedSrcAllocs map[*ast.VariableDefinition]*allocation,
	instruction ast.Instruction,
	nonSrcWildcards []*arch.RegisterConstraint,
) {
	if len(nonSrcWildcards) == 0 {
		return
	}

	// TODO: populate type
	def := &ast.VariableDefinition{
		StartEndPos:       instruction.StartEnd(),
		Name:              "%wildcard-non-source",
		ParentInstruction: instruction,
	}

	alloc := &allocation{
		definition:   def,
		numRegisters: len(nonSrcWildcards),
		evictOnly:    true,
	}
	mappedSrcAllocs[def] = alloc

	loc := &constrainedLocation{
		allocation: alloc,
		constraint: &arch.LocationConstraint{
			NumRegisters: len(nonSrcWildcards),
			Registers:    nonSrcWildcards,
		},
	}
	alloc.constrained = []*constrainedLocation{loc}
}

func (scheduler *operationsScheduler) setUpConstrainedStackDestination() {
	tempDestDef := &ast.VariableDefinition{
		StartEndPos:       scheduler.finalDest.definition.StartEnd(),
		Name:              "%temp-destination",
		Type:              scheduler.finalDest.definition.Type,
		ParentInstruction: scheduler.finalDest.definition.ParentInstruction,
	}

	tempDestAlloc := &allocation{
		definition:   tempDestDef,
		numRegisters: scheduler.finalDest.numRegisters,
	}

	scheduler.tempDest = &constrainedLocation{
		allocation: tempDestAlloc,
		constraint: scheduler.finalDest.constraint,
	}
	tempDestAlloc.constrained = append(
		tempDestAlloc.constrained,
		scheduler.tempDest)

	scheduler.finalDest.constraint = &arch.LocationConstraint{
		NumRegisters: scheduler.finalDest.numRegisters,
		AnyLocation:  true,
	}
}

func (scheduler *operationsScheduler) setUpConstrainedRegisterDestination(
	overlappedWildcards map[*arch.RegisterConstraint]struct{},
	overlappableWildcards []*arch.RegisterConstraint,
) []*arch.RegisterConstraint {
	unmodifiedConstraint := scheduler.finalDest.constraint
	registers := []*arch.RegisterConstraint{}
	nonSrcWildcards := []*arch.RegisterConstraint{}
	for _, reg := range unmodifiedConstraint.Registers {
		if reg.Require != nil {
			registers = append(registers, reg)
			continue
		}

		_, ok := overlappedWildcards[reg]
		if ok {
			registers = append(registers, reg)
			continue
		}

		foundOverlap := false
		for idx, candidate := range overlappableWildcards {
			if candidate == nil {
				continue
			}

			if candidate.AnyGeneral == reg.AnyGeneral &&
				candidate.AnyFloat == reg.AnyFloat {

				foundOverlap = true
				registers = append(registers, candidate)
				overlappableWildcards[idx] = nil
				break
			}
		}

		if !foundOverlap {
			registers = append(registers, reg)
			nonSrcWildcards = append(nonSrcWildcards, reg)
		}
	}

	scheduler.finalDest.constraint = &arch.LocationConstraint{
		NumRegisters: unmodifiedConstraint.NumRegisters,
		Registers:    registers,
	}

	return nonSrcWildcards
}

func (scheduler *operationsScheduler) scheduleCopy(
	pressure int,
	allocations []*allocation,
) {
	defer scheduler.freeDeadSources()

	if len(scheduler.srcs) != 1 {
		panic("should never happen")
	}
	src := scheduler.srcs[0]
	scheduler.srcs = nil // ignore srcs during register clean up

	// If the dest defintion is never used, we only need to clean up dead
	// source definition
	if scheduler.finalDest.nextUseDelta == 0 {
		return
	}

	srcDef := src.definition
	destDef := scheduler.finalDest.definition
	if src.nextUseDelta == 0 {
		// Src is dead after this instruction.  When possible, transfer all
		// available
		if srcDef.Name == destDef.Name || src.numActual > 0 {
			scheduler.transferLocations(srcDef, destDef, false)
			return
		}
	} else {
		// Src remains alive after this instruction, but there are multiple register
		// copies of src. Just transfer one of the register copies to dest.
		if src.numActual > 1 {
			scheduler.transferLocations(srcDef, destDef, true)
			return
		}
	}

	scratch := scheduler.allocateAnyDestination()

	if src.immediate != nil {
		scheduler.SetConstantValue(
			src.immediate,
			scheduler.finalDest.location,
			scratch)
	} else {
		scheduler.CopyLocation(
			scheduler.selectCopySourceLocation(srcDef),
			scheduler.finalDest.location,
			scratch)
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

		srcDef := src.definition
		tempStackSrcs = append(tempStackSrcs, src)
		srcDefs = append(srcDefs, srcDef)

		if src.immediate == nil {
			copySrcs[srcDef] = scheduler.selectCopySourceLocation(srcDef)
		}
	}

	var tempDestDef *ast.VariableDefinition
	if scheduler.tempDest != nil {
		tempDestDef = scheduler.tempDest.definition
	}

	stackSrcLocs, tempDestLoc := scheduler.AllocateTempStackLocations(
		srcDefs,
		tempDestDef)

	for idx, src := range tempStackSrcs {
		loc := stackSrcLocs[idx]
		if src.immediate != nil {
			scheduler.SetConstantValue(src.immediate, loc, scratchRegister)
		} else {
			copySrc, ok := copySrcs[src.definition]
			if !ok {
				panic("should never happen")
			}
			scheduler.CopyLocation(copySrc, loc, scratchRegister)
		}

		src.location = loc
		src.hasAllocated = true
	}

	if tempDestLoc != nil {
		scheduler.tempDest.location = tempDestLoc
		scheduler.InitializeZeros(tempDestLoc, scratchRegister)
		scheduler.tempDest.hasAllocated = true
	}
}

func (scheduler *operationsScheduler) reduceRegisterPressure(
	pressure int,
	allocations []*allocation,
	scratchRegister *arch.Register,
) {
	candidates := make([]*allocation, 0, len(allocations))
	for _, alloc := range allocations {
		if alloc.numRegisters == 0 { // unit data type takes up no space
			continue
		}

		if alloc.numRequired == alloc.numPreferred &&
			alloc.numRequired >= alloc.numActual {
			continue // we can't free any more copies of this definition
		}

		candidates = append(candidates, alloc)
	}

	var finalDestDef *ast.VariableDefinition
	if scheduler.finalDest != nil {
		finalDestDef = scheduler.finalDest.definition
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
			if candidate.definition != finalDestDef {
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
	alloc *allocation,
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
	allocs []*allocation,
) {
	misplaced := make([]*constrainedLocation, 0, len(scheduler.srcs))
	for _, alloc := range allocs {
		if len(alloc.constrained) == 0 {
			continue
		} else if alloc.immediate != nil ||
			alloc.evictOnly ||
			alloc.setFramePointer {

			for _, loc := range alloc.RegisterLocations() {
				if loc.hasAllocated {
					continue
				}

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
			scheduler.MoveRegister(loc.location.Registers[regIdx], selected)
		} else {
			loc.location.Registers[regIdx] = selected
		}

		if len(loc.misplacedChunks) == 0 {
			scheduler.finalizeSourceRegistersLocation(loc)
			misplaced = misplaced[1:]
		}
	}

	// Create unconstrained preferred copies that only serve to keep definitions
	// alive.
	//
	// NOTE: We must finalize destination before creating preferred copies since
	// the destination may have specific register constraints.
	for _, alloc := range allocs {
		if alloc.immediate != nil || alloc.evictOnly || alloc.setFramePointer {
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
	alloc *allocation,
	constrained *constrainedLocation,
) {
	if constrained.location != nil {
		panic("should never happen")
	}

	misplacedChunks := make([]int, alloc.numRegisters)
	for idx := 0; idx < alloc.numRegisters; idx++ {
		misplacedChunks[idx] = idx
	}
	constrained.misplacedChunks = misplacedChunks

	constrained.location = &arch.DataLocation{
		Registers: make([]*arch.Register, alloc.numRegisters),
	}

	alloc.numActual++
}

func (scheduler *operationsScheduler) assignAllocatedLocation(
	constrained *constrainedLocation,
	loc *arch.DataLocation,
) bool {
	if constrained.location != nil {
		panic("should never happen")
	}

	constrained.location = loc
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
	alloc *allocation,
	misplaced []*constrainedLocation,
) []*constrainedLocation {
	constrained := alloc.RegisterLocations()
	if len(constrained) == 0 { // no required register locations
		return misplaced
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
			existingReg := loc.location.Registers[idx]

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
				loc.location.Registers[idx] = selected
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
	if loc.immediate == nil && !loc.setFramePointer && !loc.evictOnly {
		src = scheduler.selectCopySourceLocation(loc.definition)
	}

	loc.location = scheduler.AllocateRegistersLocation(
		loc.definition,
		loc.location.Registers...)

	if loc.immediate != nil {
		scheduler.SetConstantValue(loc.immediate, loc.location, nil)
	} else if loc.setFramePointer {
		scheduler.SetFramePointerAddress(loc.location)
	} else if loc.evictOnly {
		// do nothing
	} else {
		scheduler.CopyLocation(src, loc.location, nil)
	}

	loc.hasAllocated = true
}

func (scheduler *operationsScheduler) executeInstruction() {
	var destLoc *arch.DataLocation
	if scheduler.tempDest != nil {
		destLoc = scheduler.tempDest.location
	} else if scheduler.finalDest != nil {
		destLoc = scheduler.finalDest.location
	}

	scheduler.BlockState.ExecuteInstruction(
		scheduler.instruction,
		scheduler.srcs,
		destLoc)
}

func (scheduler *operationsScheduler) freeDeadSources() {
	// Free all clobbered source locations
	processed := map[*constrainedLocation]struct{}{}
	for _, src := range scheduler.srcs {
		_, ok := processed[src]
		if ok {
			continue
		}
		processed[src] = struct{}{}

		if src.constraint.ClobberedByInstruction() {
			scheduler.FreeLocation(src.location)
		}
	}

	// Free all dead definitions.
	for def, locs := range scheduler.ValueLocations.Values {
		if scheduler.tempDest != nil && def == scheduler.tempDest.definition {
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
}

func (scheduler *operationsScheduler) allocateAnyDestination() *arch.Register {
	numFreeRegisters := 0
	for _, regInfo := range scheduler.ValueLocations.Registers {
		if regInfo.UsedBy == nil {
			numFreeRegisters++
		}
	}

	if numFreeRegisters >= scheduler.finalDest.numRegisters+1 {
		// reset selector
		scheduler.RegisterSelector = NewRegisterSelector(scheduler.BlockState)

		registers := []*arch.Register{}
		for i := 0; i < scheduler.finalDest.numRegisters; i++ {
			registers = append(registers, scheduler.SelectFreeRegister())
		}

		scheduler.finalDest.location = scheduler.AllocateRegistersLocation(
			scheduler.finalDest.definition,
			registers...)

		return nil
	}

	scheduler.finalDest.location = scheduler.AllocateFixedStackLocation(
		scheduler.finalDest.definition)
	return scheduler.SelectScratch()
}

func (scheduler *operationsScheduler) tearDownInstruction() {
	if scheduler.finalDest == nil {
		return
	}

	if ast.IsTerminal(scheduler.instruction) ||
		scheduler.finalDest.nextUseDelta == 0 {

		if scheduler.tempDest != nil {
			scheduler.FreeLocation(scheduler.tempDest.location)
		}

		return
	}

	if scheduler.finalDest.constraint.RequireOnStack {
		panic("should never happen")
	} else if scheduler.finalDest.constraint.AnyLocation {
		if scheduler.tempDest == nil {
			panic("should never happen")
		}

		scratch := scheduler.allocateAnyDestination()
		scheduler.CopyLocation(
			scheduler.tempDest.location,
			scheduler.finalDest.location,
			scratch)
		scheduler.FreeLocation(scheduler.tempDest.location)
	} else {
		registers := []*arch.Register{}
		for _, regConst := range scheduler.finalDest.constraint.Registers {
			registers = append(
				registers,
				scheduler.SelectDestinationRegister(regConst))
		}

		scheduler.finalDest.location = scheduler.AllocateRegistersLocation(
			scheduler.finalDest.definition,
			registers...)
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
	candidates []*allocation,
) (
	int,
	*allocation,
) {
	// Prefer to free candidate with extra discardable copies
	maxExtraCopies := 0
	selectedIdx := -1
	var selected *allocation
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
	candidate *allocation,
) *arch.DataLocation {
	worstMatch := math.MaxInt32
	var selected *arch.DataLocation
	for _, loc := range scheduler.ValueLocations.Values[candidate.definition] {
		if loc.OnFixedStack || loc.OnTempStack {
			continue
		}

		match := 0
		for _, src := range candidate.constrained {
			if src.hasAllocated { // encoded immediate or temp stack location
				continue
			}

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

package allocator

import (
	"strings"

	arch "github.com/pattyshack/chickadee/architecture"
	"github.com/pattyshack/chickadee/ast"
)

type instructionAllocator struct {
	*BlockState

	instruction ast.Instruction
	constraints *arch.InstructionConstraints

	srcValues []ast.Value
	destDef   *ast.VariableDefinition

	srcConstLocs map[*arch.LocationConstraint]*arch.DataLocation

	// A temp stack destination allocated during the set up phase.
	tempDestLoc *arch.DataLocation

	// Note: the set up phase will create a placeholder destination location
	// entry which won't be allocated until the tear down phase.
	finalDestLoc *arch.DataLocation
}

func newInstructionAllocator(
	state *BlockState,
	inst ast.Instruction,
	constraints *arch.InstructionConstraints,
) *instructionAllocator {
	return &instructionAllocator{
		BlockState:   state,
		instruction:  inst,
		constraints:  constraints,
		srcValues:    inst.Sources(),
		destDef:      inst.Destination(),
		srcConstLocs: map[*arch.LocationConstraint]*arch.DataLocation{},
	}
}

func (allocator *instructionAllocator) setUpTempStack() {
	allocator.selectTempRegister()
}

func (allocator *instructionAllocator) reduceRegisterPressure() {
}

func (allocator *instructionAllocator) setUpRegisterSources() {
}

func (allocator *instructionAllocator) SetUpInstruction() {
	allocator.setUpTempStack()
	allocator.reduceRegisterPressure()
	allocator.setUpRegisterSources()
}

func (allocator *instructionAllocator) ExecuteInstruction() {
	_, ok := allocator.instruction.(*ast.CopyOperation)
	if ok {
		return // value already copied during the set up phase
	}

	srcLocs := []*arch.DataLocation{}
	for _, constraint := range allocator.constraints.Sources {
		loc, ok := allocator.srcConstLocs[constraint]
		if !ok {
			panic("should never happen")
		}
		srcLocs = append(srcLocs, loc)
	}

	destLoc := allocator.finalDestLoc
	if allocator.tempDestLoc != nil {
		destLoc = allocator.tempDestLoc
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

		loc, ok := allocator.srcConstLocs[constraint]
		if !ok {
			panic("should never happen")
		}
		allocator.FreeLocation(loc)
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

	if allocator.destDef == nil {
		return
	}

	if allocator.finalDestLoc == nil {
		panic("should never happen")
	} else if allocator.finalDestLoc.OnTempStack {
		panic("should never happen")
	} else if allocator.finalDestLoc.OnFixedStack {
		// value must be on temp tack
		if allocator.tempDestLoc == nil {
			panic("should never happen")
		}

		allocator.finalDestLoc = allocator.AllocateFixedStackLocation(
			allocator.destDef)
	} else {
		// Since sources and destination may share registers, register destination
		// must be allocated after all clobbered sources are freed.
		allocator.finalDestLoc = allocator.AllocateRegistersLocation(
			allocator.destDef,
			allocator.finalDestLoc.Registers...)
	}

	if allocator.tempDestLoc != nil {
		// The destination value is on a temp stack
		allocator.CopyLocation(
			allocator.tempDestLoc,
			allocator.finalDestLoc,
			tempRegister)
		allocator.FreeLocation(allocator.tempDestLoc)
	}
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

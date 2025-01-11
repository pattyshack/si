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
}

func newInstructionAllocator(
	state *BlockState,
	inst ast.Instruction,
	constraints *arch.InstructionConstraints,
) *instructionAllocator {
	return &instructionAllocator{
		BlockState:  state,
		instruction: inst,
		constraints: constraints,
		srcValues:   inst.Sources(),
		destDef:     inst.Destination(),
	}
}

func (allocator *instructionAllocator) SetUpInstruction() {
	allocator.selectTempRegister()
}

func (allocator *instructionAllocator) ExecuteInstruction() {
}

func (allocator *instructionAllocator) TearDownInstruction() {
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

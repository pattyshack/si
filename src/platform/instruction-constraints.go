package platform

import (
	"github.com/pattyshack/chickadee/ast"
)

// Where the data is located.  The data location is either a *RegisterSlot, or
// a *StackSlot.
//
// Assumptions: A value must either be completely on register, or completely
// on stack.
type DataLocation interface {
	isDataLocation()
}

// A yet to be determined register, selected from the candidates set.
type SelectedRegister struct {
	// Clobbered registers are caller-saved; registers are callee-saved otherwise.
	Clobbered  bool
	Candidates []*Register
}

type RegisterSlot struct {
	// The value is stored in an "array" formed by a list of registers.  The list
	// could be empty to indicate a zero-sized type (e.g., empty struct)
	Registers []*SelectedRegister
}

var _ DataLocation = &RegisterSlot{}

func (*RegisterSlot) isDataLocation() {}

type StackSlot struct {
	Type ast.Type
}

var _ DataLocation = &StackSlot{}

func (StackSlot) isDataLocation() {}

// InstructionConstraints is used to specify instruction register constraints,
// and call convention's registers selection / stack layout.
//
// Note:
// 1. it's safe to reuse the same constraints from multiple instructions.
// 2. do not manually modify the fields. Use the provided methods instead.
type InstructionConstraints struct {
	// The unordered set of registers used by this instruction.  Entries are
	// created by Select.  Note that this set may include registers not used by
	// sources and destination.
	UsedRegisters []*SelectedRegister

	// Which sources/destination values should be on stack.  The sources layout is
	// specified from top to bottom (stack destination is always at the bottom).
	// Note: The stack layout depends on AddStackSource calls order.
	//
	// Source value are copied into the stack slots, and destination's stack slot
	// is initialized to zeros.
	SourceSlots     []*StackSlot
	DestinationSlot *StackSlot // nil if the destination fits on registers

	FuncValue *SelectedRegister // only set by FuncCall
	// Source data locations are in the same order as the instruction's sources.
	Sources     []DataLocation
	Destination DataLocation // not set by control flow instructions

	// TODO: add
	//   ForceSpillToMemory(valueType ast.Type) (wrappedType ast.Type)
	// option for garbage collection.
	//
	// Callee-saved register's data type information is not preserved across
	// function call.  Hence, garbage collected language must either spill
	// everything onto memory (i.e., all registers are caller-saved), or
	// selectively spill objects with pointer data onto memory.
	//
	// The return wrappedType should encapulate the original valueType and
	// should include type information to be used by the garbage collector.
}

func NewInstructionConstraints() *InstructionConstraints {
	return &InstructionConstraints{}
}

func (constraints *InstructionConstraints) Select(
	clobbered bool,
	candidates ...*Register,
) *SelectedRegister {
	if len(candidates) == 0 {
		panic("empty candidate list")
	}
	reg := &SelectedRegister{
		Clobbered:  clobbered,
		Candidates: candidates,
	}
	constraints.UsedRegisters = append(constraints.UsedRegisters, reg)
	return reg
}

func (constraints *InstructionConstraints) listToLocation(
	list []*SelectedRegister,
	isDest bool,
) DataLocation {
	for _, entry := range list {
		if isDest {
			entry.Clobbered = true
		}
	}

	return &RegisterSlot{
		Registers: list,
	}
}

func (constraints *InstructionConstraints) SetFuncValue(
	register *SelectedRegister,
) {
	if constraints.FuncValue != nil {
		panic("func value already set")
	}
	constraints.FuncValue = register
}

// The data location list must either be a single stack slot entry, or a list
// of selected registers (the list could be empty if source value does not
// occupy any space, e.g., empty struct)
func (constraints *InstructionConstraints) AddRegisterSource(
	registers ...*SelectedRegister,
) {
	constraints.Sources = append(
		constraints.Sources,
		constraints.listToLocation(registers, false))
}

func (constraints *InstructionConstraints) AddStackSource(
	srcType ast.Type,
) {
	slot := &StackSlot{
		Type: srcType,
	}
	constraints.SourceSlots = append(constraints.SourceSlots, slot)
	constraints.Sources = append(constraints.Sources, slot)
}

// The data location list must either be a single stack slot entry, or a list
// of selected registers (the list could be empty if the destination value
// does not occupy any space, e.g., empty struct)
func (constraints *InstructionConstraints) SetRegisterDestination(
	registers ...*SelectedRegister,
) {
	if constraints.Destination != nil {
		panic("destination already set")
	}
	constraints.Destination = constraints.listToLocation(registers, true)
}

func (constraints *InstructionConstraints) SetStackDestination(
	destType ast.Type,
) {
	if constraints.Destination != nil {
		panic("destination already set")
	}

	slot := &StackSlot{
		Type: destType,
	}
	constraints.DestinationSlot = slot
	constraints.Destination = slot
}

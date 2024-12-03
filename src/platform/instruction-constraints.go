package platform

import (
	"github.com/pattyshack/chickadee/ast"
)

// Where the data is located.  The data location is either a *SelectedRegister,
// a *MultiRegisterLocation, or a *StackSlot.
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

var _ DataLocation = &SelectedRegister{}

func (*SelectedRegister) isDataLocation() {}

type MultiRegisterLocation struct {
	// The value is stored in an "array" formed by a list of registers.
	Registers []*SelectedRegister
}

var _ DataLocation = &MultiRegisterLocation{}

func (*MultiRegisterLocation) isDataLocation() {}

type StackSlot struct {
	Initialized bool // false when the slot is not used by any source value
	Type        ast.Type
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
	Used []*SelectedRegister

	// Which sources/destination values should be on stack.  The layout is
	// specified from bottom to top.  Note: The stack layout depends on
	// AddStackSlot calls order.
	//
	// Source value are copied into the stack slots, and destination's stack slot
	// is initialized to zeros.
	StackLayout []*StackSlot

	FuncValue DataLocation // only set by FuncCall
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
	constraints.Used = append(constraints.Used, reg)
	return reg
}

func (constraints *InstructionConstraints) AddStackSlot(
	valueType ast.Type,
) *StackSlot {
	slot := &StackSlot{
		Type: valueType,
	}
	constraints.StackLayout = append(constraints.StackLayout, slot)
	return slot
}

func (constraints *InstructionConstraints) listToLocation(
	list []DataLocation,
	isDest bool,
) DataLocation {
	slotCount := 0
	selected := []*SelectedRegister{}
	for _, loc := range list {
		switch entry := loc.(type) {
		case *SelectedRegister:
			if isDest {
				entry.Clobbered = true
			}
			selected = append(selected, entry)
		case *StackSlot:
			if !isDest {
				entry.Initialized = true
			}
			slotCount++
		default:
			panic("invalid input")
		}
	}

	var dest DataLocation
	if len(selected) > 0 {
		if slotCount > 0 {
			panic("mixing stack slot with register")
		} else if len(selected) == 1 {
			dest = selected[0]
		} else {
			dest = &MultiRegisterLocation{
				Registers: selected,
			}
		}
	} else if slotCount == 0 {
		panic("no data location specified")
	} else if slotCount > 1 {
		panic("multiple stack slot specified")
	} else {
		dest = list[0]
	}

	return dest
}

func (constraints *InstructionConstraints) SetFuncValue(
	regListOrSlot ...DataLocation,
) {
	if constraints.FuncValue != nil {
		panic("func value already set")
	}
	constraints.FuncValue = constraints.listToLocation(regListOrSlot, false)
}

// The data location list must either be a single stack slot entry, or a list
// of selected registers.
func (constraints *InstructionConstraints) AddSource(
	regListOrSlot ...DataLocation,
) {
	constraints.Sources = append(
		constraints.Sources,
		constraints.listToLocation(regListOrSlot, false))
}

// The data location list must either be a single stack slot entry, or a list
// of selected registers.
func (constraints *InstructionConstraints) SetDestination(
	regListOrSlot ...DataLocation,
) {
	if constraints.Destination != nil {
		panic("destination already set")
	}
	constraints.Destination = constraints.listToLocation(regListOrSlot, true)
}

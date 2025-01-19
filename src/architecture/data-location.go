package architecture

import (
	"fmt"

	"github.com/pattyshack/chickadee/ast"
)

type DataLocation struct {
	Name string

	ast.Type

	// XXX: Support register / stack overlay?
	//
	// For now, data location is either completely on stack or completely in
	// registers.
	Registers    []*Register
	OnFixedStack bool // available throughout the function's lifetime
	OnTempStack  bool // temporarily allocated for a call instruction

	AlignedSize int // register aligned size

	// All offsets are relative to the top of the (preallocated) stack.
	//
	// NOTE: We'll determine the stack entry address based on stack pointer
	// rather than base pointer:
	//
	// entry address = stack pointer address + offset
	Offset int
}

func NewRegistersDataLocation(
	name string,
	valType ast.Type,
	registers []*Register,
) *DataLocation {
	if len(registers) != NumRegisters(valType) {
		panic("should never happen")
	}

	return &DataLocation{
		Name:        name,
		Type:        valType,
		Registers:   registers,
		AlignedSize: AlignedSize(valType),
	}
}

func NewFixedStackDataLocation(
	name string,
	valType ast.Type,
) *DataLocation {
	return &DataLocation{
		Name:         name,
		Type:         valType,
		OnFixedStack: true,
		OnTempStack:  false,
		AlignedSize:  AlignedSize(valType),
	}
}

func NewTempStackDataLocation(
	name string,
	valType ast.Type,
) *DataLocation {
	return &DataLocation{
		Name:         name,
		Type:         valType,
		OnFixedStack: false,
		OnTempStack:  true,
		AlignedSize:  AlignedSize(valType),
	}
}

func (loc *DataLocation) Copy() *DataLocation {
	var registers []*Register
	if loc.Registers != nil {
		registers = make([]*Register, 0, len(loc.Registers))
		registers = append(registers, loc.Registers...)
	}

	return &DataLocation{
		Name:         loc.Name,
		Type:         loc.Type,
		Registers:    registers,
		OnFixedStack: loc.OnFixedStack,
		OnTempStack:  loc.OnTempStack,
		AlignedSize:  loc.AlignedSize,
		Offset:       loc.Offset,
	}
}

func (loc *DataLocation) String() string {
	registers := []string{}
	for _, reg := range loc.Registers {
		registers = append(registers, reg.Name)
	}
	return fmt.Sprintf(
		"Name: %s Registers: %v OnFixedStack: %v OnTempStack: %v "+
			"AlignedSize: %d Offset: %d Type: %s",
		loc.Name,
		registers,
		loc.OnFixedStack,
		loc.OnTempStack,
		loc.AlignedSize,
		loc.Offset,
		loc.Type)
}

func (loc *DataLocation) IsOnStack() bool {
	return loc.OnTempStack || loc.OnFixedStack
}

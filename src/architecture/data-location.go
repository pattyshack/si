package architecture

import (
	"fmt"
)

type DataLocation struct {
	Name string

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
	byteSize int,
	registers []*Register,
) *DataLocation {
	if len(registers) != NumRegisters(byteSize) {
		panic("should never happen")
	}

	return &DataLocation{
		Name:        name,
		Registers:   registers,
		AlignedSize: AlignedSize(byteSize),
	}
}

func NewFixedStackDataLocation(
	name string,
	byteSize int,
) *DataLocation {
	return &DataLocation{
		Name:         name,
		OnFixedStack: true,
		OnTempStack:  false,
		AlignedSize:  AlignedSize(byteSize),
	}
}

func NewTempStackDataLocation(
	byteSize int,
) *DataLocation {
	return &DataLocation{
		Name:         "",
		OnFixedStack: false,
		OnTempStack:  true,
		AlignedSize:  AlignedSize(byteSize),
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
		"Name: %s Registers: %v OnFixedStack: %v OnFixedStack: %v "+
			"AlignedSize: %d Offset: %d",
		loc.Name,
		registers,
		loc.OnFixedStack,
		loc.OnTempStack,
		loc.AlignedSize,
		loc.Offset)
}

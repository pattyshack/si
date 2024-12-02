package platform

type Register struct {
	Name string

	// When true, the register is reserved for stack pointer.
	IsStackPointer bool

	// When true, the register is usable for signed/unsigned int and pointer
	// operations, as well as general data storage.  Note that a general
	// register can	also be a float register.
	AllowGeneralOp bool

	// When true, the register is usable for float operation, as well as general
	// data storage.  Note that a float register can also be a general register.
	AllowFloatOp bool
}

func NewStackPointerRegister(name string) *Register {
	return &Register{
		Name:           name,
		IsStackPointer: true,
	}
}

func NewGeneralRegister(name string, isFloat bool) *Register {
	return &Register{
		Name:           name,
		AllowGeneralOp: true,
		AllowFloatOp:   isFloat,
	}
}

func NewFloatRegister(name string) *Register {
	return &Register{
		Name:         name,
		AllowFloatOp: true,
	}
}

type RegisterSet map[*Register]struct{}

func (set RegisterSet) Add(reg *Register) {
	set[reg] = struct{}{}
}

// Assumptions (probably needs visiting):
//
// 1. When a portion (e.g., AX) of a register is used, the entire
// register (e.g., RAX) is considered occupied.  i.e., a register cannot
// be partitioned into multiple disjointed registers.
//
// 2. Each architecture have exactly one stack pointer register.  The stack
// pinter is always live and hence can't be used as a general/float register.
//
// 3. We won't make use of a base pointer, however, all function call
// conventions should respect the register that normally holds the base
// pointer (e.g., RBP), and treat that register as callee-saved.
//
// 4. We can spill to any general/float register.
type ArchitectureRegisters struct {
	StackPointer *Register

	// The set of registers usable for signed/unsigned int and pointer operations.
	General RegisterSet

	// The set of registers usable for float operations.
	Float RegisterSet

	// All non-stack-pointer general/float registers, usable for temporary data
	// storage and register spilling.
	Data RegisterSet
}

func NewArchitectureRegisters(registers ...*Register) *ArchitectureRegisters {
	set := &ArchitectureRegisters{
		General: RegisterSet{},
		Float:   RegisterSet{},
		Data:    RegisterSet{},
	}

	names := map[string]struct{}{}
	for _, register := range registers {
		if register.Name == "" {
			panic("no register name")
		}

		_, ok := names[register.Name]
		if ok {
			panic("added duplicate register: " + register.Name)
		}
		names[register.Name] = struct{}{}

		set.add(register)
	}

	if set.StackPointer == nil {
		panic("no stack pointer register specified")
	}

	return set
}

func (set *ArchitectureRegisters) add(register *Register) {
	if register.IsStackPointer {
		if register.AllowGeneralOp || register.AllowFloatOp {
			panic("stack pointer register cannot be general/float register")
		}

		if set.StackPointer != nil {
			panic("multiple stack pointer register specified")
		}
		set.StackPointer = register
		return
	}

	if !register.AllowGeneralOp && !register.AllowFloatOp {
		panic("added unusable register")
	}

	set.Data.Add(register)

	if register.AllowGeneralOp {
		set.General.Add(register)
	}

	if register.AllowFloatOp {
		set.Float.Add(register)
	}
}

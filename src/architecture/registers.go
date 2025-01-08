package architecture

const (
	// Internal labels for various stack/register data locations
	PreviousFramePointer = "%previous-frame-pointer"
	ReturnAddress        = "%return-address"
	StackDestination     = "%stack-destination"
)

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

	// Internally assigned.  Stack pointer register always have index -1.
	// Data registers' index are in the ordered given by NewRegisterSet
	Index int
}

func NewStackPointerRegister(name string) *Register {
	return &Register{
		Name:           name,
		IsStackPointer: true,
		Index:          -1,
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
type RegisterSet struct {
	StackPointer *Register

	// The set of registers usable for signed/unsigned int and pointer operations.
	General []*Register

	// The set of registers usable for float operations.
	Float []*Register

	// All non-stack-pointer general/float registers, usable for temporary data
	// storage and register spilling.
	Data []*Register
}

func NewRegisterSet(registers ...*Register) *RegisterSet {
	set := &RegisterSet{}

	idx := 0
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

		if !register.IsStackPointer {
			register.Index = idx
			idx++
		}

		set.add(register)
	}

	if set.StackPointer == nil {
		panic("no stack pointer register specified")
	}

	return set
}

func (set *RegisterSet) add(register *Register) {
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

	set.Data = append(set.Data, register)

	if register.AllowGeneralOp {
		set.General = append(set.General, register)
	}

	if register.AllowFloatOp {
		set.Float = append(set.Float, register)
	}
}

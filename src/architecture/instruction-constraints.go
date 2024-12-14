package architecture

// A yet to be determined register.
type RegisterCandidate struct {
	// Clobbered registers are caller-saved; registers are callee-saved otherwise.
	Clobbered bool

	AnyGeneral bool
	AnyFloat   bool

	// XXX: do we need to support custom candidates set?

	Require *Register
}

// Where the data is located.
//
// Assumptions: A value must either be completely on register, or completely
// on stack.
type LocationConstraint struct {
	Size int // in byte

	// When true, the data could be on either stack or registers
	AnyLocation bool

	// When true, the data must be on stack
	RequireOnStack bool

	// The value is stored in an "array" formed by a list of registers.  The list
	// could be empty to indicate a zero-sized type (e.g., empty struct)
	Registers []*RegisterCandidate
}

// InstructionConstraints is used to specify instruction register constraints,
// and call convention's registers selection / stack layout.
//
// Note:
//  1. Source and destination LocationConstraint that share the same
//     AnyGeneral/AnyFloat RegisterCandidate pointer shares the same selected
//     register.
//  2. it's safe to reuse the same instruction constraints for multiple
//     instructions.
//  3. do not manually modify the fields. Use the provided methods instead.
type InstructionConstraints struct {
	// Register -> clobbered.  This is mainly used by call convention.
	RequiredRegisters map[*Register]bool

	// Which sources/destination values should be on stack.  The layout is
	// specified from top to bottom (stack destination is always at the bottom).
	// Note: The stack layout depends on AddStackSource calls order.
	//
	// Source value are copied into the stack slots, and destination's stack slot
	// is initialized to zeros.
	//
	// All stack sources are caller-saved and their values may be modified
	// by the instruction/call.
	// XXX: maybe add option to control this behavior
	//
	// That ret instruction uses caller's preallocated stack location rather than
	// initializing a new location.
	SrcStackLocations []*LocationConstraint
	// nil if the destination is on registers
	DestStackLocation *LocationConstraint

	// Source data locations are in the same order as the instruction's Sources().
	// For call, the first entry is func value.
	Sources []*LocationConstraint
	// Pseudo sources are used to track callee-saved pseudo source registers in
	// function definition (populated by funcDefConstraintsGenerator).  It's also
	// used as temporary storage to keep track of callee-saved parameters in ret
	// instruction (funcDefConstraintsGenerator move these to Sources after the
	// destination is set)
	PseudoSources []*LocationConstraint
	Destination   *LocationConstraint // not set by control flow instructions

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
	return &InstructionConstraints{
		RequiredRegisters: map[*Register]bool{},
	}
}

func (constraints *InstructionConstraints) SelectAnyGeneral(
	clobbered bool,
) *RegisterCandidate {
	return &RegisterCandidate{
		Clobbered:  clobbered,
		AnyGeneral: true,
	}
}

func (constraints *InstructionConstraints) SelectAnyFloat(
	clobbered bool,
) *RegisterCandidate {
	return &RegisterCandidate{
		Clobbered: clobbered,
		AnyFloat:  true,
	}
}

func (constraints *InstructionConstraints) Require(
	clobbered bool,
	register *Register,
) *RegisterCandidate {
	if register.IsStackPointer {
		panic("cannot select stack pointer")
	}

	orig, ok := constraints.RequiredRegisters[register]
	if ok {
		if orig != clobbered {
			panic(register.Name + " cannot be both clobbered and not clobbered")
		}
	} else {
		constraints.RequiredRegisters[register] = clobbered
	}

	return &RegisterCandidate{
		Clobbered: clobbered,
		Require:   register,
	}
}

func (constraints *InstructionConstraints) registerLocation(
	isDest bool,
	list ...*RegisterCandidate,
) *LocationConstraint {
	for _, entry := range list {
		if isDest {
			entry.Clobbered = true
		}
	}

	return &LocationConstraint{
		Registers: list,
	}
}

func (constraints *InstructionConstraints) AddAnySource(
	size int,
) {
	constraints.Sources = append(
		constraints.Sources,
		&LocationConstraint{
			Size:        size,
			AnyLocation: true,
		})
}

// The list could be empty if source value does not occupy any space,
// e.g., empty struct)
func (constraints *InstructionConstraints) AddRegisterSource(
	registers ...*RegisterCandidate,
) {
	constraints.Sources = append(
		constraints.Sources,
		constraints.registerLocation(false, registers...))
}

func (constraints *InstructionConstraints) AddStackSource(
	size int,
) {
	loc := &LocationConstraint{
		Size:           size,
		RequireOnStack: true,
	}
	constraints.SrcStackLocations = append(constraints.SrcStackLocations, loc)
	constraints.Sources = append(constraints.Sources, loc)
}

func (constraints *InstructionConstraints) AddPseudoSource(
	registers ...*RegisterCandidate,
) {
	constraints.PseudoSources = append(
		constraints.PseudoSources,
		constraints.registerLocation(false, registers...))
}

// The list could be empty if the destination value does not occupy any
// space, e.g., empty struct
func (constraints *InstructionConstraints) SetRegisterDestination(
	registers ...*RegisterCandidate,
) {
	if constraints.Destination != nil {
		panic("destination already set")
	}
	loc := constraints.registerLocation(true, registers...)
	constraints.Destination = loc
}

func (constraints *InstructionConstraints) SetStackDestination(
	size int,
) {
	if constraints.Destination != nil {
		panic("destination already set")
	}

	loc := &LocationConstraint{
		Size:           size,
		RequireOnStack: true,
	}
	constraints.DestStackLocation = loc
	constraints.Destination = loc
}

type CallConvention struct {
	CallConstraints          *InstructionConstraints
	CalleeSavedSourceIndices []int // list of callee saved sources from call

	RetConstraints *InstructionConstraints
}

func NewCallConvention(
	funcValueClobbered bool,
	funcValueRegister *Register,
) *CallConvention {
	con := &CallConvention{
		CallConstraints: NewInstructionConstraints(),
		RetConstraints:  NewInstructionConstraints(),
	}

	con.CallConstraints.AddRegisterSource(
		con.CallConstraints.Require(funcValueClobbered, funcValueRegister))
	return con
}

func (con *CallConvention) AddRegisterSource(
	clobbered bool,
	registers ...*Register,
) {
	callSrc := []*RegisterCandidate{}
	for _, reg := range registers {
		callSrc = append(callSrc, con.CallConstraints.Require(clobbered, reg))
	}
	con.CallConstraints.AddRegisterSource(callSrc...)

	if !clobbered {
		pseudoSrc := []*RegisterCandidate{}
		for _, reg := range registers {
			pseudoSrc = append(pseudoSrc, con.RetConstraints.Require(false, reg))
		}
		con.RetConstraints.AddPseudoSource(pseudoSrc...)

		con.CalleeSavedSourceIndices = append(
			con.CalleeSavedSourceIndices,
			len(con.CallConstraints.Sources)-1)
	}
}

func (con *CallConvention) AddStackSource(
	size int,
) {
	con.CallConstraints.AddStackSource(size)
}

func (con *CallConvention) SetRegisterDestination(
	registers ...*Register,
) {
	callDest := []*RegisterCandidate{}
	retSrc := []*RegisterCandidate{}
	for _, reg := range registers {
		callDest = append(callDest, con.CallConstraints.Require(true, reg))
		retSrc = append(retSrc, con.RetConstraints.Require(true, reg))
	}
	con.CallConstraints.SetRegisterDestination(callDest...)
	con.RetConstraints.AddRegisterSource(retSrc...)
}

func (con *CallConvention) SetStackDestination(
	size int,
) {
	con.CallConstraints.SetStackDestination(size)
	con.RetConstraints.AddStackSource(size)
}

// All caller-saved registers that are clobbered by the call.
func (con *CallConvention) CallerSaved(registers ...*Register) {
	for _, reg := range registers {
		con.CallConstraints.Require(true, reg)
	}
}

// All callee-saved registers that are not clobbered by the call.
func (con *CallConvention) CalleeSaved(registers ...*Register) {
	for _, reg := range registers {
		con.CallConstraints.Require(false, reg)
	}
}

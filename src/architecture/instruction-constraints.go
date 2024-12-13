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
//  1. both call and ret instruction for the same function definition shares
//     the same constraints. Ret derived its constraints from the call
//     constraints' destination and pseudo sources.
//  2. it's safe to reuse the same constraints from multiple instructions.
//  3. do not manually modify the fields. Use the provided methods instead.
type InstructionConstraints struct {
	// The unordered set of registers used by this instruction.  Entries are
	// created by Select*.  Note that this set may include registers not used by
	// sources and destination.  LocationConstraint that share the same
	// RegisterCandidate pointer shares the same selected register.
	UsedRegisters []*RegisterCandidate

	// Which sources/destination values should be on stack.  The layout is
	// specified from top to bottom (stack destination is always at the bottom).
	// Note: The stack layout depends on AddStackSource calls order.
	//
	// Source value are copied into the stack slots, and destination's stack slot
	// is initialized to zeros.
	//
	// All stack sources are "callee-saved" and retain the original value at
	// the end of call's execution.  This is potentially less memory efficient,
	// but does not leak data to caller.
	//
	// That ret instruction uses caller's preallocated stack location rather than
	// initializing a new location.
	SrcStackLocations []*LocationConstraint
	// nil if the destination is on registers
	DestStackLocation *LocationConstraint

	// Only set by FuncCall (must be a register)
	FuncValue *LocationConstraint
	// Source data locations are in the same order as the instruction's sources.
	Sources []*LocationConstraint
	// Pseudo sources are used to track callee-saved registers in call
	// conventions.
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
	return &InstructionConstraints{}
}

func (constraints *InstructionConstraints) SelectAnyGeneral(
	clobbered bool,
) *RegisterCandidate {
	reg := &RegisterCandidate{
		Clobbered:  clobbered,
		AnyGeneral: true,
	}
	constraints.UsedRegisters = append(constraints.UsedRegisters, reg)
	return reg
}

func (constraints *InstructionConstraints) SelectAnyFloat(
	clobbered bool,
) *RegisterCandidate {
	reg := &RegisterCandidate{
		Clobbered: clobbered,
		AnyFloat:  true,
	}
	constraints.UsedRegisters = append(constraints.UsedRegisters, reg)
	return reg
}

func (constraints *InstructionConstraints) Require(
	clobbered bool,
	register *Register,
) *RegisterCandidate {
	if register.IsStackPointer {
		panic("cannot select stack pointer")
	}

	reg := &RegisterCandidate{
		Clobbered: clobbered,
		Require:   register,
	}
	constraints.UsedRegisters = append(constraints.UsedRegisters, reg)
	return reg
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

func (constraints *InstructionConstraints) SetFuncValue(
	register *RegisterCandidate,
) {
	if constraints.FuncValue != nil {
		panic("func value already set")
	}
	loc := constraints.registerLocation(false, register)
	constraints.FuncValue = loc
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
	register *Register,
) {
	constraints.PseudoSources = append(
		constraints.PseudoSources,
		constraints.registerLocation(
			false,
			constraints.Require(false, register)))
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

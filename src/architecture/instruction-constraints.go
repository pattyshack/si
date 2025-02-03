package architecture

import (
	"github.com/pattyshack/chickadee/ast"
)

// A yet to be determined register.
type RegisterConstraint struct {
	// Clobbered registers are caller-saved; registers are callee-saved otherwise.
	Clobbered bool

	AnyGeneral bool
	AnyFloat   bool

	// XXX: do we need to support custom candidates set?

	Require *Register
}

func (candidate *RegisterConstraint) SatisfyBy(register *Register) bool {
	if candidate.Require != nil {
		return candidate.Require == register
	}

	if candidate.AnyGeneral && register.AllowGeneralOp {
		return true
	}

	if candidate.AnyFloat && register.AllowFloatOp {
		return true
	}

	return false
}

// Where the data is located.
//
// Assumptions: A value must either be completely on register, or completely
// on stack.
type LocationConstraint struct {
	NumRegisters int // size in number of registers

	// When true, the data could be on either stack or registers
	AnyLocation bool

	// When true, the data must be on stack
	RequireOnStack bool

	// The value is stored in an "array" formed by a list of registers.  The list
	// could be empty to indicate a zero-sized type (e.g., empty struct)
	Registers []*RegisterConstraint

	// When true and the source value is an immediate (or a label reference),
	// the source value could be encoded as part of the instruction if the value
	// passes the Platform.CanEncodeImmediate check.
	SupportEncodedImmediate bool
}

func (loc *LocationConstraint) ClobberedByInstruction() bool {
	return loc.RequireOnStack ||
		(loc.NumRegisters > 0 && loc.Registers[0].Clobbered)
}

// TODO Add option to allow register source reuse
// TODO Add option to allow immediate bypassing register
//
// InstructionConstraints is used to specify instruction register constraints,
// and call convention's registers selection / stack layout.
//
// Note:
//  1. Source and destination LocationConstraint that share the same
//     AnyGeneral/AnyFloat RegisterConstraint pointer shares the same selected
//     register.
//  2. it's safe to reuse the same instruction constraints for multiple
//     instructions.
//  3. do not manually modify the fields. Use the provided methods instead.
type InstructionConstraints struct {
	// Register -> clobbered.  This is mainly used by call convention.
	RequiredRegisters map[*Register]bool

	// FramePointerRegister is a special pseudo callee-saved register reserved
	// for the frame pointer hidden parameter.  A corresponding location
	// constraint entry is in the pseudo sources list (added by
	// funcDefConstraintsGenerator).
	FramePointerRegister *Register

	// Source data locations are in the same order as the instruction's Sources().
	// For call, the first entry is func value.
	//
	// Note that the call stack layout (from top to bottom) is in the same order
	// as sources (register locations don't take up any stack space); stack
	// destination is always at the bottom.
	//
	// Source value are copied into the stack slots, and destination's stack slot
	// is initialized to zeros.
	//
	// All stack sources are caller-saved and their values may be modified
	// by the instruction/call.
	// XXX: maybe add option to control this behavior
	//
	// ret instruction will use caller's preallocated stack location rather than
	// initializing a new location.
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

func (constraints *InstructionConstraints) AllSources() []*LocationConstraint {
	result := make(
		[]*LocationConstraint,
		0,
		len(constraints.Sources)+len(constraints.PseudoSources))
	result = append(result, constraints.Sources...)
	result = append(result, constraints.PseudoSources...)
	return result
}

func (constraints *InstructionConstraints) SelectAnyGeneral(
	clobbered bool,
) *RegisterConstraint {
	return &RegisterConstraint{
		Clobbered:  clobbered,
		AnyGeneral: true,
	}
}

func (constraints *InstructionConstraints) SelectAnyFloat(
	clobbered bool,
) *RegisterConstraint {
	return &RegisterConstraint{
		Clobbered: clobbered,
		AnyFloat:  true,
	}
}

func (constraints *InstructionConstraints) Require(
	clobbered bool,
	register *Register,
) *RegisterConstraint {
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

	return &RegisterConstraint{
		Clobbered: clobbered,
		Require:   register,
	}
}

func (constraints *InstructionConstraints) registerLocation(
	isDest bool,
	supportEncodedImmediate bool,
	list ...*RegisterConstraint,
) *LocationConstraint {
	for _, entry := range list {
		if isDest {
			entry.Clobbered = true
		}
	}

	return &LocationConstraint{
		NumRegisters:            len(list),
		Registers:               list,
		SupportEncodedImmediate: supportEncodedImmediate,
	}
}

// Can only be used by ast.CopyOperation
func (constraints *InstructionConstraints) AddAnyCopySource(
	valueType ast.Type,
) {
	constraints.Sources = append(
		constraints.Sources,
		&LocationConstraint{
			NumRegisters: NumRegisters(valueType),
			AnyLocation:  true,
		})
}

// The list could be empty if source value does not occupy any space,
// e.g., empty struct)
func (constraints *InstructionConstraints) AddRegisterSource(
	supportEncodedImmediate bool,
	registers ...*RegisterConstraint,
) {
	constraints.Sources = append(
		constraints.Sources,
		constraints.registerLocation(false, supportEncodedImmediate, registers...))
}

func (constraints *InstructionConstraints) AddStackSource(
	valueType ast.Type,
) {
	loc := &LocationConstraint{
		NumRegisters:   NumRegisters(valueType),
		RequireOnStack: true,
	}
	constraints.Sources = append(constraints.Sources, loc)
}

func (constraints *InstructionConstraints) SetFramePointerRegister(
	register *Register,
) {
	if constraints.FramePointerRegister != nil {
		panic("frame pointer register already set")
	}

	constraints.FramePointerRegister = register
	constraints.Require(false, register)
}

func (constraints *InstructionConstraints) AddPseudoSource(
	registers ...*RegisterConstraint,
) {
	constraints.PseudoSources = append(
		constraints.PseudoSources,
		constraints.registerLocation(false, false, registers...))
}

// The list could be empty if the destination value does not occupy any
// space, e.g., empty struct
func (constraints *InstructionConstraints) SetRegisterDestination(
	registers ...*RegisterConstraint,
) {
	if constraints.Destination != nil {
		panic("destination already set")
	}
	loc := constraints.registerLocation(true, false, registers...)
	constraints.Destination = loc
}

func (constraints *InstructionConstraints) SetStackDestination(
	valueType ast.Type,
) {
	if constraints.Destination != nil {
		panic("destination already set")
	}

	loc := &LocationConstraint{
		NumRegisters:   NumRegisters(valueType),
		RequireOnStack: true,
	}
	constraints.Destination = loc
}

// Can only be used by ast.CopyOperation
func (constraints *InstructionConstraints) SetAnyCopyDestination(
	valueType ast.Type,
) {
	if constraints.Destination != nil {
		panic("destination already set")
	}

	constraints.Destination = &LocationConstraint{
		NumRegisters: NumRegisters(valueType),
		AnyLocation:  true,
	}
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
		false,
		con.CallConstraints.Require(funcValueClobbered, funcValueRegister))
	return con
}

func (con *CallConvention) AddRegisterSource(
	clobbered bool,
	registers ...*Register,
) {
	callSrc := []*RegisterConstraint{}
	for _, reg := range registers {
		callSrc = append(callSrc, con.CallConstraints.Require(clobbered, reg))
	}
	con.CallConstraints.AddRegisterSource(false, callSrc...)

	if !clobbered {
		pseudoSrc := []*RegisterConstraint{}
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
	valueType ast.Type,
) {
	con.CallConstraints.AddStackSource(valueType)
}

func (con *CallConvention) SetFramePointerRegister(
	register *Register,
) {
	con.CallConstraints.SetFramePointerRegister(register)
}

func (con *CallConvention) SetRegisterDestination(
	registers ...*Register,
) {
	callDest := []*RegisterConstraint{}
	retSrc := []*RegisterConstraint{}
	for _, reg := range registers {
		callDest = append(callDest, con.CallConstraints.Require(true, reg))
		retSrc = append(retSrc, con.RetConstraints.Require(true, reg))
	}
	con.CallConstraints.SetRegisterDestination(callDest...)
	con.RetConstraints.AddRegisterSource(false, retSrc...)
}

func (con *CallConvention) SetStackDestination(
	valueType ast.Type,
) {
	con.CallConstraints.SetStackDestination(valueType)
	con.RetConstraints.AddStackSource(valueType)
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

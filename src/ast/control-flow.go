package ast

import (
	"github.com/pattyshack/gt/parseutil"
)

// XXX: Need to support generic jump to some arbitrary offset?  Preferably not
// since single entry point per block simplifies ssa generation.

type JumpKind string

const (
	Jmp = JumpKind("jmp")
)

// Unconditional jump instruction of the form: jmp <label>
type Jump struct {
	controlFlowInstruction

	parseutil.StartEndPos

	Kind JumpKind

	Label string
}

var _ Instruction = &Jump{}
var _ Validator = &Jump{}

func (jump *Jump) Walk(visitor Visitor) {
	visitor.Enter(jump)
	visitor.Exit(jump)
}

func (jump *Jump) Validate(emitter *parseutil.Emitter) {
	if jump.Kind != Jmp {
		emitter.Emit(
			jump.Loc(),
			"unexpected unconditional jump kind (%s)",
			jump.Kind)
	}
}

type ConditionalJumpKind string

const (
	Jeq = ConditionalJumpKind("jeq")
	Jne = ConditionalJumpKind("jne")
	Jlt = ConditionalJumpKind("jlt")
	Jge = ConditionalJumpKind("jge")
)

// Instructions of the form: <op> <label>, <src1>, <src2>
type ConditionalJump struct {
	controlFlowInstruction

	parseutil.StartEndPos

	Kind ConditionalJumpKind

	Label string
	Src1  Value
	Src2  Value
}

var _ Instruction = &ConditionalJump{}
var _ Validator = &ConditionalJump{}

func (jump *ConditionalJump) Walk(visitor Visitor) {
	visitor.Enter(jump)
	jump.Src1.Walk(visitor)
	jump.Src2.Walk(visitor)
	visitor.Exit(jump)
}

func (jump *ConditionalJump) Validate(emitter *parseutil.Emitter) {
	switch jump.Kind {
	case Jeq, Jne, Jlt, Jge: // ok
	default:
		emitter.Emit(jump.Loc(), "unexpected conditional jump kind (%s)", jump.Kind)
	}
}

type TerminalKind string

const (
	Ret  = TerminalKind("ret")
	Exit = TerminalKind("exit")
)

// Exit instruction of the form: <op> <src>
//
// Note: this translates into a syscall instruction, but we've special cased
// exit since it has semantic meaning in the control flow graph.
type Terminal struct {
	controlFlowInstruction

	parseutil.StartEndPos

	Kind TerminalKind

	Src Value
}

var _ Instruction = &Terminal{}
var _ Validator = &Terminal{}

func (term *Terminal) Walk(visitor Visitor) {
	visitor.Enter(term)
	term.Src.Walk(visitor)
	visitor.Exit(term)
}

func (term *Terminal) Validate(emitter *parseutil.Emitter) {
	switch term.Kind {
	case Ret, Exit: // ok
	default:
		emitter.Emit(term.Loc(), "unexpected terminate kind (%s)", term.Kind)
	}
}

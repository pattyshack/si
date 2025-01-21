package ast

import (
	"fmt"
	"strings"

	"github.com/pattyshack/gt/parseutil"
)

type ControlFlowInstruction interface {
	Instruction
	isControlFlow()
}

type controlFlowInstruction struct {
	instruction
}

func (controlFlowInstruction) isControlFlow() {}

// XXX: Need to support generic jump to some arbitrary offset?  Preferably not
// since single entry point per block simplifies ssa generation.

// Unconditional jump instruction of the form: jmp <label>
type Jump struct {
	controlFlowInstruction

	parseutil.StartEndPos

	Label string
}

var _ Instruction = &Jump{}
var _ Validator = &Jump{}

func (Jump) replaceSource(Value, Value) {
	panic("should never happen")
}

func (Jump) Sources() []Value {
	return nil
}

func (Jump) Destination() *VariableDefinition {
	return nil
}

func (jump *Jump) Walk(visitor Visitor) {
	visitor.Enter(jump)
	visitor.Exit(jump)
}

func (jump *Jump) Validate(emitter *parseutil.Emitter) {
	if strings.HasPrefix(jump.Label, ":") {
		emitter.Emit(jump.Loc(), ":-prefixed label is reserved for internal use")
	}
}

func (jump *Jump) String() string {
	return fmt.Sprintf("jmp :%s", jump.Label)
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

func (jump *ConditionalJump) replaceSource(oldSrc Value, newSrc Value) {
	replaceCount := 0
	if jump.Src1 == oldSrc {
		jump.Src1 = newSrc
		replaceCount++
	}
	if jump.Src2 == oldSrc {
		jump.Src2 = newSrc
		replaceCount++
	}

	if replaceCount != 1 {
		panic("should never happen")
	}
}

func (jump *ConditionalJump) Sources() []Value {
	return []Value{jump.Src1, jump.Src2}
}

func (ConditionalJump) Destination() *VariableDefinition {
	return nil
}

func (jump *ConditionalJump) Walk(visitor Visitor) {
	visitor.Enter(jump)
	jump.Src1.Walk(visitor)
	jump.Src2.Walk(visitor)
	visitor.Exit(jump)
}

func (jump *ConditionalJump) Validate(emitter *parseutil.Emitter) {
	if strings.HasPrefix(jump.Label, ":") {
		emitter.Emit(jump.Loc(), ":-prefixed label is reserved for internal use")
	}

	switch jump.Kind {
	case Jeq, Jne, Jlt, Jge: // ok
	default:
		emitter.Emit(jump.Loc(), "unexpected conditional jump kind (%s)", jump.Kind)
	}
}

func (jump *ConditionalJump) String() string {
	return fmt.Sprintf(
		"%s :%s, %s, %s",
		jump.Kind,
		jump.Label,
		jump.Src1,
		jump.Src2)
}

type TerminalKind string

const (
	Ret  = TerminalKind("ret")
	Exit = TerminalKind("exit")
)

// This returns true if the instruction is either
//  1. a *Terminal, or
//  2. a *FuncCall with IsExitTerminal set.
func IsTerminal(inst Instruction) bool {
	_, ok := inst.(*Terminal)
	if ok {
		return true
	}

	funcCall, ok := inst.(*FuncCall)
	if ok {
		return funcCall.IsExitTerminal
	}

	return false
}

// Terminal instruction of the form: <op> <src>
//
// Note: exit instruction is translated into a syscall instruction immediately
// after initializing the control flow graph, and serve no purpose afterward.
type Terminal struct {
	controlFlowInstruction

	parseutil.StartEndPos

	Kind TerminalKind

	RetVal Value

	// Internal

	// Only used by ret instruction.
	//
	// Callee-saved register values must be restore to their original register
	// before returning.
	CalleeSavedSources []Value
}

var _ Instruction = &Terminal{}
var _ Validator = &Terminal{}

func (term *Terminal) replaceSource(oldSrc Value, newSrc Value) {
	replaceCount := 0
	if term.RetVal == oldSrc {
		term.RetVal = newSrc
		replaceCount++
	}

	for idx, src := range term.CalleeSavedSources {
		if src == oldSrc {
			term.CalleeSavedSources[idx] = newSrc
			replaceCount++
		}
	}

	if replaceCount != 1 {
		panic("should never happen")
	}
}

func (term *Terminal) Sources() []Value {
	return append([]Value{term.RetVal}, term.CalleeSavedSources...)
}

func (Terminal) Destination() *VariableDefinition {
	return nil
}

func (term *Terminal) Walk(visitor Visitor) {
	visitor.Enter(term)
	term.RetVal.Walk(visitor)
	for _, src := range term.CalleeSavedSources {
		src.Walk(visitor)
	}
	visitor.Exit(term)
}

func (term *Terminal) Validate(emitter *parseutil.Emitter) {
	switch term.Kind {
	case Ret, Exit: // ok
	default:
		emitter.Emit(term.Loc(), "unexpected terminate kind (%s)", term.Kind)
	}
}

func (term *Terminal) String() string {
	calleeSavedParameters := ""
	for idx, val := range term.CalleeSavedSources {
		if idx > 0 {
			calleeSavedParameters += ", "
		}
		calleeSavedParameters += val.String()
	}

	return fmt.Sprintf(
		"%s %s [%s]",
		term.Kind,
		term.RetVal,
		calleeSavedParameters)
}

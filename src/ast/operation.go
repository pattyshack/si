package ast

import (
	"github.com/pattyshack/gt/parseutil"
)

type AssignOperation struct {
	instruction

	parseutil.StartEndPos

	Dest *RegisterDefinition
	Src  Value
}

var _ Instruction = &AssignOperation{}

func (assign *AssignOperation) Sources() []Value {
	return []Value{assign.Src}
}

func (assign *AssignOperation) Destination() *RegisterDefinition {
	return assign.Dest
}

func (assign *AssignOperation) Walk(visitor Visitor) {
	visitor.Enter(assign)
	assign.Dest.Walk(visitor)
	assign.Src.Walk(visitor)
	visitor.Exit(assign)
}

type UnaryOperationKind string

const (
	Neg = UnaryOperationKind("neg")
	Not = UnaryOperationKind("not")
)

// Instructions of the form: <dest> = <type> <src>
type UnaryOperation struct {
	instruction

	parseutil.StartEndPos

	Kind UnaryOperationKind

	Dest *RegisterDefinition
	Src  Value
}

var _ Instruction = &UnaryOperation{}
var _ Validator = &UnaryOperation{}

func (unary *UnaryOperation) Sources() []Value {
	return []Value{unary.Src}
}

func (unary *UnaryOperation) Destination() *RegisterDefinition {
	return unary.Dest
}

func (unary *UnaryOperation) Walk(visitor Visitor) {
	visitor.Enter(unary)
	unary.Dest.Walk(visitor)
	unary.Src.Walk(visitor)
	visitor.Exit(unary)
}

func (unary *UnaryOperation) Validate(emitter *parseutil.Emitter) {
	switch unary.Kind {
	case Neg, Not: // ok
	default:
		emitter.Emit(unary.Loc(), "unexpected unary operation (%s)", unary.Kind)
	}
}

type BinaryOperationKind string

const (
	Add = BinaryOperationKind("add")
	Sub = BinaryOperationKind("sub")
	Mul = BinaryOperationKind("mul")
	// uint uses div, int uses idiv
	Div = BinaryOperationKind("div")
	Rem = BinaryOperationKind("rem")
	Xor = BinaryOperationKind("xor")
	Or  = BinaryOperationKind("or")
	And = BinaryOperationKind("and")
	Shl = BinaryOperationKind("shl")
	// uint uses logical shift shr, int uses arithmetic shift sar
	Shr = BinaryOperationKind("shr")
	Slt = BinaryOperationKind("slt") // dest = (src1 < src2)? 1 : 0
)

// Instructions of the form: <dest> = <type> <src1>, <src2>
type BinaryOperation struct {
	instruction

	parseutil.StartEndPos

	Kind BinaryOperationKind

	Dest *RegisterDefinition
	Src1 Value
	Src2 Value
}

var _ Instruction = &BinaryOperation{}
var _ Validator = &BinaryOperation{}

func (binary *BinaryOperation) Sources() []Value {
	return []Value{binary.Src1, binary.Src2}
}

func (binary *BinaryOperation) Destination() *RegisterDefinition {
	return binary.Dest
}

func (binary *BinaryOperation) Walk(visitor Visitor) {
	visitor.Enter(binary)
	binary.Dest.Walk(visitor)
	binary.Src1.Walk(visitor)
	binary.Src2.Walk(visitor)
	visitor.Exit(binary)
}

func (binary *BinaryOperation) Validate(emitter *parseutil.Emitter) {
	switch binary.Kind {
	case Add, Sub, Mul, Div, Rem, Xor, Or, And, Shl, Shr, Slt: // ok
	default:
		emitter.Emit(binary.Loc(), "unexpected binary operation (%s)", binary.Kind)
	}
}

type FuncCallKind string

const (
	Call    = FuncCallKind("call")
	SysCall = FuncCallKind("syscall")
)

// Call of the form: [dests]* = <op> <func/sysno> ( [srcs,]* )
//
// The number of return values and arguments must match the function/syscall's
// signature.
type FuncCall struct {
	instruction

	parseutil.StartEndPos

	Kind FuncCallKind

	Dest *RegisterDefinition
	Func Value
	Srcs []Value
}

var _ Instruction = &FuncCall{}
var _ Validator = &FuncCall{}

func (call *FuncCall) Sources() []Value {
	return call.Srcs
}

func (call *FuncCall) Destination() *RegisterDefinition {
	return call.Dest
}

func (call *FuncCall) Walk(visitor Visitor) {
	visitor.Enter(call)
	call.Dest.Walk(visitor)
	call.Func.Walk(visitor)
	for _, src := range call.Srcs {
		src.Walk(visitor)
	}
	visitor.Exit(call)
}

func (call *FuncCall) Validate(emitter *parseutil.Emitter) {
	switch call.Kind {
	case Call, SysCall: // ok
	default:
		emitter.Emit(call.Loc(), "unexpected call operation (%s)", call.Kind)
	}
}

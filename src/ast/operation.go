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

func (unary *UnaryOperation) Walk(visitor Visitor) {
	visitor.Enter(unary)
	unary.Dest.Walk(visitor)
	unary.Src.Walk(visitor)
	visitor.Exit(unary)
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

func (binary *BinaryOperation) Walk(visitor Visitor) {
	visitor.Enter(binary)
	binary.Dest.Walk(visitor)
	binary.Src1.Walk(visitor)
	binary.Src2.Walk(visitor)
	visitor.Exit(binary)
}

var _ Instruction = &BinaryOperation{}

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

func (call *FuncCall) Walk(visitor Visitor) {
	visitor.Enter(call)
	call.Dest.Walk(visitor)
	call.Func.Walk(visitor)
	for _, src := range call.Srcs {
		src.Walk(visitor)
	}
	visitor.Exit(call)
}

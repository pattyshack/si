package ast

import (
	"github.com/pattyshack/gt/parseutil"
)

type Instruction interface {
	Node
	Line

	ParentBlock() *Block
	SetParentBlock(*Block)

	// NOTE: caller is responsible for copying newSrc and discarding oldSrc.
	replaceSource(oldSrc Value, newSrc Value)

	Sources() []Value                 // empty if there are no src dependencies
	Destination() *RegisterDefinition // nil if instruction has no destination
}

type instruction struct {
	// Internal (set during ssa construction)
	Parent *Block
}

func (instruction) IsLine() {}

func (ins *instruction) ParentBlock() *Block {
	return ins.Parent
}

func (ins *instruction) SetParentBlock(block *Block) {
	ins.Parent = block
}

type AssignOperation struct {
	instruction

	parseutil.StartEndPos

	Dest *RegisterDefinition
	Src  Value
}

var _ Instruction = &AssignOperation{}

func (assign *AssignOperation) replaceSource(oldVal Value, newVal Value) {
	if assign.Src != oldVal {
		panic("should never happen")
	}

	assign.Src = newVal
}

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
	Neg   = UnaryOperationKind("neg")
	Not   = UnaryOperationKind("not")
	ToI8  = UnaryOperationKind("toI8")
	ToI16 = UnaryOperationKind("toI16")
	ToI32 = UnaryOperationKind("toI32")
	ToI64 = UnaryOperationKind("toI64")
	ToU8  = UnaryOperationKind("toU8")
	ToU16 = UnaryOperationKind("toU16")
	ToU32 = UnaryOperationKind("toU32")
	ToU64 = UnaryOperationKind("toU64")
	ToF32 = UnaryOperationKind("toF32")
	ToF64 = UnaryOperationKind("toF64")
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

func (unary *UnaryOperation) replaceSource(oldVal Value, newVal Value) {
	if unary.Src != oldVal {
		panic("should never happen")
	}

	unary.Src = newVal
}

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
	case Neg, Not,
		ToI8, ToI16, ToI32, ToI64,
		ToU8, ToU16, ToU32, ToU64,
		ToF32, ToF64: // ok
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

func (binary *BinaryOperation) replaceSource(oldVal Value, newVal Value) {
	replaceCount := 0
	if binary.Src1 == oldVal {
		binary.Src1 = newVal
		replaceCount++
	}
	if binary.Src2 == oldVal {
		binary.Src2 = newVal
		replaceCount++
	}

	if replaceCount != 1 {
		panic("should never happen")
	}
}

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
	case Add, Sub, Mul, Div, Rem, Xor, Or, And, Shl, Shr: // ok
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
	Args []Value
}

var _ Instruction = &FuncCall{}
var _ Validator = &FuncCall{}

func (call *FuncCall) replaceSource(oldVal Value, newVal Value) {
	replaceCount := 0
	for idx, src := range call.Args {
		if src == oldVal {
			call.Args[idx] = newVal
			replaceCount++
		}
	}

	if replaceCount != 1 {
		panic("should never happen")
	}
}

func (call *FuncCall) Sources() []Value {
	return append([]Value{call.Func}, call.Args...)
}

func (call *FuncCall) Destination() *RegisterDefinition {
	return call.Dest
}

func (call *FuncCall) Walk(visitor Visitor) {
	visitor.Enter(call)
	call.Dest.Walk(visitor)
	call.Func.Walk(visitor)
	for _, src := range call.Args {
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

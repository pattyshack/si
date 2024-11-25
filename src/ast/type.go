package ast

import (
	"github.com/pattyshack/gt/parseutil"
)

type Type interface {
	Node
	isTypeExpr()

	String() string
}

type isType struct{}

func (isType) isTypeExpr() {}

// Internal use only.  Used by type checker to indicate an definition with
// unspecified/inferred type failed type checking.
type ErrorType struct {
	isType
	parseutil.StartEndPos
}

func NewErrorType(pos parseutil.StartEndPos) ErrorType {
	return ErrorType{
		StartEndPos: pos,
	}
}

func (t ErrorType) Walk(visitor Visitor) {
	visitor.Enter(t)
	visitor.Exit(t)
}

func (ErrorType) String() string {
	return "ErrorType"
}

// Internal use only. Compatible with all sign/unsigned int types.
type IntLiteralType struct {
	isType
	parseutil.StartEndPos
}

func (t IntLiteralType) Walk(visitor Visitor) {
	visitor.Enter(t)
	visitor.Exit(t)
}

func (IntLiteralType) String() string {
	return "IntLiteralType"
}

// Internal use only. Compatible with all sign/unsigned float types.
type FloatLiteralType struct {
	isType
	parseutil.StartEndPos
}

func (t FloatLiteralType) Walk(visitor Visitor) {
	visitor.Enter(t)
	visitor.Exit(t)
}

func (FloatLiteralType) String() string {
	return "FloatLiteralType"
}

func validateUsableType(typeExpr Type, emitter *parseutil.Emitter) {
	switch typeExpr.(type) {
	case ErrorType:
		emitter.Emit(typeExpr.Loc(), "cannot use ErrorType as return type")
	case IntLiteralType:
		emitter.Emit(typeExpr.Loc(), "cannot use IntLiteralType as return type")
	case FloatLiteralType:
		emitter.Emit(typeExpr.Loc(), "cannot use FloatLiteralType as return type")
	default: // ok
	}
}

type NumberTypeKind string

const (
	I8  = NumberTypeKind("I8")
	I16 = NumberTypeKind("I16")
	I32 = NumberTypeKind("I32")
	I64 = NumberTypeKind("I64")

	U8  = NumberTypeKind("U8")
	U16 = NumberTypeKind("U16")
	U32 = NumberTypeKind("U32")
	U64 = NumberTypeKind("U64")

	F32 = NumberTypeKind("F32")
	F64 = NumberTypeKind("F64")
)

type NumberType struct {
	isType
	parseutil.StartEndPos

	Kind NumberTypeKind
}

var _ Type = NumberType{}
var _ Validator = NumberType{}

func (numType NumberType) Walk(visitor Visitor) {
	visitor.Enter(numType)
	visitor.Exit(numType)
}

func (numType NumberType) Validate(emitter *parseutil.Emitter) {
	switch numType.Kind {
	case I8, I16, I32, I64, U8, U16, U32, U64, F32, F64: // ok
	default:
		emitter.Emit(numType.Loc(), "unexpected number type (%s)", numType.Kind)
	}
}

func (numType NumberType) String() string {
	return string(numType.Kind)
}

type FunctionType struct {
	isType
	parseutil.StartEndPos

	ReturnType     Type
	ParameterTypes []Type
}

var _ Type = FunctionType{}
var _ Validator = FunctionType{}

func (funcType FunctionType) Walk(visitor Visitor) {
	visitor.Enter(funcType)
	funcType.ReturnType.Walk(visitor)
	for _, param := range funcType.ParameterTypes {
		param.Walk(visitor)
	}
	visitor.Exit(funcType)
}

func (funcType FunctionType) Validate(emitter *parseutil.Emitter) {
	validateUsableType(funcType.ReturnType, emitter)
	for _, paramType := range funcType.ParameterTypes {
		validateUsableType(paramType, emitter)
	}
}

func (funcType FunctionType) String() string {
	result := "func("
	for idx, param := range funcType.ParameterTypes {
		if idx == 0 {
			result += param.String()
		} else {
			result += ", " + param.String()
		}
	}
	result += ") " + funcType.ReturnType.String()
	return result
}

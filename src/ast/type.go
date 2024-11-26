package ast

import (
	"github.com/pattyshack/gt/parseutil"
)

type Type interface {
	Node
	isTypeExpr()

	String() string

	Equals(Type) bool

	// type1.IsSubTypeOf(type2) returns true if a type1 value can be used as a
	// type2 value.
	IsSubTypeOf(Type) bool
}

type isType struct{}

func (isType) isTypeExpr() {}

func IsErrorType(t Type) bool {
	_, ok := t.(ErrorType)
	return ok
}

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

func (ErrorType) Equals(Type) bool {
	return false
}

func (ErrorType) IsSubTypeOf(Type) bool {
	return false
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

func (IntLiteralType) Equals(other Type) bool {
	_, ok := other.(IntLiteralType)
	return ok
}

func (IntLiteralType) IsSubTypeOf(other Type) bool {
	switch other.(type) {
	case IntLiteralType:
		return true
	case IntType:
		return true
	default:
		return false
	}
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

func (FloatLiteralType) Equals(other Type) bool {
	_, ok := other.(FloatLiteralType)
	return ok
}

func (FloatLiteralType) IsSubTypeOf(other Type) bool {
	switch other.(type) {
	case FloatLiteralType:
		return true
	case FloatType:
		return true
	default:
		return false
	}
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

type IntTypeKind string

const (
	I8  = IntTypeKind("I8")
	I16 = IntTypeKind("I16")
	I32 = IntTypeKind("I32")
	I64 = IntTypeKind("I64")

	U8  = IntTypeKind("U8")
	U16 = IntTypeKind("U16")
	U32 = IntTypeKind("U32")
	U64 = IntTypeKind("U64")
)

type IntType struct {
	isType
	parseutil.StartEndPos

	Kind IntTypeKind
}

var _ Type = IntType{}
var _ Validator = IntType{}

func (intType IntType) Walk(visitor Visitor) {
	visitor.Enter(intType)
	visitor.Exit(intType)
}

func (intType IntType) Validate(emitter *parseutil.Emitter) {
	switch intType.Kind {
	case I8, I16, I32, I64, U8, U16, U32, U64: // ok
	default:
		emitter.Emit(intType.Loc(), "unexpected int type (%s)", intType.Kind)
	}
}

func (intType IntType) String() string {
	return string(intType.Kind)
}

func (intType IntType) Equals(other Type) bool {
	otherType, ok := other.(IntType)
	if !ok {
		return false
	}

	return intType.Kind == otherType.Kind
}

func (intType IntType) IsSubTypeOf(other Type) bool {
	// Int types must be explicitly converted.
	return intType.Equals(other)
}

type FloatTypeKind string

const (
	F32 = FloatTypeKind("F32")
	F64 = FloatTypeKind("F64")
)

type FloatType struct {
	isType
	parseutil.StartEndPos

	Kind FloatTypeKind
}

var _ Type = FloatType{}
var _ Validator = FloatType{}

func (floatType FloatType) Walk(visitor Visitor) {
	visitor.Enter(floatType)
	visitor.Exit(floatType)
}

func (floatType FloatType) Validate(emitter *parseutil.Emitter) {
	switch floatType.Kind {
	case F32, F64: // ok
	default:
		emitter.Emit(floatType.Loc(), "unexpected float type (%s)", floatType.Kind)
	}
}

func (floatType FloatType) String() string {
	return string(floatType.Kind)
}

func (floatType FloatType) Equals(other Type) bool {
	otherType, ok := other.(FloatType)
	if !ok {
		return false
	}

	return floatType.Kind == otherType.Kind
}

func (floatType FloatType) IsSubTypeOf(other Type) bool {
	// Float types must be explicitly converted.
	return floatType.Equals(other)
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

func (funcType FunctionType) Equals(other Type) bool {
	otherFuncType, ok := other.(FunctionType)
	if !ok {
		return false
	}

	if len(funcType.ParameterTypes) != len(otherFuncType.ParameterTypes) {
		return false
	}

	for idx, paramType := range funcType.ParameterTypes {
		otherParamType := otherFuncType.ParameterTypes[idx]
		if !paramType.Equals(otherParamType) {
			return false
		}
	}

	return funcType.ReturnType.Equals(otherFuncType.ReturnType)
}

func (funcType FunctionType) IsSubTypeOf(other Type) bool {
	return funcType.Equals(other)
}

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

func IsSignedIntSubType(t Type) bool {
	switch t.(type) {
	case PositiveIntLiteralType:
		return true
	case NegativeIntLiteralType:
		return true
	case SignedIntType:
		return true
	default:
		return false
	}
}

// signed/unsigned int or int literal
func IsIntSubType(t Type) bool {
	switch t.(type) {
	case PositiveIntLiteralType:
		return true
	case NegativeIntLiteralType:
		return true
	case SignedIntType:
		return true
	case UnsignedIntType:
		return true
	default:
		return false
	}
}

// float or float literal
func IsFloatSubType(t Type) bool {
	switch t.(type) {
	case FloatLiteralType:
		return true
	case FloatType:
		return true
	default:
		return false
	}
}

func IsNumberSubType(t Type) bool {
	return IsIntSubType(t) || IsFloatSubType(t)
}

// == and !=
// NOTE: float is not comparable
func IsComparableType(t Type) bool {
	switch t.(type) {
	case PositiveIntLiteralType:
		return true
	case NegativeIntLiteralType:
		return true
	case SignedIntType:
		return true
	case UnsignedIntType:
		return true
	case FunctionType:
		return true
	default:
		return false
	}
}

// < and >=
func IsOrderedType(t Type) bool {
	switch t.(type) {
	case PositiveIntLiteralType:
		return true
	case NegativeIntLiteralType:
		return true
	case SignedIntType:
		return true
	case UnsignedIntType:
		return true
	case FloatLiteralType:
		return true
	case FloatType:
		return true
	default:
		return false
	}
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

// Internal use only. Compatible with all signed/unsigned int types.
type PositiveIntLiteralType struct {
	isType
	parseutil.StartEndPos
}

func (t PositiveIntLiteralType) Walk(visitor Visitor) {
	visitor.Enter(t)
	visitor.Exit(t)
}

func (PositiveIntLiteralType) String() string {
	return "PositiveIntLiteralType"
}

func (PositiveIntLiteralType) Equals(other Type) bool {
	_, ok := other.(PositiveIntLiteralType)
	return ok
}

func (PositiveIntLiteralType) IsSubTypeOf(other Type) bool {
	switch other.(type) {
	case PositiveIntLiteralType:
		return true
	case NegativeIntLiteralType:
		return true
	case SignedIntType:
		return true
	case UnsignedIntType:
		return true
	default:
		return false
	}
}

// Internal use only. Compatible with all signed int types.
type NegativeIntLiteralType struct {
	isType
	parseutil.StartEndPos
}

func (t NegativeIntLiteralType) Walk(visitor Visitor) {
	visitor.Enter(t)
	visitor.Exit(t)
}

func (NegativeIntLiteralType) String() string {
	return "NegativeIntLiteralType"
}

func (NegativeIntLiteralType) Equals(other Type) bool {
	_, ok := other.(NegativeIntLiteralType)
	return ok
}

func (NegativeIntLiteralType) IsSubTypeOf(other Type) bool {
	switch other.(type) {
	case NegativeIntLiteralType:
		return true
	case SignedIntType:
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
	case PositiveIntLiteralType:
		emitter.Emit(
			typeExpr.Loc(),
			"cannot use PositiveIntLiteralType as return type")
	case NegativeIntLiteralType:
		emitter.Emit(
			typeExpr.Loc(),
			"cannot use NegativeIntLiteralType as return type")
	case FloatLiteralType:
		emitter.Emit(typeExpr.Loc(), "cannot use FloatLiteralType as return type")
	default: // ok
	}
}

type SignedIntTypeKind string

const (
	I8  = SignedIntTypeKind("I8")
	I16 = SignedIntTypeKind("I16")
	I32 = SignedIntTypeKind("I32")
	I64 = SignedIntTypeKind("I64")
)

type SignedIntType struct {
	isType
	parseutil.StartEndPos

	Kind SignedIntTypeKind
}

func NewI8(pos parseutil.StartEndPos) Type {
	return SignedIntType{
		StartEndPos: pos,
		Kind:        I8,
	}
}

func NewI16(pos parseutil.StartEndPos) Type {
	return SignedIntType{
		StartEndPos: pos,
		Kind:        I16,
	}
}

func NewI32(pos parseutil.StartEndPos) Type {
	return SignedIntType{
		StartEndPos: pos,
		Kind:        I32,
	}
}

func NewI64(pos parseutil.StartEndPos) Type {
	return SignedIntType{
		StartEndPos: pos,
		Kind:        I64,
	}
}

var _ Type = SignedIntType{}
var _ Validator = SignedIntType{}

func (intType SignedIntType) Walk(visitor Visitor) {
	visitor.Enter(intType)
	visitor.Exit(intType)
}

func (intType SignedIntType) Validate(emitter *parseutil.Emitter) {
	switch intType.Kind {
	case I8, I16, I32, I64: // ok
	default:
		emitter.Emit(intType.Loc(), "unexpected signed int type (%s)", intType.Kind)
	}
}

func (intType SignedIntType) String() string {
	return string(intType.Kind)
}

func (intType SignedIntType) Equals(other Type) bool {
	otherType, ok := other.(SignedIntType)
	if !ok {
		return false
	}

	return intType.Kind == otherType.Kind
}

func (intType SignedIntType) IsSubTypeOf(other Type) bool {
	// Int types must be explicitly converted.
	return intType.Equals(other)
}

type UnsignedIntTypeKind string

const (
	U8  = UnsignedIntTypeKind("U8")
	U16 = UnsignedIntTypeKind("U16")
	U32 = UnsignedIntTypeKind("U32")
	U64 = UnsignedIntTypeKind("U64")
)

type UnsignedIntType struct {
	isType
	parseutil.StartEndPos

	Kind UnsignedIntTypeKind
}

func NewU8(pos parseutil.StartEndPos) Type {
	return UnsignedIntType{
		StartEndPos: pos,
		Kind:        U8,
	}
}

func NewU16(pos parseutil.StartEndPos) Type {
	return UnsignedIntType{
		StartEndPos: pos,
		Kind:        U16,
	}
}

func NewU32(pos parseutil.StartEndPos) Type {
	return UnsignedIntType{
		StartEndPos: pos,
		Kind:        U32,
	}
}

func NewU64(pos parseutil.StartEndPos) Type {
	return UnsignedIntType{
		StartEndPos: pos,
		Kind:        U64,
	}
}

var _ Type = UnsignedIntType{}
var _ Validator = UnsignedIntType{}

func (intType UnsignedIntType) Walk(visitor Visitor) {
	visitor.Enter(intType)
	visitor.Exit(intType)
}

func (intType UnsignedIntType) Validate(emitter *parseutil.Emitter) {
	switch intType.Kind {
	case U8, U16, U32, U64: // ok
	default:
		emitter.Emit(
			intType.Loc(),
			"unexpected unsigned int type (%s)",
			intType.Kind)
	}
}

func (intType UnsignedIntType) String() string {
	return string(intType.Kind)
}

func (intType UnsignedIntType) Equals(other Type) bool {
	otherType, ok := other.(UnsignedIntType)
	if !ok {
		return false
	}

	return intType.Kind == otherType.Kind
}

func (intType UnsignedIntType) IsSubTypeOf(other Type) bool {
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

func NewF32(pos parseutil.StartEndPos) Type {
	return FloatType{
		StartEndPos: pos,
		Kind:        F32,
	}
}

func NewF64(pos parseutil.StartEndPos) Type {
	return FloatType{
		StartEndPos: pos,
		Kind:        F64,
	}
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

	CallConvention

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
	if !funcType.CallConvention.isValid() {
		emitter.Emit(
			funcType.Loc(),
			"unsupported call convention (%s)",
			funcType.CallConvention)
	}

	validateUsableType(funcType.ReturnType, emitter)
	for _, paramType := range funcType.ParameterTypes {
		validateUsableType(paramType, emitter)
	}
}

func (funcType FunctionType) String() string {
	result := "func{" + string(funcType.CallConvention) + "}("
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

	if funcType.CallConvention != otherFuncType.CallConvention {
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

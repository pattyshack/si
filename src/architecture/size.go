package architecture

import (
	"github.com/pattyshack/chickadee/ast"
)

const (
	// Assumption: we only support 64 bit architecture.
	RegisterByteSize = 8
	AddressByteSize  = RegisterByteSize
)

func ByteSize(valType ast.Type) int {
	// TODO: Deal with variable pointer types.  For now, we'll allow return
	// address to have unspecified type.
	if valType == nil {
		return AddressByteSize
	}

	switch valueType := valType.(type) {
	case ast.ErrorType:
		panic("error type has no size")
	case ast.PositiveIntLiteralType:
		panic("positive int literal type has no size")
	case ast.NegativeIntLiteralType:
		panic("negative int literal type has no size")
	case ast.FloatLiteralType:
		panic("float literal type has no size")
	case ast.SignedIntType:
		switch valueType.Kind {
		case ast.I8:
			return 1
		case ast.I16:
			return 2
		case ast.I32:
			return 4
		case ast.I64:
			return 8
		default:
			panic("should never reach here")
		}
	case ast.UnsignedIntType:
		switch valueType.Kind {
		case ast.U8:
			return 1
		case ast.U16:
			return 2
		case ast.U32:
			return 4
		case ast.U64:
			return 8
		default:
			panic("should never reach here")
		}
	case ast.FloatType:
		switch valueType.Kind {
		case ast.F32:
			return 4
		case ast.F64:
			return 8
		default:
			panic("should never reach here")
		}
	case ast.FunctionType:
		return AddressByteSize
	default:
		panic("unhandled type")
	}
}

func NumRegisters(valType ast.Type) int {
	byteSize := ByteSize(valType)
	return (byteSize + RegisterByteSize - 1) / RegisterByteSize
}

func AlignedSize(valType ast.Type) int {
	return NumRegisters(valType) * RegisterByteSize
}

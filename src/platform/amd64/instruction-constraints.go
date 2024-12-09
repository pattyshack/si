package amd64

import (
	"github.com/pattyshack/chickadee/architecture"
	"github.com/pattyshack/chickadee/ast"
)

var (
	// Unconditional jump has no constraints
	jumpConstraints = architecture.NewInstructionConstraints()

	intConditionalJumpConstraints = newConditionalJumpConstraints(
		RegisterSet.General)
	floatConditionalJumpConstraints = newConditionalJumpConstraints(
		RegisterSet.Float)

	intUnaryOpConstraints   = newUnaryOpConstraints(RegisterSet.General)
	floatUnaryOpConstraints = newUnaryOpConstraints(RegisterSet.Float)

	floatToIntConstraints = newConversionUnaryOpConstraints(
		RegisterSet.Float,
		RegisterSet.General)
	intToFloatConstraints = newConversionUnaryOpConstraints(
		RegisterSet.General,
		RegisterSet.Float)

	intBinaryOpConstraints = newBinaryOpConstraints(
		RegisterSet.General)
	floatBinaryOpConstraints = newBinaryOpConstraints(RegisterSet.Float)
)

// nil indicates the value should be in memory.  Otherwise, the return
// list indicates the number of registers needed, and the corresponding class
// of registers to choose form.
func getRegisterClasses(
	valueType ast.Type,
) [][]*architecture.Register {
	switch valueType.(type) {
	case ast.ErrorType:
		panic("should never happen")
	case ast.PositiveIntLiteralType:
		panic("should never happen")
	case ast.NegativeIntLiteralType:
		panic("should never happen")
	case ast.FloatLiteralType:
		panic("should never happen")

	case ast.SignedIntType:
		return [][]*architecture.Register{RegisterSet.General}
	case ast.UnsignedIntType:
		return [][]*architecture.Register{RegisterSet.General}
	case ast.FunctionType:
		return [][]*architecture.Register{RegisterSet.General}

	case ast.FloatType:
		return [][]*architecture.Register{RegisterSet.Float}

	default:
		panic("unhandled type")
	}
}

func newCopyOpConstraints(
	valueType ast.Type,
) *architecture.InstructionConstraints {
	constraints := architecture.NewInstructionConstraints()

	constraints.AddAnySource(valueType.Size())

	classes := getRegisterClasses(valueType)
	if classes == nil {
		constraints.SetStackDestination(valueType.Size())
	} else {
		dest := []*architecture.RegisterCandidate{}
		for _, class := range classes {
			dest = append(dest, constraints.Select(true, class...))
		}
		constraints.SetRegisterDestination(dest...)
	}

	return constraints
}

func newConditionalJumpConstraints(
	candidates []*architecture.Register,
) *architecture.InstructionConstraints {
	constraints := architecture.NewInstructionConstraints()

	// Conditional jump compare two source registers without clobbering them.
	// There's no destination register.
	constraints.AddRegisterSource(constraints.Select(false, candidates...))
	constraints.AddRegisterSource(constraints.Select(false, candidates...))

	return constraints
}

func newUnaryOpConstraints(
	candidates []*architecture.Register,
) *architecture.InstructionConstraints {
	constraints := architecture.NewInstructionConstraints()

	// Destination reuses the source register.
	reg := constraints.Select(true, candidates...)
	constraints.AddRegisterSource(reg)
	constraints.SetRegisterDestination(reg)

	return constraints
}

func newConversionUnaryOpConstraints(
	fromCandidates []*architecture.Register,
	toCandidates []*architecture.Register,
) *architecture.InstructionConstraints {
	constraints := architecture.NewInstructionConstraints()

	constraints.AddRegisterSource(constraints.Select(false, fromCandidates...))
	constraints.SetRegisterDestination(constraints.Select(true, toCandidates...))

	return constraints
}

func newBinaryOpConstraints(
	candidates []*architecture.Register,
) *architecture.InstructionConstraints {
	constraints := architecture.NewInstructionConstraints()

	// Destination reuses the first source register, the second source register is
	// not clobbered.
	src1 := constraints.Select(true, candidates...)
	constraints.AddRegisterSource(src1)
	constraints.SetRegisterDestination(src1)
	constraints.AddRegisterSource(constraints.Select(false, candidates...))

	return constraints
}

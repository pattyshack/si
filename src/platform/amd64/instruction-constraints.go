package amd64

import (
	"github.com/pattyshack/chickadee/architecture"
	"github.com/pattyshack/chickadee/ast"
)

var (
	// Unconditional jump has no constraints
	jumpConstraints = architecture.NewInstructionConstraints()

	intConditionalJumpConstraints   = newConditionalJumpConstraints(false)
	floatConditionalJumpConstraints = newConditionalJumpConstraints(true)

	intUnaryOpConstraints   = newUnaryOpConstraints(false)
	floatUnaryOpConstraints = newUnaryOpConstraints(true)

	floatToIntConstraints = newConversionUnaryOpConstraints(true)
	intToFloatConstraints = newConversionUnaryOpConstraints(false)

	intBinaryOpConstraints   = newBinaryOpConstraints(false)
	floatBinaryOpConstraints = newBinaryOpConstraints(true)
)

// nil indicates the value should be in memory.  Otherwise, the return
// list indicates the number of registers needed; true indicates any float
// register while false indicates any general register.
func getRegisterClasses(
	valueType ast.Type,
) []bool {
	switch valueType.(type) {
	case *ast.ErrorType:
		panic("should never happen")
	case *ast.PositiveIntLiteralType:
		panic("should never happen")
	case *ast.NegativeIntLiteralType:
		panic("should never happen")
	case *ast.FloatLiteralType:
		panic("should never happen")

	case *ast.SignedIntType:
		return []bool{false}
	case *ast.UnsignedIntType:
		return []bool{false}
	case *ast.FunctionType:
		return []bool{false}

	case *ast.FloatType:
		return []bool{true}

	default:
		panic("unhandled type")
	}
}

func newCopyOpConstraints(
	valueType ast.Type,
) *architecture.InstructionConstraints {
	constraints := architecture.NewInstructionConstraints()

	constraints.AddAnySource(valueType)

	classes := getRegisterClasses(valueType)
	if classes == nil {
		constraints.SetStackDestination(valueType)
	} else {
		dest := []*architecture.RegisterCandidate{}
		for _, anyFloat := range classes {
			if anyFloat {
				dest = append(dest, constraints.SelectAnyFloat(true))
			} else {
				dest = append(dest, constraints.SelectAnyGeneral(true))
			}
		}
		constraints.SetRegisterDestination(dest...)
	}

	return constraints
}

func newConditionalJumpConstraints(
	isFloat bool,
) *architecture.InstructionConstraints {
	constraints := architecture.NewInstructionConstraints()

	// Conditional jump compare two source registers without clobbering them.
	// There's no destination register.
	if isFloat {
		constraints.AddRegisterSource(constraints.SelectAnyFloat(false))
		constraints.AddRegisterSource(constraints.SelectAnyFloat(false))
	} else {
		constraints.AddRegisterSource(constraints.SelectAnyGeneral(false))
		constraints.AddRegisterSource(constraints.SelectAnyGeneral(false))
	}

	return constraints
}

func newUnaryOpConstraints(
	isFloat bool,
) *architecture.InstructionConstraints {
	constraints := architecture.NewInstructionConstraints()

	// Destination reuses the source register.
	var reg *architecture.RegisterCandidate
	if isFloat {
		reg = constraints.SelectAnyFloat(true)
	} else {
		reg = constraints.SelectAnyGeneral(true)
	}
	constraints.AddRegisterSource(reg)
	constraints.SetRegisterDestination(reg)

	return constraints
}

func newConversionUnaryOpConstraints(
	fromFloat bool,
) *architecture.InstructionConstraints {
	constraints := architecture.NewInstructionConstraints()

	if fromFloat {
		constraints.AddRegisterSource(constraints.SelectAnyFloat(false))
		constraints.SetRegisterDestination(constraints.SelectAnyGeneral(true))
	} else {
		constraints.AddRegisterSource(constraints.SelectAnyGeneral(false))
		constraints.SetRegisterDestination(constraints.SelectAnyFloat(true))
	}

	return constraints
}

func newBinaryOpConstraints(
	isFloat bool,
) *architecture.InstructionConstraints {
	constraints := architecture.NewInstructionConstraints()

	selectAny := constraints.SelectAnyGeneral
	if isFloat {
		selectAny = constraints.SelectAnyFloat
	}

	// Destination reuses the first source register, the second source register is
	// not clobbered.
	src1 := selectAny(true)
	constraints.AddRegisterSource(src1)
	constraints.SetRegisterDestination(src1)
	constraints.AddRegisterSource(selectAny(false))

	return constraints
}

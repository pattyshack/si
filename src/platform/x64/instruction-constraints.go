package x64

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

	genericIntBinaryOpConstraints   = newGenericBinaryOpConstraints(false)
	genericFloatBinaryOpConstraints = newGenericBinaryOpConstraints(true)

	divConstraints = newDivRemConstraints(false)
	remConstraints = newDivRemConstraints(true)
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
	constraints.AddAnyCopySource(valueType)
	constraints.SetAnyCopyDestination(valueType)

	return constraints
}

func newConditionalJumpConstraints(
	isFloat bool,
) *architecture.InstructionConstraints {
	constraints := architecture.NewInstructionConstraints()

	// Conditional jump compare two source registers without clobbering them.
	// There's no destination register.
	if isFloat {
		constraints.AddRegisterSource(false, constraints.SelectAnyFloat(false))
		constraints.AddRegisterSource(false, constraints.SelectAnyFloat(false))
	} else {
		constraints.AddRegisterSource(false, constraints.SelectAnyGeneral(false))
		constraints.AddRegisterSource(false, constraints.SelectAnyGeneral(false))
	}

	return constraints
}

func newUnaryOpConstraints(
	isFloat bool,
) *architecture.InstructionConstraints {
	constraints := architecture.NewInstructionConstraints()

	// Destination reuses the source register.
	var reg *architecture.RegisterConstraint
	if isFloat {
		reg = constraints.SelectAnyFloat(true)
	} else {
		reg = constraints.SelectAnyGeneral(true)
	}

	// TODO support encoded immediate?
	constraints.AddRegisterSource(false, reg)
	constraints.SetRegisterDestination(reg)

	return constraints
}

func newConversionUnaryOpConstraints(
	fromFloat bool,
) *architecture.InstructionConstraints {
	constraints := architecture.NewInstructionConstraints()

	// TODO support encoded immediate?
	if fromFloat {
		constraints.AddRegisterSource(false, constraints.SelectAnyFloat(false))
		constraints.SetRegisterDestination(constraints.SelectAnyGeneral(true))
	} else {
		constraints.AddRegisterSource(false, constraints.SelectAnyGeneral(false))
		constraints.SetRegisterDestination(constraints.SelectAnyFloat(true))
	}

	return constraints
}

func newGenericBinaryOpConstraints(
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
	constraints.AddRegisterSource(false, src1)
	constraints.SetRegisterDestination(src1)
	constraints.AddRegisterSource(true, selectAny(false))

	return constraints
}

// x64's div/idiv is retarded
func newDivRemConstraints(
	isRem bool,
) *architecture.InstructionConstraints {
	constraints := architecture.NewInstructionConstraints()

	// rdx:rax forms a double quad word
	upper := constraints.Require(true, rdx)
	lower := constraints.Require(true, rax)

	constraints.AddRegisterSource(false, lower)
	constraints.AddRegisterSource(false, constraints.SelectAnyGeneral(false))

	if isRem {
		constraints.SetRegisterDestination(upper)
	} else { // div
		constraints.SetRegisterDestination(lower)
	}

	return constraints
}

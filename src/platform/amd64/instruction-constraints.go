package amd64

import (
	"github.com/pattyshack/chickadee/architecture"
)

var (
	// Unconditional jump has no constraints
	jumpConstraints = architecture.NewInstructionConstraints()

	intConditionalJumpConstraints = newConditionalJumpConstraints(
		RegisterSet.General)
	floatConditionalJumpConstraints = newConditionalJumpConstraints(
		RegisterSet.Float)

	intAssignOpConstraint = newAssignOpConstraints(
		RegisterSet.General)
	floatAssignOpConstraint = newAssignOpConstraints(RegisterSet.Float)

	intUnaryOpConstraints   = newUnaryOpConstraints(RegisterSet.General)
	floatUnaryOpConstraints = newUnaryOpConstraints(RegisterSet.Float)

	intBinaryOpConstraints = newBinaryOpConstraints(
		RegisterSet.General)
	floatBinaryOpConstraints = newBinaryOpConstraints(RegisterSet.Float)

	// TODO func call / ret constraints
)

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

func newAssignOpConstraints(
	candidates []*architecture.Register,
) *architecture.InstructionConstraints {
	constraints := architecture.NewInstructionConstraints()

	// Copy from source to destination without clobbering the source register.
	constraints.AddRegisterSource(constraints.Select(false, candidates...))
	constraints.SetRegisterDestination(constraints.Select(true, candidates...))

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

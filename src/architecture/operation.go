package architecture

import (
	"github.com/pattyshack/chickadee/ast"
)

type OperationKind string

const (
	// Note: ast.CopyOp and ast.Phi are never used as ExecuteInstruction's
	// parameter. The allocator will emit CopyLocation operations instead.
	ExecuteInstruction = OperationKind("ExecuteInstruction")

	// The following are allocator generated operations.

	PushStackFrame = OperationKind("PushStackFrame")
	PopStackFrame  = OperationKind("PopStackFrame")

	MoveRegister     = OperationKind("MoveRegister")
	CopyLocation     = OperationKind("CopyLocation")
	SetConstantValue = OperationKind("SetConstantValue")
	// Only used for zero-ing temp stack destination
	InitializeZeros = OperationKind("InitializeZeros")

	// The following are allocator generated debugging operations.  They should
	// not emit any platform instruction.
	AllocateLocation           = OperationKind("AllocateLocation")
	FreeLocation               = OperationKind("FreeLocation")
	AssignLocationToDefinition = OperationKind("AssignLocationToDefinition")
)

type Operation struct {
	Kind OperationKind

	// Used by all operations except PushStackFrame, PopStackFrame, and
	// MoveRegister
	Destination *DataLocation

	// Used by ExecuteInstruction, and CopyLocation.
	Sources []*DataLocation

	// Used by AssignLocationToDefinition
	*ast.VariableDefinition

	// Used by ExecuteInstruction
	ast.Instruction

	// Used by SetConstantValue
	ast.Value

	// Used by PushStackFrame / PopStackFrame
	*StackFrame

	// Used by MoveRegister
	SrcRegister *Register

	// Used by MoveRegister, and by SetConstantValue/InitializeZeros/CopyLocation
	// as a temp register when locations are on stack.
	DestRegister *Register
}

func NewExecuteInstructionOp(
	inst ast.Instruction,
	srcs []*DataLocation,
	dest *DataLocation,
) Operation {
	copiedSrcs := make([]*DataLocation, len(srcs))
	for idx, src := range srcs {
		copiedSrcs[idx] = src.Copy()
	}

	if dest != nil {
		dest = dest.Copy()
	}

	return Operation{
		Kind:        ExecuteInstruction,
		Destination: dest,
		Sources:     copiedSrcs,
		Instruction: inst,
	}
}

func NewPushStackFrameOp(frame *StackFrame) Operation {
	return Operation{
		Kind:       PushStackFrame,
		StackFrame: frame,
	}
}

func NewPopStackFrameOp(frame *StackFrame) Operation {
	return Operation{
		Kind:       PopStackFrame,
		StackFrame: frame,
	}
}

func NewMoveRegisterOp(src *Register, dest *Register) Operation {
	return Operation{
		Kind:         MoveRegister,
		SrcRegister:  src,
		DestRegister: dest,
	}
}

func NewCopyLocationOp(
	src *DataLocation,
	dest *DataLocation,
	temp *Register,
) Operation {
	return Operation{
		Kind:         CopyLocation,
		Sources:      []*DataLocation{src.Copy()},
		Destination:  dest.Copy(),
		DestRegister: temp,
	}
}

func NewSetConstantValueOp(
	value ast.Value,
	dest *DataLocation,
	temp *Register,
) Operation {
	return Operation{
		Kind:         SetConstantValue,
		Destination:  dest.Copy(),
		Value:        value,
		DestRegister: temp,
	}
}

func NewInitializeZerosOp(
	dest *DataLocation,
	temp *Register,
) Operation {
	return Operation{
		Kind:         InitializeZeros,
		Destination:  dest.Copy(),
		DestRegister: temp,
	}
}

func NewAllocateLocationOp(loc *DataLocation) Operation {
	return Operation{
		Kind:        AllocateLocation,
		Destination: loc.Copy(),
	}
}

func NewFreeLocationOp(loc *DataLocation) Operation {
	return Operation{
		Kind:        FreeLocation,
		Destination: loc.Copy(),
	}
}

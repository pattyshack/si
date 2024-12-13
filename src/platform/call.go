package platform

import (
	"github.com/pattyshack/chickadee/architecture"
	"github.com/pattyshack/chickadee/ast"
)

// Call convention specific, os/architecture-independent, type specification
// used by the type checker.
type CallTypeSpec interface {
	IsValidArgType(ast.Type) bool

	IsValidReturnType(ast.Type) bool
}

type CallSpec interface {
	CallTypeSpec

	// Used by both call and ret instructions.
	CallRetConstraints(
		ast.FunctionType,
	) *architecture.CallConvention
}

type InternalCallTypeSpec struct{}

func (InternalCallTypeSpec) IsValidArgType(ast.Type) bool    { return true }
func (InternalCallTypeSpec) IsValidReturnType(ast.Type) bool { return true }

func isPrimitiveType(t ast.Type) bool {
	if ast.IsIntSubType(t) || ast.IsFloatSubType(t) {
		return true
	}

	// TODO pointer
	return false
}

type SystemVLiteCallTypeSpec struct {
}

func (SystemVLiteCallTypeSpec) IsValidArgType(t ast.Type) bool {
	return isPrimitiveType(t)
}

func (SystemVLiteCallTypeSpec) IsValidReturnType(t ast.Type) bool {
	return isPrimitiveType(t)
}

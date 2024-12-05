package platform

import (
	"github.com/pattyshack/chickadee/ast"
)

// Call convention specific, os/architecture-independent, type specification
// used by the type checker.
type CallTypeSpec interface {
	IsValidArgType(ast.Type) bool

	IsValidReturnType(ast.Type) bool
}

func NewCallTypeSpec(convention ast.CallConvention) CallTypeSpec {
	switch convention {
	case ast.InternalCallConvention:
		return noOpCallTypeSpec{}
	case ast.SystemVLiteCallConvention:
		return systemVLiteCallTypeSpec{}
	default:
		// This is an ast syntax error.  The error is emitted by Validate.
		return noOpCallTypeSpec{}
	}
}

type noOpCallTypeSpec struct{}

func (noOpCallTypeSpec) IsValidArgType(ast.Type) bool    { return true }
func (noOpCallTypeSpec) IsValidReturnType(ast.Type) bool { return true }

func isPrimitiveType(t ast.Type) bool {
	if ast.IsIntSubType(t) || ast.IsFloatSubType(t) {
		return true
	}

	// TODO pointer
	return false
}

type systemVLiteCallTypeSpec struct {
}

func (systemVLiteCallTypeSpec) IsValidArgType(t ast.Type) bool {
	return isPrimitiveType(t)
}

func (systemVLiteCallTypeSpec) IsValidReturnType(t ast.Type) bool {
	return isPrimitiveType(t)
}

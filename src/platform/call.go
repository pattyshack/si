package platform

import (
	"sync"

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
	Name() ast.CallConventionName

	CallTypeSpec

	// Used by both call and ret instructions.
	CallConvention(
		*ast.FunctionType,
	) *architecture.CallConvention
}

type InternalCallTypeSpec struct{}

func (InternalCallTypeSpec) IsValidArgType(ast.Type) bool    { return true }
func (InternalCallTypeSpec) IsValidReturnType(ast.Type) bool { return true }

func isPrimitiveType(t ast.Type) bool {
	if ast.IsNumberSubType(t) {
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

type cachedCallSpec struct {
	CallSpec

	mutex sync.Mutex

	// guarded by mutex
	cache map[*ast.FunctionType]*architecture.CallConvention
}

func newCachedCallSpec(
	spec CallSpec,
) *cachedCallSpec {
	return &cachedCallSpec{
		CallSpec: spec,
		cache:    map[*ast.FunctionType]*architecture.CallConvention{},
	}
}

func (cached *cachedCallSpec) getConvention(
	funcType *ast.FunctionType,
) *architecture.CallConvention {
	cached.mutex.Lock()
	defer cached.mutex.Unlock()

	return cached.cache[funcType]
}

func (cached *cachedCallSpec) CallConvention(
	funcType *ast.FunctionType,
) *architecture.CallConvention {
	convention := cached.getConvention(funcType)
	if convention != nil {
		return convention
	}

	convention = cached.CallSpec.CallConvention(funcType)

	cached.mutex.Lock()
	defer cached.mutex.Unlock()

	cached.cache[funcType] = convention

	return convention
}

type CallSpecs struct {
	defaultSpec *cachedCallSpec
	specs       map[ast.CallConventionName]*cachedCallSpec
}

func NewCallSpecs(
	defaultSpec CallSpec,
	otherSpecs ...CallSpec,
) *CallSpecs {
	cachedDefault := newCachedCallSpec(defaultSpec)
	specs := map[ast.CallConventionName]*cachedCallSpec{
		cachedDefault.Name(): cachedDefault,
	}
	for _, spec := range otherSpecs {
		specs[spec.Name()] = newCachedCallSpec(spec)
	}
	return &CallSpecs{
		defaultSpec: cachedDefault,
		specs:       specs,
	}
}

func (specs *CallSpecs) CallSpec(convention ast.CallConventionName) CallSpec {
	spec, ok := specs.specs[convention]
	if ok {
		return spec
	}
	return specs.defaultSpec
}

func (specs *CallSpecs) CallConvention(
	funcType *ast.FunctionType,
) *architecture.CallConvention {
	spec := specs.CallSpec(funcType.CallConventionName)
	return spec.CallConvention(funcType)
}

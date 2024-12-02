package platform

import (
	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
)

// OS specific, (hopefully mostly) architecture-independent, syscall type
// specification, used by the type checker.
type SysCallTypeSpec interface {
	// This is usually an int type (e.g., I32)
	IsValidFuncValueType(ast.Type) bool

	// Maximum number of arguments that syscall could take.  A weak check
	MaxNumberOfArgs() int

	// This usually consist of int and pointer types.
	IsValidArgType(ast.Type) bool

	// This is usually an int type.  This must be a subset of valid arg types.
	IsValidExitArgType(ast.Type) bool

	// SysCall's return value type.  This is usually an int type.
	ReturnType(parseutil.StartEndPos) ast.Type
}

func NewSysCallTypeSpec(os OperatingSystemName) SysCallTypeSpec {
	switch os {
	case Linux:
		return LinuxSysCallTypeSpec{}
	default:
		panic("unsupported os: " + os)
	}
}

// Resources:
//
// x86-64 call convention: https://refspecs.linuxfoundation.org/elf/x86_64-abi-0.99.pdf
//
// caller-saved:  rax rcx r11
// system number: rax
// arguments:     rdi rsi rdx r10 r8 r9
// return value:  rax
//
// syscall numbers: https://chromium.googlesource.com/chromiumos/docs/+/master/constants/syscalls.md
type LinuxSysCallTypeSpec struct {
}

func (LinuxSysCallTypeSpec) IsValidFuncValueType(funcType ast.Type) bool {
	return ast.IsIntSubType(funcType)
}

func (LinuxSysCallTypeSpec) MaxNumberOfArgs() int {
	// Note: on most popular architectures (amd64 and arm64), linux accept up to
	// 6 arguments, but on other architectures, the accepted number of arguments
	// could be up to 7.  Make this architecture specific if necessary.
	return 6
}

func (LinuxSysCallTypeSpec) IsValidArgType(argType ast.Type) bool {
	if ast.IsIntSubType(argType) {
		return true
	}

	// TODO pointer
	return false
}

func (LinuxSysCallTypeSpec) IsValidExitArgType(argType ast.Type) bool {
	return ast.IsSignedIntSubType(argType)
}

func (LinuxSysCallTypeSpec) ReturnType(pos parseutil.StartEndPos) ast.Type {
	return ast.NewI32(pos)
}

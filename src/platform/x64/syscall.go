package x64

import (
	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/architecture"
	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/platform"
)

func newSysCallSpec(os platform.OperatingSystemName) platform.SysCallSpec {
	switch os {
	case platform.Linux:
		return linuxSysCallSpec{}
	default:
		panic("unsupported os: " + os)
	}
}

type linuxSysCallSpec struct {
	platform.LinuxSysCallTypeSpec
}

func (linuxSysCallSpec) ExitSysCallFuncValue(
	pos parseutil.StartEndPos,
) ast.Value {
	return ast.NewIntImmediate(pos, 60, false)
}

func newSysCallConstraints(
	os platform.OperatingSystemName,
	call *ast.FuncCall,
) *architecture.InstructionConstraints {
	switch os {
	case platform.Linux:
		return newLinuxSysCallConstraints(len(call.Args)).CallConstraints
	default:
		panic("unsupported os: " + os)
	}
}

func newLinuxSysCallConstraints(
	numArgs int,
) *architecture.CallConvention {
	// Syscall number and return value
	convention := architecture.NewCallConvention(true, rax)
	convention.SetRegisterDestination(rax)

	// Clobbered by syscall

	// Syscall arguments
	for _, reg := range RegisterSet.Data {
		if reg == rax || reg == rcx || reg == r11 {
			convention.CallerSaved(reg)
		} else {
			convention.CalleeSaved(reg)
		}
	}

	calleeSavedArguments := []*architecture.Register{rdi, rsi, rdx, r10, r8, r9}
	for _, register := range calleeSavedArguments[:numArgs] {
		convention.AddRegisterSource(false, register)
	}

	return convention
}

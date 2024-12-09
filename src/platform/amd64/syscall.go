package amd64

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
		return newLinuxSysCallConstraints(len(call.Args))
	default:
		panic("unsupported os: " + os)
	}
}

func newLinuxSysCallConstraints(
	numArgs int,
) *architecture.InstructionConstraints {
	constraints := architecture.NewInstructionConstraints()

	// Clobbered by syscall
	constraints.SelectFrom(true, rcx)
	constraints.SelectFrom(true, r11)

	// Syscall number and return value
	ret := constraints.SelectFrom(true, rax)
	constraints.SetFuncValue(ret)
	constraints.SetRegisterDestination(ret)

	// Syscall arguments
	calleeSavedArguments := []*architecture.Register{rdi, rsi, rdx, r10, r8, r9}
	for _, register := range calleeSavedArguments[:numArgs] {
		constraints.AddRegisterSource(constraints.SelectFrom(false, register))
	}

	return constraints
}

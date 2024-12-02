package amd64

import (
	"github.com/pattyshack/gt/parseutil"

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

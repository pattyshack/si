package amd64

import (
	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/platform"
)

type Platform struct {
	os          platform.OperatingSystemName
	sysCallSpec platform.SysCallSpec
}

func NewPlatform(os platform.OperatingSystemName) platform.Platform {
	return Platform{
		os:          os,
		sysCallSpec: newSysCallSpec(os),
	}
}

func (Platform) ArchitectureName() platform.ArchitectureName {
	return platform.Amd64
}

func (p Platform) OperatingSystemName() platform.OperatingSystemName {
	return p.os
}

func (Platform) CallTypeSpec(
	convention ast.CallConvention,
) platform.CallTypeSpec {
	return platform.NewCallTypeSpec(convention)
}

func (p Platform) SysCallSpec() platform.SysCallSpec {
	return p.sysCallSpec
}

func (Platform) ArchitectureRegisters() *platform.ArchitectureRegisters {
	return ArchitectureRegisters
}

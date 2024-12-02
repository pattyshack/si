package amd64

import (
	"github.com/pattyshack/chickadee/platform"
)

type Platform struct {
	os              platform.OperatingSystemName
	sysCallTypeSpec platform.SysCallTypeSpec
}

func NewPlatform(os platform.OperatingSystemName) platform.Platform {
	return Platform{
		os:              os,
		sysCallTypeSpec: platform.NewSysCallTypeSpec(os),
	}
}

func (Platform) ArchitectureName() platform.ArchitectureName {
	return platform.Amd64
}

func (p Platform) OperatingSystemName() platform.OperatingSystemName {
	return p.os
}

func (p Platform) SysCallTypeSpec() platform.SysCallTypeSpec {
	return p.sysCallTypeSpec
}

func (Platform) ArchitectureRegisters() *platform.ArchitectureRegisters {
	return ArchitectureRegisters
}

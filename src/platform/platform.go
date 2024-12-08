package platform

import (
	"github.com/pattyshack/chickadee/architecture"
	"github.com/pattyshack/chickadee/ast"
)

type ArchitectureName string
type OperatingSystemName string

const (
	Amd64 = ArchitectureName("amd64")

	Linux = OperatingSystemName("linux")
)

type Platform interface {
	ArchitectureName() ArchitectureName
	OperatingSystemName() OperatingSystemName

	CallSpec(ast.CallConvention) CallSpec

	SysCallSpec() SysCallSpec

	ArchitectureRegisters() *architecture.RegisterSet

	InstructionConstraints(
		ast.Instruction,
	) *architecture.InstructionConstraints
}

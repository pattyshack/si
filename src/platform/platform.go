package platform

import (
	"encoding/binary"

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

	ByteOrder() binary.ByteOrder

	CallSpec(ast.CallConventionName) CallSpec
	CallConvention(*ast.FunctionType) *architecture.CallConvention

	SysCallSpec() SysCallSpec

	ArchitectureRegisters() *architecture.RegisterSet

	InstructionConstraints(
		ast.Instruction,
	) *architecture.InstructionConstraints

	CanEncodeImmediate(ast.Value) bool
}

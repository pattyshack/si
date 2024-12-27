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

	CallSpec(ast.CallConventionName) CallSpec
	CallConvention(*ast.FunctionType) *architecture.CallConvention

	SysCallSpec() SysCallSpec

	ArchitectureRegisters() *architecture.RegisterSet

	InstructionConstraints(
		ast.Instruction,
	) *architecture.InstructionConstraints

	// XXX: maybe this belongs to call/syscall convention?
	//
	// e.g., System V amd64 requires 16 (=2*RegisterByteSize) byte aligned
	// stack frames (%rsp before calling any funcion must have the form
	// 0x???????????????0).
	StackFrameAlignment() int // in byte
}

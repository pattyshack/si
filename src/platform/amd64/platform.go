package amd64

import (
	"github.com/pattyshack/chickadee/architecture"
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

func (Platform) CallSpec(
	convention ast.CallConvention,
) platform.CallSpec {
	return NewCallSpec(convention)
}

func (p Platform) SysCallSpec() platform.SysCallSpec {
	return p.sysCallSpec
}

func (Platform) ArchitectureRegisters() *architecture.RegisterSet {
	return RegisterSet
}

func (p Platform) InstructionConstraints(
	in ast.Instruction,
) *architecture.InstructionConstraints {
	switch inst := in.(type) {
	case *ast.Phi:
		panic("TODO") // should use the same constraints as copy op
	case *ast.CopyOperation:
		panic("TODO")
	case *ast.UnaryOperation:
		if ast.IsFloatSubType(inst.Dest.Type) {
			return floatUnaryOpConstraints
		} else {
			return intUnaryOpConstraints
		}
	case *ast.BinaryOperation:
		if ast.IsFloatSubType(inst.Dest.Type) {
			return floatBinaryOpConstraints
		} else {
			return intBinaryOpConstraints
		}
	case *ast.Jump:
		return jumpConstraints
	case *ast.ConditionalJump:
		if ast.IsFloatSubType(inst.Src1.Type()) {
			return floatConditionalJumpConstraints
		} else {
			return intConditionalJumpConstraints
		}
	case *ast.FuncCall:
		switch inst.Kind {
		case ast.Call:
			switch value := inst.Func.(type) {
			case *ast.VariableReference:
				funcType := value.UseDef.Type.(ast.FunctionType)
				callSpec := p.CallSpec(funcType.CallConvention)
				constraints, _ := callSpec.CallRetConstraints(funcType)
				return constraints
			case *ast.GlobalLabelReference:
				return value.Signature.(*ast.FunctionDefinition).CallRetConstraints
			default: // immediate can't have func type
				panic("This should never happen")
			}
		case ast.SysCall:
			return newSysCallConstraints(p.os, inst)
		default:
			panic("unhandled func call kind: " + inst.Kind)
		}
	case *ast.Terminal:
		switch inst.Kind {
		case ast.Ret:
			return inst.ParentBlock().ParentFuncDef.CallRetConstraints
		case ast.Exit:
			return newSysCallConstraints(p.os, inst.ExitSysCall)
		default:
			panic("unhandled terminal kind: " + inst.Kind)
		}
	}

	panic("should never reach here")
}

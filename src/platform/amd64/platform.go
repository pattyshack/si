package amd64

import (
	"fmt"
	"math"

	"github.com/pattyshack/chickadee/architecture"
	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/platform"
)

type Platform struct {
	os          platform.OperatingSystemName
	sysCallSpec platform.SysCallSpec

	*platform.CallSpecs
}

func NewPlatform(os platform.OperatingSystemName) platform.Platform {
	return Platform{
		os:          os,
		sysCallSpec: newSysCallSpec(os),
		CallSpecs:   newCallSpecs(),
	}
}

func (Platform) ArchitectureName() platform.ArchitectureName {
	return platform.Amd64
}

func (p Platform) OperatingSystemName() platform.OperatingSystemName {
	return p.os
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
		return newCopyOpConstraints(inst.Dest.Type)
	case *ast.CopyOperation:
		return newCopyOpConstraints(inst.Dest.Type)
	case *ast.UnaryOperation:
		if ast.IsFloatSubType(inst.Dest.Type) {
			switch inst.Kind {
			case ast.ToI8, ast.ToI16, ast.ToI32, ast.ToI64,
				ast.ToU8, ast.ToU16, ast.ToU32, ast.ToU64:

				return floatToIntConstraints
			default:
				return floatUnaryOpConstraints
			}
		} else {
			switch inst.Kind {
			case ast.ToF32, ast.ToF64:
				return intToFloatConstraints
			default:
				return intUnaryOpConstraints
			}
		}
	case *ast.BinaryOperation:
		if ast.IsFloatSubType(inst.Dest.Type) {
			return genericFloatBinaryOpConstraints
		} else {
			if inst.Kind == ast.Div {
				return divConstraints
			} else if inst.Kind == ast.Rem {
				return remConstraints
			} else {
				return genericIntBinaryOpConstraints
			}
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
			funcType := inst.Func.Type().(*ast.FunctionType)
			return p.CallConvention(funcType).CallConstraints
		case ast.SysCall:
			return newSysCallConstraints(p.os, inst)
		default:
			panic("unhandled func call kind: " + inst.Kind)
		}
	case *ast.Terminal:
		switch inst.Kind {
		case ast.Ret:
			funcType := inst.ParentBlock().ParentFuncDef.FuncType
			return p.CallConvention(funcType).RetConstraints
		case ast.Exit:
			// exit is replaced by syscall immediately after cfg initialization
			panic("should never happen")
		default:
			panic("unhandled terminal kind: " + inst.Kind)
		}
	}

	panic(fmt.Sprintf("should never reach here: %s", in.Loc()))
}

func (p Platform) CanEncodeImmediate(value ast.Value) bool {
	switch val := value.(type) {
	case *ast.IntImmediate:
		return p.canEncodeIntImmediate(val)
	default:
		// TODO handle label references / float immediate
		return false
	}
}

func (p Platform) canEncodeIntImmediate(imm *ast.IntImmediate) bool {
	// NOTE: x64's immediate support is ad hoc at best, most support imm32, but
	// mov supports imm64, and div/idiv don't support any immediate.
	//
	// TODO figure out all the corner cases ...

	binary, ok := imm.ParentInstruction.(*ast.BinaryOperation)
	if ok && (binary.Kind == ast.Div || binary.Kind == ast.Rem) {
		return false
	}

	if ast.IsSignedIntSubType(imm.Type()) {
		if imm.IsNegative {
			return uint64(-math.MinInt32) >= imm.Value
		} else {
			return math.MaxInt32 >= imm.Value
		}
	} else {
		return math.MaxUint32 >= imm.Value
	}
}

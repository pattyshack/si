package amd64

import (
	"sort"

	"github.com/pattyshack/chickadee/architecture"
	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/platform"
)

const (
	registerSize = 8
)

func NewCallSpec(convention ast.CallConvention) platform.CallSpec {
	switch convention {
	case ast.InternalCallConvention:
		return internalCallSpec{
			NumGeneral:            9,
			NumFloat:              8,
			NumCallerSavedGeneral: 9,
			NumCallerSavedFloat:   8,
		}
	case ast.InternalCalleeSavedCallConvention:
		return internalCallSpec{
			NumGeneral:            14,
			NumFloat:              16,
			NumCallerSavedGeneral: 14,
			NumCallerSavedFloat:   16,
		}
	case ast.InternalCallerSavedCallConvention:
		return internalCallSpec{
			NumGeneral:            14,
			NumFloat:              16,
			NumCallerSavedGeneral: 1,
			NumCallerSavedFloat:   0,
		}
	case ast.SystemVLiteCallConvention:
		return systemVLiteCallSpec{}
	default: // Error emitted by ast syntax validator
		return internalCallSpec{}
	}
}

type internalCallSpec struct {
	platform.InternalCallTypeSpec

	// The number of general registers used for sources and destination, not
	// counting frame pointer. (Must be one or larger)
	//
	// Not counting the frame pointer register, the first general register is
	// always used for function location value, the next (NumGeneral - 1) general
	// registers are usable for int/data arguments.
	//
	// Not counting the frame pointer register, the first NumGeneral general
	// registers are usable for int/data return value.
	NumGeneral int

	// Number of float registers used (can be zero).
	//
	// The first NumFloat float registers are usable for float/data arguments and
	// return value.
	NumFloat int

	// 1 <= NumCallerSavedGeneral <= NumGeneral <= len(RegisterSet.General)-2
	NumCallerSavedGeneral int

	// 0 <= NumCallerSavedFloat <= NumFloat <= len(RegisterSet.Float)
	NumCallerSavedFloat int
}

// The first general register is always used for frame pointer (callee-saved).
//
// The second general register is always used for function location value.
// The next (NumGeneral - 1) general registers are usable for int/data
// arguments.
//
// The first NumFloat float registers are usable for float/data arguments.
//
// The same set of NumGeneral general registers and NumFloat float registers
// are usable for the return value.
func (spec internalCallSpec) CallRetConstraints(
	funcType ast.FunctionType,
) *architecture.CallConvention {
	convention := architecture.NewCallConvention(true, RegisterSet.General[1])
	convention.SetFramePointerRegister(RegisterSet.General[0])

	general := RegisterSet.General[1 : spec.NumGeneral+1]
	convention.CallerSaved(general[:spec.NumCallerSavedGeneral]...)
	convention.CalleeSaved(RegisterSet.General[spec.NumCallerSavedGeneral+1:]...)

	float := RegisterSet.Float[:spec.NumFloat]
	convention.CallerSaved(float[:spec.NumCallerSavedFloat]...)
	convention.CalleeSaved(RegisterSet.Float[spec.NumCallerSavedFloat:]...)

	// Arguments are grouped by number of registers needed, and iterated from
	// smallest group to largest group.  For each argument, the registers are
	// greedily selected from the remaining available registers.
	paramSizeGroups := map[int][]ast.Type{}
	for _, paramType := range funcType.ParameterTypes {
		numNeeded := paramType.RegisterSize()
		paramSizeGroups[numNeeded] = append(paramSizeGroups[numNeeded], paramType)
	}

	sortedNumNeeded := []int{}
	for numNeeded, _ := range paramSizeGroups {
		sortedNumNeeded = append(sortedNumNeeded, numNeeded)
	}
	sort.Ints(sortedNumNeeded)

	argumentRegisterPicker := &internalCallRegisterPicker{
		availableGeneral: general[1:],
		availableFloat:   float,
	}

	paramLocations := map[ast.Type][]*architecture.Register{}
	for _, numNeeded := range sortedNumNeeded {
		for _, paramType := range paramSizeGroups[numNeeded] {
			registers := argumentRegisterPicker.Pick(paramType, numNeeded)

			_, ok := paramLocations[paramType]
			if ok {
				panic("should never happen")
			}
			paramLocations[paramType] = registers
		}
	}

	for _, paramType := range funcType.ParameterTypes {
		registers, ok := paramLocations[paramType]
		if !ok {
			panic("should never happen")
		}

		if registers == nil { // need to be on memory
			convention.AddStackSource(paramType.ByteSize())
		} else {
			clobbered := true
			if len(registers) > 0 {
				clobbered = convention.CallConstraints.RequiredRegisters[registers[0]]
			}
			convention.AddRegisterSource(clobbered, registers...)
		}
	}

	destinationRegisterPicker := &internalCallRegisterPicker{
		availableGeneral: general,
		availableFloat:   float,
	}

	registers := destinationRegisterPicker.Pick(
		funcType.ReturnType,
		funcType.ReturnType.RegisterSize())

	if registers == nil { // need to be on memory
		convention.SetStackDestination(funcType.ReturnType.ByteSize())
	} else {
		convention.SetRegisterDestination(registers...)
	}

	return convention
}

type internalCallRegisterPicker struct {
	availableGeneral []*architecture.Register
	availableFloat   []*architecture.Register
}

// This return nil if the value should be on memory, empty list if the value
// occupies no space, or a non-empty list if value fits in memory.
func (picker *internalCallRegisterPicker) Pick(
	valueType ast.Type,
	numNeeded int,
) []*architecture.Register {
	if len(picker.availableGeneral)+len(picker.availableFloat) < numNeeded {
		return nil
	}

	if numNeeded == 0 {
		return []*architecture.Register{}
	}

	if ast.IsIntSubType(valueType) || ast.IsFunctionType(valueType) {
		if len(picker.availableGeneral) > 0 {
			result := []*architecture.Register{picker.availableGeneral[0]}
			picker.availableGeneral = picker.availableGeneral[1:]
			return result
		}

		result := []*architecture.Register{picker.availableFloat[0]}
		picker.availableFloat = picker.availableFloat[1:]
		return result
	} else if ast.IsFloatSubType(valueType) {
		if len(picker.availableFloat) > 0 {
			result := []*architecture.Register{picker.availableFloat[0]}
			picker.availableFloat = picker.availableFloat[1:]
			return result
		}

		result := []*architecture.Register{picker.availableGeneral[0]}
		picker.availableGeneral = picker.availableGeneral[1:]
		return result
	} else {
		panic("unhandled type")
	}
}

type systemVLiteCallSpec struct {
	platform.SystemVLiteCallTypeSpec
}

func (systemVLiteCallSpec) CallRetConstraints(
	funcType ast.FunctionType,
) *architecture.CallConvention {
	convention := architecture.NewCallConvention(true, r11)
	convention.SetFramePointerRegister(rbp) // callee-saved

	// See Figure 3.4 Register Usage in
	// https://gitlab.com/x86-psABIs/x86-64-ABI/-/jobs/artifacts/master/raw/x86-64-ABI/abi.pdf?job=build

	// General argument registers are caller-saved.
	generalArgs := []*architecture.Register{rdi, rsi, rdx, rcx, r8, r9}
	convention.CallerSaved(generalArgs...)

	// r10 and r11 are caller-saved temporary registers.  r10 is sometimes a
	// hidden argument for passing static chain pointer.
	//
	// Chickadee uses r11 for func location value since it has no hidden meaning.
	convention.CallerSaved(r10, r11)

	// rax is the caller-saved int return value register.  It's also a hidden
	// argument register for vararg.
	generalRet := rax
	convention.CallerSaved(rax)

	convention.CalleeSaved(rbx, r12, r13, r14, r15)

	// All xmm registers are caller-saved.
	// Only xmm0-xmm7 are usable for argument
	// xmm0 is also the float return register
	convention.CallerSaved(RegisterSet.Float...)
	floatArgs := RegisterSet.Float[:8]
	floatRet := floatArgs[0]

	picker := &systemVLiteCallRegisterPicker{
		availableGeneral: generalArgs,
		availableFloat:   floatArgs,
	}

	for _, paramType := range funcType.ParameterTypes {
		register := picker.Pick(paramType)
		if register == nil { // need to be on memory
			convention.AddStackSource(paramType.ByteSize())
		} else {
			convention.AddRegisterSource(true, register)
		}
	}

	if ast.IsFloatSubType(funcType.ReturnType) {
		// xmm0 is also the float return register
		convention.SetRegisterDestination(floatRet)
	} else {
		convention.SetRegisterDestination(generalRet)
	}

	return convention
}

type systemVLiteCallRegisterPicker struct {
	availableGeneral []*architecture.Register
	availableFloat   []*architecture.Register
}

func (picker *systemVLiteCallRegisterPicker) Pick(
	valueType ast.Type,
) *architecture.Register {
	if ast.IsFloatSubType(valueType) {
		if len(picker.availableFloat) == 0 {
			return nil
		}

		result := picker.availableFloat[0]
		picker.availableFloat = picker.availableFloat[1:]
		return result
	} else {
		if len(picker.availableGeneral) == 0 {
			return nil
		}

		result := picker.availableGeneral[0]
		picker.availableGeneral = picker.availableGeneral[1:]
		return result
	}
}

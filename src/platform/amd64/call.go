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
		return internalCallSpec{}
	case ast.SystemVLiteCallConvention:
		return systemVLiteCallSpec{}
	default: // Error emitted by ast syntax validator
		return internalCallSpec{}
	}
}

type internalCallSpec struct {
	platform.InternalCallTypeSpec
}

func (internalCallSpec) CallRetConstraints(
	funcType ast.FunctionType,
) (
	*architecture.InstructionConstraints,
	[]*ast.VariableDefinition,
) {
	constraints := architecture.NewInstructionConstraints()

	// The first general register is always used for function location value,
	// the next 8 general registers are usable for int/data arguments.  The first
	// 8 float registers are usable for float/data arguments.  All these registers
	// clobbered.  XXX: maybe make the number of registers dynamic?
	//
	// The return value will uses the same set of registers using the same
	// assignment algorithm, except the first general register is also usable for
	// the return value.

	general := []*architecture.RegisterCandidate{}
	for _, reg := range RegisterSet.General[:9] {
		general = append(general, constraints.Select(true, reg))
	}

	float := []*architecture.RegisterCandidate{}
	for _, reg := range RegisterSet.Float[:8] {
		float = append(float, constraints.Select(true, reg))
	}

	constraints.SetFuncValue(general[0])

	// Arguments are grouped by number of registers needed, and iterated from
	// smallest group to largest group.  For each argument, the registers are
	// greedily selected from the remaining available registers.
	paramSizeGroups := map[int][]ast.Type{}
	for _, paramType := range funcType.ParameterTypes {
		numNeeded := (paramType.Size() + registerSize - 1) / registerSize
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

	paramLocations := map[ast.Type][]*architecture.RegisterCandidate{}
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
			constraints.AddStackSource(paramType.Size())
		} else {
			constraints.AddRegisterSource(registers...)
		}
	}

	destinationRegisterPicker := &internalCallRegisterPicker{
		availableGeneral: general,
		availableFloat:   float,
	}

	registers := destinationRegisterPicker.Pick(
		funcType.ReturnType,
		(funcType.ReturnType.Size()+registerSize-1)/registerSize)

	if registers == nil { // need to be on memory
		constraints.SetStackDestination(funcType.ReturnType.Size())
	} else {
		constraints.SetRegisterDestination(registers...)
	}

	// Create pseudo parameter entries for callee-saved registers

	calleeSaved := []*ast.VariableDefinition{}

	for _, reg := range RegisterSet.General[9:] {
		constraints.AddRegisterSource(constraints.Select(false, reg))
		calleeSaved = append(
			calleeSaved,
			&ast.VariableDefinition{
				StartEndPos: funcType.StartEnd(),
				Name:        "%" + reg.Name,
				Type:        ast.NewU64(funcType.StartEnd()),
				DefUses:     map[*ast.VariableReference]struct{}{},
			})
	}

	for _, reg := range RegisterSet.Float[8:] {
		constraints.AddRegisterSource(constraints.Select(false, reg))
		calleeSaved = append(
			calleeSaved,
			&ast.VariableDefinition{
				StartEndPos: funcType.StartEnd(),
				Name:        "%" + reg.Name,
				Type:        ast.NewF64(funcType.StartEnd()),
				DefUses:     map[*ast.VariableReference]struct{}{},
			})
	}

	return constraints, calleeSaved
}

type internalCallRegisterPicker struct {
	availableGeneral []*architecture.RegisterCandidate
	availableFloat   []*architecture.RegisterCandidate
}

// This return nil if the value should be on memory, empty list if the value
// occupies no space, or a non-empty list if value fits in memory.
func (picker *internalCallRegisterPicker) Pick(
	valueType ast.Type,
	numNeeded int,
) []*architecture.RegisterCandidate {
	if len(picker.availableGeneral)+len(picker.availableFloat) < numNeeded {
		return nil
	}

	if numNeeded == 0 {
		return []*architecture.RegisterCandidate{}
	}

	if ast.IsIntSubType(valueType) || ast.IsFunctionType(valueType) {
		if len(picker.availableGeneral) > 0 {
			result := []*architecture.RegisterCandidate{picker.availableGeneral[0]}
			picker.availableGeneral = picker.availableGeneral[1:]
			return result
		}

		result := []*architecture.RegisterCandidate{picker.availableFloat[0]}
		picker.availableFloat = picker.availableFloat[1:]
		return result
	} else if ast.IsFloatSubType(valueType) {
		if len(picker.availableFloat) > 0 {
			result := []*architecture.RegisterCandidate{picker.availableFloat[0]}
			picker.availableFloat = picker.availableFloat[1:]
			return result
		}

		result := []*architecture.RegisterCandidate{picker.availableGeneral[0]}
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
) (
	*architecture.InstructionConstraints,
	[]*ast.VariableDefinition,
) {
	constraints := architecture.NewInstructionConstraints()

	// See Figure 3.4 Register Usage in
	// https://gitlab.com/x86-psABIs/x86-64-ABI/-/jobs/artifacts/master/raw/x86-64-ABI/abi.pdf?job=build

	// General argument registers are caller-saved.
	general := []*architecture.RegisterCandidate{}
	for _, reg := range []*architecture.Register{rdi, rsi, rdx, rcx, r8, r9} {
		general = append(general, constraints.Select(true, reg))
	}

	// All xmm registers are caller-saved.
	float := []*architecture.RegisterCandidate{}
	for _, reg := range RegisterSet.Float {
		float = append(float, constraints.Select(true, reg))
	}

	picker := &systemVLiteCallRegisterPicker{
		availableGeneral: general,
		availableFloat:   float[:8], // Only xmm0-xmm7 are usable for argument
	}

	for _, paramType := range funcType.ParameterTypes {
		register := picker.Pick(paramType)
		if register == nil { // need to be on memory
			constraints.AddStackSource(paramType.Size())
		} else {
			constraints.AddRegisterSource(register)
		}
	}

	// r10 and r11 are caller-saved temporary registers.  r10 is sometimes a
	// hidden argument for passing static chain pointer.
	//
	// We'll use r11 for func location value since it has no hidden meaning.
	constraints.Select(true, r10)
	constraints.SetFuncValue(constraints.Select(true, r11))

	// rax is the caller-saved int return value register.  It's also a hidden
	// argument register for vararg.
	generalRet := constraints.Select(true, rax)

	if ast.IsFloatSubType(funcType.ReturnType) {
		// xmm0 is also the float return register
		constraints.SetRegisterDestination(float[0])
	} else {
		constraints.SetRegisterDestination(generalRet)
	}

	// Create pseudo parameter entries for callee-saved registers

	calleeSaved := []*ast.VariableDefinition{}

	for _, reg := range []*architecture.Register{rbx, rbp, r12, r13, r14, r15} {
		constraints.AddRegisterSource(constraints.Select(false, reg))
		calleeSaved = append(
			calleeSaved,
			&ast.VariableDefinition{
				StartEndPos: funcType.StartEnd(),
				Name:        "%" + reg.Name,
				Type:        ast.NewU64(funcType.StartEnd()),
				DefUses:     map[*ast.VariableReference]struct{}{},
			})
	}

	return constraints, calleeSaved
}

type systemVLiteCallRegisterPicker struct {
	availableGeneral []*architecture.RegisterCandidate
	availableFloat   []*architecture.RegisterCandidate
}

func (picker *systemVLiteCallRegisterPicker) Pick(
	valueType ast.Type,
) *architecture.RegisterCandidate {
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

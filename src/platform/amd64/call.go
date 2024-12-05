package amd64

import (
	"sort"

	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/platform"
)

const (
	registerSize = 8
)

func newInternalCallConstraints(
	funcType ast.FunctionType,
) *platform.InstructionConstraints {
	constraints := platform.NewInstructionConstraints()

	// The first general register is always used for function location value,
	// the next 8 general registers are usable for int/data arguments.  The first
	// 8 float registers are usable for float/data arguments.  All these registers
	// clobbered.  XXX: maybe make the number of registers dynamic?
	//
	// The return value will uses the same set of registers using the same
	// assignment algorithm, except the first general register is also usable for
	// the return value.

	general := []*platform.SelectedRegister{}
	for _, reg := range ArchitectureRegisters.General[:9] {
		general = append(general, constraints.Select(true, reg))
	}

	float := []*platform.SelectedRegister{}
	for _, reg := range ArchitectureRegisters.Float[:8] {
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

	argumentRegisterPicker := &registerPicker{
		availableGeneral: general[1:],
		availableFloat:   float,
	}

	paramLocations := map[ast.Type][]*platform.SelectedRegister{}
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
			constraints.AddStackSource(paramType)
		} else {
			constraints.AddRegisterSource(registers...)
		}
	}

	destinationRegisterPicker := &registerPicker{
		availableGeneral: general,
		availableFloat:   float,
	}

	registers := destinationRegisterPicker.Pick(
		funcType.ReturnType,
		(funcType.ReturnType.Size()+registerSize-1)/registerSize)

	if registers == nil { // need to be on memory
		constraints.SetStackDestination(funcType.ReturnType)
	} else {
		constraints.SetRegisterDestination(registers...)
	}

	return constraints
}

type registerPicker struct {
	availableGeneral []*platform.SelectedRegister
	availableFloat   []*platform.SelectedRegister
}

// This return nil if the value should be on memory, empty list if the value
// occupies no space, or a non-empty list if value fits in memory.
func (picker *registerPicker) Pick(
	valueType ast.Type,
	numNeeded int,
) []*platform.SelectedRegister {
	if len(picker.availableGeneral)+len(picker.availableFloat) < numNeeded {
		return nil
	}

	if numNeeded == 0 {
		return []*platform.SelectedRegister{}
	}

	if ast.IsIntSubType(valueType) || ast.IsFunctionType(valueType) {
		if len(picker.availableGeneral) > 0 {
			result := []*platform.SelectedRegister{picker.availableGeneral[0]}
			picker.availableGeneral = picker.availableGeneral[1:]
			return result
		}

		result := []*platform.SelectedRegister{picker.availableFloat[0]}
		picker.availableFloat = picker.availableFloat[1:]
		return result
	} else if ast.IsFloatSubType(valueType) {
		if len(picker.availableFloat) > 0 {
			result := []*platform.SelectedRegister{picker.availableFloat[0]}
			picker.availableFloat = picker.availableFloat[1:]
			return result
		}

		result := []*platform.SelectedRegister{picker.availableGeneral[0]}
		picker.availableGeneral = picker.availableGeneral[1:]
		return result
	} else {
		panic("unhandled type")
	}
}

package allocator

import (
	"github.com/pattyshack/chickadee/architecture"
	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/platform"
)

// Where definition's data is located at block boundaries.
type LocationSet map[*ast.VariableDefinition]*architecture.DataLocation

// A register is free if it is not reserved, and is not used by a variable
// definition.
type RegisterInfo struct {
	// This indicates the register is temporary not selectable/free.
	//
	// This flag is reset after each instruction.
	Reserved bool

	// Which variable definition is currently using this register.
	UsedBy *architecture.DataLocation
}

func (info *RegisterInfo) SetUsedBy(loc *architecture.DataLocation) {
	if info.UsedBy != nil {
		panic("should never happen")
	}
	info.UsedBy = loc
}

// Where values are located at a particular point in execution within a block.
// Note that copies of a value may temporarily reside in multiple locations.
type ValueLocations struct {
	*architecture.StackFrame

	// NOTE: TempStack is ordered from top to bottom
	TempStack []*architecture.DataLocation

	Registers map[*architecture.Register]*RegisterInfo

	Values     map[*ast.VariableDefinition]map[*architecture.DataLocation]struct{}
	allocated  map[*architecture.DataLocation]*ast.VariableDefinition
	valueNames map[string]*ast.VariableDefinition
}

func NewValueLocations(
	targetPlatform platform.Platform,
	frame *architecture.StackFrame,
	locationIn LocationSet,
) *ValueLocations {
	locations := &ValueLocations{
		StackFrame: frame,
		Registers:  map[*architecture.Register]*RegisterInfo{},
		Values:     map[*ast.VariableDefinition]map[*architecture.DataLocation]struct{}{},
		allocated:  map[*architecture.DataLocation]*ast.VariableDefinition{},
		valueNames: map[string]*ast.VariableDefinition{},
	}

	for _, reg := range targetPlatform.ArchitectureRegisters().Data {
		locations.Registers[reg] = &RegisterInfo{}
	}

	for def, loc := range locationIn {
		if loc.OnFixedStack {
			locations.AssignLocationToDefinition(
				locations.AllocateFixedStackLocation(def),
				def)
		} else if loc.OnTempStack {
			panic("should never happen")
		} else {
			registers := make([]*architecture.Register, 0, len(loc.Registers))
			registers = append(registers, loc.Registers...)

			locations.AssignLocationToDefinition(
				locations.AllocateRegistersLocation(def.Name, def.Type, registers),
				def)
		}
	}

	return locations
}

func (locations *ValueLocations) getRegInfo(
	reg *architecture.Register,
) *RegisterInfo {
	info, ok := locations.Registers[reg]
	if !ok {
		panic("invalid register: " + reg.Name)
	}
	return info
}

func (locations *ValueLocations) getLocations(
	def *ast.VariableDefinition,
) map[*architecture.DataLocation]struct{} {
	set, ok := locations.Values[def]
	if !ok {
		panic("should never happen")
	}
	return set
}

func (locations *ValueLocations) AllocateRegistersLocation(
	name string, // use "" for immediate / global label value
	valueType ast.Type,
	registers []*architecture.Register,
) *architecture.DataLocation {
	dest := architecture.NewRegistersDataLocation(
		name,
		valueType,
		registers)
	locations.allocateRegistersLocation(dest)
	return dest
}

func (locations *ValueLocations) allocateRegistersLocation(
	loc *architecture.DataLocation,
) {
	for _, reg := range loc.Registers {
		locations.getRegInfo(reg).SetUsedBy(loc)
	}
	locations.allocated[loc] = nil
}

func (locations *ValueLocations) AllocateFixedStackLocation(
	def *ast.VariableDefinition,
) *architecture.DataLocation {
	dest := locations.StackFrame.MaybeAddLocalVariable(def.Name, def.Type)
	locations.allocated[dest] = nil
	return dest
}

func (locations *ValueLocations) AllocateTempStackLocations(
	targetPlatform platform.Platform,
	stackArgumentTypes []ast.Type, // top to bottom order
	stackReturnType ast.Type, // always at the bottom, could be nil
) (
	[]*architecture.DataLocation, // locations, same order as argument types
	*architecture.DataLocation, // return value location
) {
	if locations.TempStack != nil {
		// TODO: ensure TempStack is nil at end of block
		for _, loc := range locations.TempStack {
			_, ok := locations.allocated[loc]
			if ok {
				panic("should never happen") // forgot to free temp stack location
			}
		}
		locations.TempStack = nil
	}

	callTempSize := 0
	argLocs := []*architecture.DataLocation{}
	tempStack := []*architecture.DataLocation{}
	for _, argType := range stackArgumentTypes {
		loc := architecture.NewTempStackDataLocation(argType)
		loc.Offset = callTempSize

		locations.allocated[loc] = nil

		callTempSize += loc.AlignedSize
		argLocs = append(argLocs, loc)
		tempStack = append(tempStack, loc)
	}

	var returnLoc *architecture.DataLocation
	if stackReturnType != nil {
		returnLoc = architecture.NewTempStackDataLocation(stackReturnType)
		returnLoc.Offset = callTempSize

		locations.allocated[returnLoc] = nil

		callTempSize += returnLoc.AlignedSize
		tempStack = append(tempStack, returnLoc)
	}

	locations.StackFrame.UpdateMaxTempSize(callTempSize)
	locations.TempStack = tempStack
	return argLocs, returnLoc
}

// Note: freed registers' Reserved states are reset to false.
func (locations *ValueLocations) FreeLocation(
	toFree *architecture.DataLocation,
) {
	for _, reg := range toFree.Registers {
		info := locations.getRegInfo(reg)
		if info.UsedBy != toFree {
			panic("should never happen")
		}
		info.Reserved = false
		info.UsedBy = nil
	}

	def, ok := locations.allocated[toFree]
	if !ok {
		panic("should never happen")
	}

	delete(locations.allocated, toFree)
	if def == nil {
		return
	}

	set := locations.getLocations(def)
	_, ok = set[toFree]
	if !ok {
		panic("should never happen")
	}

	if len(set) > 1 {
		delete(set, toFree)
	} else {
		delete(locations.Values, def)
		delete(locations.valueNames, def.Name)
	}
}

// Note: srcRegister's Reserved state is not modified.  destRegister must be
// unoccupied.
func (locations *ValueLocations) MoveRegister(
	srcRegister *architecture.Register,
	destRegister *architecture.Register,
) *architecture.DataLocation {
	srcInfo := locations.getRegInfo(srcRegister)
	if srcInfo.UsedBy == nil {
		panic("should never happen")
	}

	oldLoc := srcInfo.UsedBy
	def := locations.allocated[oldLoc]
	locations.FreeLocation(oldLoc)

	modified := false
	newLoc := oldLoc.Copy()
	for idx, reg := range newLoc.Registers {
		if reg == srcRegister {
			newLoc.Registers[idx] = destRegister
			modified = true
			break
		}
	}

	if !modified {
		panic("should never happen")
	}

	locations.allocateRegistersLocation(newLoc)
	if def != nil {
		locations.AssignLocationToDefinition(newLoc, def)
	}

	return newLoc
}

// Note: this assumes that the location already hold the correct data.
func (locations *ValueLocations) AssignLocationToDefinition(
	loc *architecture.DataLocation,
	def *ast.VariableDefinition,
) {
	// Ensure definition name is unique.
	foundDef, ok := locations.valueNames[def.Name]
	if !ok {
		locations.valueNames[def.Name] = def
	} else if foundDef != def {
		panic("should never happen")
	}

	// Ensure the location was correctly allocated.
	foundDef, ok = locations.allocated[loc]
	if !ok {
		panic("should never happen")
	} else if foundDef != nil {
		panic("should never happen")
	}

	set, ok := locations.Values[def]
	if !ok {
		set = map[*architecture.DataLocation]struct{}{}
		locations.Values[def] = set
	}

	set[loc] = struct{}{}
	locations.allocated[loc] = def
}

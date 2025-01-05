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

	// NOTE: Global reference, immediate, and temp stack argument/result values
	// are tracked via pseudo variable definitions with internally generated
	// names.
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
			locations.AllocateFixedStackLocation(def)
		} else if loc.OnTempStack {
			panic("should never happen")
		} else {
			locations.AllocateRegistersLocation(def, loc.Registers...)
		}
	}

	return locations
}

func (locations *ValueLocations) AssertNotAllocated(
	loc *architecture.DataLocation,
) {
	_, ok := locations.allocated[loc]
	if ok {
		panic("should never happen")
	}
}

func (locations *ValueLocations) AssertAllocated(
	loc *architecture.DataLocation,
) {
	_, ok := locations.allocated[loc]
	if !ok {
		panic("should never happen")
	}
}

func (locations *ValueLocations) AssertNotFree(
	reg *architecture.Register,
) {
	if locations.getRegInfo(reg).UsedBy == nil {
		panic("should never happen")
	}
}

func (locations *ValueLocations) AssertFree(
	reg *architecture.Register,
) {
	if locations.getRegInfo(reg).UsedBy != nil {
		panic("should never happen")
	}
}

func (locations *ValueLocations) AssertNoTempLocations() {
	for _, loc := range locations.TempStack {
		_, ok := locations.allocated[loc]
		if ok {
			panic("should never happen")
		}
	}

	locations.TempStack = nil
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

func (locations *ValueLocations) allocate(
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

	_, ok = locations.allocated[loc]
	if ok {
		panic("should never happen")
	}
	locations.allocated[loc] = def

	set, ok := locations.Values[def]
	if !ok {
		set = map[*architecture.DataLocation]struct{}{}
		locations.Values[def] = set
	}
	set[loc] = struct{}{}
}

func (locations *ValueLocations) allocateRegisters(
	loc *architecture.DataLocation,
	def *ast.VariableDefinition,
) {
	for _, reg := range loc.Registers {
		locations.getRegInfo(reg).SetUsedBy(loc)
	}
	locations.allocate(loc, def)
}

func (locations *ValueLocations) AllocateRegistersLocation(
	def *ast.VariableDefinition,
	registers ...*architecture.Register,
) *architecture.DataLocation {
	dest := architecture.NewRegistersDataLocation(
		def.Name,
		def.Type,
		registers)
	locations.allocateRegisters(dest, def)
	return dest
}

func (locations *ValueLocations) AllocateFixedStackLocation(
	def *ast.VariableDefinition,
) *architecture.DataLocation {
	dest := locations.StackFrame.MaybeAddLocalVariable(def.Name, def.Type)
	locations.allocate(dest, def)
	return dest
}

func (locations *ValueLocations) AllocateTempStackLocations(
	argDefs []*ast.VariableDefinition, // top to bottom order
	returnDef *ast.VariableDefinition, // always at the bottom, could be nil
) (
	[]*architecture.DataLocation, // locations, same order as argument types
	*architecture.DataLocation, // return value location
) {
	locations.AssertNoTempLocations()

	callTempSize := 0
	argLocs := []*architecture.DataLocation{}
	tempStack := []*architecture.DataLocation{}
	for _, argDef := range argDefs {
		loc := architecture.NewTempStackDataLocation(argDef.Type)
		loc.Offset = callTempSize

		locations.allocated[loc] = nil

		callTempSize += loc.AlignedSize
		argLocs = append(argLocs, loc)
		tempStack = append(tempStack, loc)
	}

	var returnLoc *architecture.DataLocation
	if returnDef != nil {
		returnLoc = architecture.NewTempStackDataLocation(returnDef.Type)
		returnLoc.Offset = callTempSize

		locations.allocated[returnLoc] = nil

		callTempSize += returnLoc.AlignedSize
		tempStack = append(tempStack, returnLoc)
	}

	locations.StackFrame.UpdateMaxTempSize(callTempSize)
	locations.TempStack = tempStack
	return argLocs, returnLoc
}

func (locations *ValueLocations) FreeLocation(
	toFree *architecture.DataLocation,
) {
	for _, reg := range toFree.Registers {
		info := locations.getRegInfo(reg)
		if info.UsedBy != toFree {
			panic("should never happen")
		}
		info.UsedBy = nil
	}

	def, ok := locations.allocated[toFree]
	if !ok {
		panic("should never happen")
	}
	delete(locations.allocated, toFree)

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

func (locations *ValueLocations) MoveRegister(
	srcRegister *architecture.Register,
	destRegister *architecture.Register,
) *architecture.DataLocation {
	srcInfo := locations.getRegInfo(srcRegister)
	if srcInfo.UsedBy == nil {
		panic("should never happen")
	}

	destInfo := locations.getRegInfo(destRegister)
	if destInfo.UsedBy != nil {
		panic("should never happen")
	}

	oldLoc := srcInfo.UsedBy
	def := locations.allocated[oldLoc]
	locations.FreeLocation(oldLoc)

	modifiedCount := 0
	newLoc := oldLoc.Copy()
	for idx, reg := range newLoc.Registers {
		if reg == srcRegister {
			newLoc.Registers[idx] = destRegister
			modifiedCount++
		}
	}

	if modifiedCount != 1 {
		panic("should never happen")
	}

	locations.allocateRegisters(newLoc, def)
	return newLoc
}

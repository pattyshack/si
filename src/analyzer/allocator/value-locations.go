package allocator

import (
	"sort"
	"strings"

	"github.com/pattyshack/chickadee/architecture"
	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/platform"
)

// Where definition's data is located at block boundaries.
type LocationSet map[*ast.VariableDefinition]*architecture.DataLocation

// Stack frame layout from top to bottom:
//
// |              | (low address)
// |...           |
// |              |  / start of current stack frame's temp portion
// |--------------| <
// |local var m   |  \ end of current stack frame's fixed portion
// |--------------|
// |...           |
// |--------------|
// |local var 2   |
// |--------------|
// |local var 1   | non-argument local variables that are spills to stack
// |--------------|
// |prev frame ptr| Optional, depending on call convention
// |--------------| <- start of current stack frame's fixed portion
// |ret address   |
// |--------------| <- this is stack frame size aligned (see padding below)
// |argument 1    | first argument that goes on the stack
// |--------------|
// |...           |
// |--------------|
// |argument n-1  |
// |--------------|
// |argument n    |
// |--------------|
// |destination   |
// |--------------|
// |padding       | extra space allocated that is not used by this call
// |--------------| <- start of previous stack frame's temp portion
// |              |  \ end of previous stack frame's fixed portion
// |...           |
// |              | (high address)
//
// For allocation purpose, StackFrame keeps track of the fixed portion of
// the stack frame, starting from the caller's allocated destination/arguments.
//
// For simplicity, each unique variable name (could map to multiple definitions)
// that gets spill onto stack will occupy a unique/predetermined location in
// the fixed portion of the stack frame.  Calls may increase the temp portion
// of the stack frame, and thus the total stack frame size.  We'll simply
// preallocate the maximum amount of space needed for any call.  The fixed
// portion is aligned to the bottom of the stack frame and the temp portion
// is aligned to the top of the stack frame.
//
// For now, we assume all stack arguments pass to the function are
// caller-saved to a different location.  The function will reuse the caller
// allocated space for re-splilling those variables.  The stack return value
// should be copied out of the temp portion of the stack asap.
//
// The exact layout of the stack frame is finalized at the end of the
// allocation process. The layout from bottom to top is:
// - previous frame pointer
// - callee-saved sources / pseudo sources (sorted by name)
// - local variables (sorted by name)
type StackFrame struct {
	// All variable name -> location
	Locations map[string]*architecture.DataLocation

	Destination *architecture.DataLocation

	// In natural order (the layout will be in reverse order)
	Parameters []*architecture.DataLocation

	ReturnAddress *architecture.DataLocation

	// Local variables includes all non-stack-passed parameters.  i.e.,
	// register passed parameters, callee-saved registers including previous
	// frame pointer, and locally defined variable)
	LocalVariables map[string]*architecture.DataLocation

	// Note: Total frame size = max temp frame size + fixed frame size
	MaxTempSize int // This respects register alignment (but not frame alignment)

	// Computed by FinalizeFrame()
	TotalFrameSize int                          // This respects stack frame alignment
	Layout         []*architecture.DataLocation // from bottom to top
}

func NewStackFrame() *StackFrame {
	return &StackFrame{
		Locations:      map[string]*architecture.DataLocation{},
		LocalVariables: map[string]*architecture.DataLocation{},
	}
}

func (frame *StackFrame) UpdateMaxTempSize(size int) {
	if size > frame.MaxTempSize {
		frame.MaxTempSize = size
	}
}

func (frame *StackFrame) add(name string, valueType ast.Type) *architecture.DataLocation {
	_, ok := frame.Locations[name]
	if ok {
		panic("duplicate data location: " + name)
	}

	// TODO: for now, we'll allow return address to have unspecified type.  Deal
	// with variable pointer types.
	byteSize := architecture.AddressByteSize
	if name != architecture.ReturnAddress {
		byteSize = valueType.ByteSize()
	}

	loc := architecture.NewFixedStackDataLocation(name, byteSize)
	frame.Locations[name] = loc
	return loc
}

// Must be call before StartCurrentFrame()
func (frame *StackFrame) SetDestination(valueType ast.Type) *architecture.DataLocation {
	if frame.ReturnAddress != nil {
		panic("cannot set destination after starting current frame")
	}
	frame.Destination = frame.add(architecture.StackDestination, valueType)
	return frame.Destination
}

// Must be call before StartCurrentFrame().  Parameters must be added in
// natural (top to bottom) order.
func (frame *StackFrame) AddParameter(
	name string,
	valueType ast.Type,
) *architecture.DataLocation {
	if frame.ReturnAddress != nil {
		panic("cannot add parameters after starting current frame")
	}
	loc := frame.add(name, valueType)
	frame.Parameters = append(frame.Parameters, loc)
	return loc
}

func (frame *StackFrame) StartCurrentFrame() {
	frame.ReturnAddress = frame.add(
		architecture.ReturnAddress,
		nil)
}

func (frame *StackFrame) MaybeAddLocalVariable(
	name string,
	valueType ast.Type,
) *architecture.DataLocation {
	if frame.ReturnAddress == nil {
		panic("StartCurrentFrame not called")
	}
	if frame.Layout != nil {
		panic("cannot add local variable after finalize")
	}
	loc, ok := frame.Locations[name]
	if ok {
		return loc
	}
	loc = frame.add(name, valueType)
	frame.LocalVariables[name] = loc
	return loc
}

func (frame *StackFrame) FinalizeFrame(
	targetPlatform platform.Platform,
) {
	fixedSize := 0
	frameEntries := make([]*architecture.DataLocation, 0, len(frame.LocalVariables))
	for _, loc := range frame.LocalVariables {
		fixedSize += loc.AlignedSize
		frameEntries = append(frameEntries, loc)
	}

	sort.Slice(
		frameEntries,
		func(i int, j int) bool {
			first := frameEntries[i].Name
			second := frameEntries[j].Name

			if first == second {
				panic("should never happen")
			}

			// Frame pointer is always at the bottom of the frame
			if first == architecture.PreviousFramePointer {
				return true
			} else if second == architecture.PreviousFramePointer {
				return false
			}

			firstIsCalleeSaved := strings.HasPrefix(first, "%")
			secondIsCalleeSaved := strings.HasPrefix(second, "%")

			// Callee saved are below real variables
			if firstIsCalleeSaved {
				if !secondIsCalleeSaved {
					return true
				}
				return first < second
			} else if secondIsCalleeSaved {
				return false
			}

			return first < second
		})

	totalFrameSize := fixedSize + frame.MaxTempSize
	frameAlignment := targetPlatform.StackFrameAlignment()
	roundUp := (totalFrameSize + frameAlignment - 1) / frameAlignment
	frame.TotalFrameSize = roundUp * frameAlignment

	layout := make(
		[]*architecture.DataLocation,
		0,
		len(frame.Parameters)+len(frame.LocalVariables)+2)

	if frame.Destination != nil {
		layout = append(layout, frame.Destination)
	}
	// stack arguments are push in reverse order
	for idx := len(frame.Parameters) - 1; idx >= 0; idx-- {
		layout = append(layout, frame.Parameters[idx])
	}
	layout = append(layout, frame.ReturnAddress)
	layout = append(layout, frameEntries...)

	frame.Layout = layout
	currentOffset := frame.TotalFrameSize - fixedSize
	for idx := len(layout) - 1; idx >= 0; idx-- {
		entry := layout[idx]
		entry.Offset = currentOffset
		currentOffset += entry.AlignedSize
	}
}

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
	*StackFrame

	// NOTE: TempStack is ordered from top to bottom
	TempStack []*architecture.DataLocation

	Registers map[*architecture.Register]*RegisterInfo

	Values     map[*ast.VariableDefinition]map[*architecture.DataLocation]struct{}
	allocated  map[*architecture.DataLocation]*ast.VariableDefinition
	valueNames map[string]*ast.VariableDefinition
}

func NewValueLocations(
	targetPlatform platform.Platform,
	frame *StackFrame,
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
		valueType.ByteSize(),
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
		loc := architecture.NewTempStackDataLocation(argType.ByteSize())
		loc.Offset = callTempSize

		locations.allocated[loc] = nil

		callTempSize += loc.AlignedSize
		argLocs = append(argLocs, loc)
		tempStack = append(tempStack, loc)
	}

	var returnLoc *architecture.DataLocation
	if stackReturnType != nil {
		returnLoc = architecture.NewTempStackDataLocation(
			stackReturnType.ByteSize())
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

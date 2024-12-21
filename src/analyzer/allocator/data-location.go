package allocator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pattyshack/chickadee/architecture"
	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/platform"
)

type DataLocation struct {
	Name string

	// TODO: for now, we'll use nil for return address.  Deal with variable sized
	// pointer types
	ast.Type

	// XXX: Support register / stack overlay?
	//
	// For now, data location is either completely on stack or completely in
	// registers.
	Registers    []*architecture.Register
	OnFixedStack bool // available throughout the function's lifetime
	OnTempStack  bool // temporarily allocated for a call instruction

	AlignedSize int // register aligned size

	// For fixed stack locations, the offset is relative to the end of the fixed
	// portion of the stack frame.
	//
	// For temp stack locations, the offset is relative to the top of the stack.
	//
	// NOTE: We'll determine the stack entry address based on stack pointer rather
	// than base pointer:
	//
	// fixed entry address = stack pointer address + temp stack size + offset
	// temp entry address = stack pointer address + offset
	Offset int
}

func NewRegistersDataLocation(
	name string,
	valueType ast.Type,
	registers []*architecture.Register,
) *DataLocation {
	if len(registers) != valueType.RegisterSize() {
		panic("should never happen")
	}

	return &DataLocation{
		Name:      name,
		Type:      valueType,
		Registers: registers,
	}
}

func (loc *DataLocation) Copy() *DataLocation {
	var registers []*architecture.Register
	if loc.Registers != nil {
		registers = make([]*architecture.Register, 0, len(loc.Registers))
		registers = append(registers, loc.Registers...)
	}

	return &DataLocation{
		Name:         loc.Name,
		Type:         loc.Type,
		Registers:    registers,
		OnFixedStack: loc.OnFixedStack,
		OnTempStack:  loc.OnTempStack,
		AlignedSize:  loc.AlignedSize,
		Offset:       loc.Offset,
	}
}

func (loc *DataLocation) String() string {
	registers := []string{}
	for _, reg := range loc.Registers {
		registers = append(registers, reg.Name)
	}
	return fmt.Sprintf(
		"Name: %s Registers: %v OnFixedStack: %v AlignedSize: %d Offset: %d Type: %s",
		loc.Name,
		registers,
		loc.OnFixedStack,
		loc.AlignedSize,
		loc.Offset,
		loc.Type)
}

// Where definition's data is located at block boundaries.
type LocationSet map[*ast.VariableDefinition]*DataLocation

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
// |temp padding  | padding to ensure argument 1 is stack frame size aligned
// |--------------| <- start of previous stack frame's temp portion
// |fixed padding |  \ end of previous stack frame's fixed portion
// |--------------|
// |              |
// |...           |
// |              | (high address)
//
// For allocation purpose, StackFrame keeps track of the fixed portion of
// the stack frame, starting from the caller's allocated destination/arguments.
//
// For simplicity, each unique variable name (could map to multiple definitions)
// that gets spill onto stack will occupy a unique/predetermined location in
// the fixed portion of the stack frame.  Calls may temporarily increase the
// stack frame's size, but the stack frame will immediately shrink back to the
// original size after the call (the call's stack arguments are discarded, and
// its stack destination value is copied).
//
// For now, we assume all stack arguments pass to the function are
// caller-saved to a different location.  The function definition won't
// allocate additional memory for spilling those variable.
//
// The exact layout of the stack frame is finalized at the end of the
// allocation process. The layout from bottom to top is:
// - previous frame pointer
// - callee-saved sources / pseudo sources (sorted by name)
// - local variables (sorted by name)
type StackFrame struct {
	// All variable name -> location
	Locations map[string]*DataLocation

	Destination *DataLocation

	// In natural order (the layout will be in reverse order)
	Parameters []*DataLocation

	ReturnAddress *DataLocation

	// Local variables includes all non-stack-passed parameters.  i.e.,
	// register passed parameters, callee-saved registers including previous
	// frame pointer, and locally defined variable)
	LocalVariables map[string]*DataLocation

	// Note: Total frame size = temp frame size + fixed frame size
	MaxTempSize int

	// Computed by FinalizeFrame()
	FixedSize int             // This respects stack frame alignment
	Layout    []*DataLocation // from bottom to top
}

func NewStackFrame() *StackFrame {
	return &StackFrame{
		Locations:      map[string]*DataLocation{},
		LocalVariables: map[string]*DataLocation{},
	}
}

func (frame *StackFrame) UpdateMaxTempFrameSize(size int) {
	if size > frame.MaxTempSize {
		frame.MaxTempSize = size
	}
}

func (frame *StackFrame) add(name string, valueType ast.Type) *DataLocation {
	_, ok := frame.Locations[name]
	if ok {
		panic("duplicate data location: " + name)
	}
	alignedSize := architecture.AddressByteSize
	if valueType != nil {
		alignedSize = architecture.AlignedSize(valueType.ByteSize())
	}
	loc := &DataLocation{
		Name:         name,
		Type:         valueType,
		OnFixedStack: true,
		AlignedSize:  alignedSize,
	}
	frame.Locations[name] = loc
	return loc
}

// Must be call before StartCurrentFrame()
func (frame *StackFrame) SetDestination(valueType ast.Type) *DataLocation {
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
) *DataLocation {
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
) *DataLocation {
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
	unalignedFrameSize := 0
	frameEntries := make([]*DataLocation, 0, len(frame.LocalVariables))
	for _, loc := range frame.LocalVariables {
		unalignedFrameSize += loc.AlignedSize
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

	frameAlignment := targetPlatform.StackFrameAlignment()
	roundUp := (unalignedFrameSize + frameAlignment - 1) / frameAlignment
	frame.FixedSize = roundUp * frameAlignment

	layout := make(
		[]*DataLocation,
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
	currentOffset := frame.FixedSize - unalignedFrameSize
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
	UsedBy *DataLocation
}

func (info *RegisterInfo) SetUsedBy(loc *DataLocation) {
	if info.UsedBy != nil {
		panic("should never happen")
	}
	info.UsedBy = loc
}

// Where values are located at a particular point in execution within a block.
// Note that copies of a value may temporarily reside in multiple locations.
type ValueLocations struct {
	FixedStack *StackFrame

	// TODO temporary portion of the stack frame (call's arguments and
	// return value)

	Registers map[*architecture.Register]*RegisterInfo

	Values     map[*ast.VariableDefinition]map[*DataLocation]struct{}
	valueNames map[string]*ast.VariableDefinition
}

func NewValueLocations(
	targetPlatform platform.Platform,
	frame *StackFrame,
	locationIn LocationSet,
) *ValueLocations {
	locations := &ValueLocations{
		FixedStack: frame,
		Registers:  map[*architecture.Register]*RegisterInfo{},
		Values:     map[*ast.VariableDefinition]map[*DataLocation]struct{}{},
		valueNames: map[string]*ast.VariableDefinition{},
	}

	for _, reg := range targetPlatform.ArchitectureRegisters().Data {
		locations.Registers[reg] = &RegisterInfo{}
	}

	for def, loc := range locationIn {
		if loc.OnFixedStack {
			locations.NewDefinition(def, locations.FixedStackLocation(def))
		} else if loc.OnTempStack {
			panic("should never happen")
		} else {
			registers := make([]*architecture.Register, 0, len(loc.Registers))
			registers = append(registers, loc.Registers...)

			locations.NewDefinition(
				def,
				locations.RegistersLocation(def.Name, def.Type, registers))
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
) map[*DataLocation]struct{} {
	set, ok := locations.Values[def]
	if !ok {
		panic("should never happen")
	}
	return set
}

func (locations *ValueLocations) ResetForNextInstruction() {
	for _, info := range locations.Registers {
		info.Reserved = false
	}

	// TODO should ensure temp stack is copied/freed/popped
}

// Note: srcRegister's Reserved state is not modified.  destRegister must be
// unoccupied.
func (locations *ValueLocations) MoveData(
	srcRegister *architecture.Register,
	destRegister *architecture.Register,
) {
	srcInfo := locations.getRegInfo(srcRegister)
	if srcInfo.UsedBy == nil {
		panic("should never happen")
	}

	modified := false
	loc := srcInfo.UsedBy
	for idx, reg := range loc.Registers {
		if reg == srcRegister {
			loc.Registers[idx] = destRegister
			modified = true
			break
		}
	}

	if !modified {
		panic("should never happen")
	}

	// TODO record move operation from srcRegister to destRegister

	srcInfo.UsedBy = nil
	locations.getRegInfo(destRegister).SetUsedBy(loc)
}

func (locations *ValueLocations) RegistersLocation(
	name string, // use "" for immediate / global label value
	valueType ast.Type,
	registers []*architecture.Register,
) *DataLocation {
	dest := NewRegistersDataLocation(name, valueType, registers)
	for _, reg := range registers {
		locations.getRegInfo(reg).SetUsedBy(dest)
	}
	return dest
}

func (locations *ValueLocations) FixedStackLocation(
	def *ast.VariableDefinition,
) *DataLocation {
	return locations.FixedStack.MaybeAddLocalVariable(def.Name, def.Type)
}

func (locations *ValueLocations) AssignConstantTo(
	constant ast.Value, // immediate or global label
	dest *DataLocation,
) {
	// TODO record assign constant operation
}

// Note: this assumes that the location already hold the correct data.
func (locations *ValueLocations) NewDefinition(
	def *ast.VariableDefinition,
	loc *DataLocation,
) {
	_, ok := locations.valueNames[def.Name]
	if ok {
		panic("should never happen")
	}
	locations.valueNames[def.Name] = def

	locations.Values[def] = map[*DataLocation]struct{}{
		loc: struct{}{},
	}
}

// Note: registers' Reserved states are not modified.
func (locations *ValueLocations) CopyDefinition(
	def *ast.VariableDefinition,
	dest *DataLocation,
) {
	if !dest.Type.Equals(def.Type) {
		panic("should never happen")
	}

	set := locations.getLocations(def)
	_, ok := set[dest]
	if ok {
		return // A copy of the value is already in dest.  Do nothing
	}

	var src *DataLocation
	for loc, _ := range set {
		src = loc
		if len(src.Registers) > 0 {
			break // prefer copying from registers source
		}
	}

	// TODO record operation copy from src to dest

	set[dest] = struct{}{}
}

// Note: freed registers' Reserved states are reset to false.
func (locations *ValueLocations) FreeLocation(
	toFree *DataLocation,
) {
	for _, reg := range toFree.Registers {
		info := locations.getRegInfo(reg)
		if info.UsedBy != toFree {
			panic("should never happen")
		}
		info.Reserved = false
		info.UsedBy = nil
	}

	def, ok := locations.valueNames[toFree.Name]
	if !ok {
		return // location held immediate or global label value
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

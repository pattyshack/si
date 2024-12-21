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
	Registers []*architecture.Register
	OnStack   bool

	AlignedSize int // register aligned size

	// The offset is relative to the end of the fixed portion of the stack frame.
	//
	// NOTE: We'll determine the stack entry address based on stack pointer rather
	// than base pointer.
	//
	// entry address = stack pointer address + variabled portion size + offset
	Offset int
}

func NewRegistersDataLocation(
	name string,
	valueType ast.Type,
	registers []*architecture.Register,
) *DataLocation {
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
		Name:        loc.Name,
		Type:        loc.Type,
		Registers:   registers,
		OnStack:     loc.OnStack,
		AlignedSize: loc.AlignedSize,
		Offset:      loc.Offset,
	}
}

func (loc *DataLocation) String() string {
	registers := []string{}
	for _, reg := range loc.Registers {
		registers = append(registers, reg.Name)
	}
	return fmt.Sprintf(
		"Name: %s Registers: %v OnStack: %v AlignedSize: %d Offset: %d Type: %s",
		loc.Name,
		registers,
		loc.OnStack,
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
// |              |  / start of current stack frame's variable sized portion
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
// |--------------| <- start of current stack frame's fixed size portion
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
// |padding       | padding to ensure argument 1 is stack frame size aligned
// |--------------| <- start of previous stack frame's variable sized portion
// |              |  \ end of previous stack frame's fixed portion
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

	// Computed by FinalizeFrame()
	FrameSize int             // This respects stack frame alignment
	Layout    []*DataLocation // from bottom to top
}

func NewStackFrame() *StackFrame {
	return &StackFrame{
		Locations:      map[string]*DataLocation{},
		LocalVariables: map[string]*DataLocation{},
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
		Name:        name,
		Type:        valueType,
		OnStack:     true,
		AlignedSize: alignedSize,
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
	frame.FrameSize = roundUp * frameAlignment

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
	currentOffset := frame.FrameSize - unalignedFrameSize
	for idx := len(layout) - 1; idx >= 0; idx-- {
		entry := layout[idx]
		entry.Offset = currentOffset
		currentOffset += entry.AlignedSize
	}
}

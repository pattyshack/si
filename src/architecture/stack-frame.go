package architecture

import (
	"sort"
	"strings"

	"github.com/pattyshack/chickadee/ast"
)

func CompareDefinitionNames(
	first string,
	second string,
) int {
	if first == second {
		return 0
	}

	// Frame pointer is always before other definitions
	if first == PreviousFramePointer {
		return -1
	} else if second == PreviousFramePointer {
		return 1
	}

	// Callee saved defintions are before real definitions
	firstIsCalleeSaved := strings.HasPrefix(first, "%")
	secondIsCalleeSaved := strings.HasPrefix(second, "%")
	if firstIsCalleeSaved {
		if !secondIsCalleeSaved {
			return -1
		}
	} else if secondIsCalleeSaved {
		return 1
	}

	if first < second {
		return -1
	}
	return 1
}

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
	Locations map[string]*DataLocation

	Destination *DataLocation

	// In natural order (the layout will be in reverse order)
	Parameters []*DataLocation

	ReturnAddress *DataLocation

	// Local variables includes all non-stack-passed parameters.  i.e.,
	// register passed parameters, callee-saved registers including previous
	// frame pointer, and locally defined variable)
	LocalVariables map[string]*DataLocation

	// Note: Total frame size = max temp frame size + fixed frame size
	MaxTempSize int // This respects register alignment (but not frame alignment)

	// Computed by FinalizeFrame()
	TotalFrameSize int             // This respects stack frame alignment
	Layout         []*DataLocation // from bottom to top
}

func NewStackFrame() *StackFrame {
	return &StackFrame{
		Locations:      map[string]*DataLocation{},
		LocalVariables: map[string]*DataLocation{},
	}
}

func (frame *StackFrame) UpdateMaxTempSize(size int) {
	if size > frame.MaxTempSize {
		frame.MaxTempSize = size
	}
}

func (frame *StackFrame) add(name string, valueType ast.Type) *DataLocation {
	_, ok := frame.Locations[name]
	if ok {
		panic("duplicate data location: " + name)
	}

	loc := NewFixedStackDataLocation(name, valueType)
	frame.Locations[name] = loc
	return loc
}

// Must be call before StartCurrentFrame()
func (frame *StackFrame) SetDestination(valueType ast.Type) *DataLocation {
	if frame.ReturnAddress != nil {
		panic("cannot set destination after starting current frame")
	}
	frame.Destination = frame.add(StackDestination, valueType)
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
		ReturnAddress,
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

func (frame *StackFrame) FinalizeFrame() {
	fixedSize := 0
	frameEntries := make([]*DataLocation, 0, len(frame.LocalVariables))
	for _, loc := range frame.LocalVariables {
		fixedSize += loc.AlignedSize
		frameEntries = append(frameEntries, loc)
	}

	sort.Slice(
		frameEntries,
		func(i int, j int) bool {
			cmp := CompareDefinitionNames(frameEntries[i].Name, frameEntries[j].Name)
			return cmp < 0
		})

	totalFrameSize := fixedSize + frame.MaxTempSize
	roundUp := (totalFrameSize + StackFrameAlignment - 1) / StackFrameAlignment
	frame.TotalFrameSize = roundUp * StackFrameAlignment

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
	currentOffset := frame.TotalFrameSize - fixedSize
	for idx := len(layout) - 1; idx >= 0; idx-- {
		entry := layout[idx]
		entry.Offset = currentOffset
		currentOffset += entry.AlignedSize
	}
}

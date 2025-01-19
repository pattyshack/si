package allocator

import (
	"fmt"
	"math"
	"sort"

	"github.com/pattyshack/chickadee/architecture"
	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/platform"
)

// All distances are in terms of instruction distance relative to the beginning
// of the block (phis are at distance zero).
type LiveRange struct {
	Start int // first live distance (relative to beginning of current block)
	End   int // last use distance, inclusive.

	// NextUses are always at least one instruction ahead of the current,
	// yet to be executed, instruction.
	NextUses []int
}

type PreferredAllocation struct {
	// Instruction distance where the variable is required
	Use int

	// Def could be nil since the source ast.Value could be an immediate
	// or a global label.
	Def *ast.VariableDefinition

	// Which chunk of the definition maps to this register
	ChunkIdx int
}

func (pref PreferredAllocation) String() string {
	if pref.Def != nil {
		return fmt.Sprintf(
			"%d %s %d (%s)",
			pref.Use,
			pref.Def.Name,
			pref.ChunkIdx,
			pref.Def.Loc())
	}
	return fmt.Sprintf("%d (nil) %d", pref.Use, pref.ChunkIdx)
}

// The block's execution state at a particular point in time.
type BlockState struct {
	platform.Platform
	*ast.Block

	*architecture.StackFrame

	DebugMode bool

	LiveIn  LiveSet
	LiveOut LiveSet

	Constraints map[ast.Instruction]*architecture.InstructionConstraints

	// Note: unused definitions are not included in LiveRanges
	LiveRanges map[*ast.VariableDefinition]*LiveRange

	// Preferences are always at least one instruction ahead of the current,
	// yet to be executed, instruction.
	Preferences map[*architecture.Register][]PreferredAllocation

	// Where data are located immediately prior to executing the block.
	// Every entry maps one-to-one to the corresponding live in set.
	LocationIn LocationSet

	// Where data are located immediately after the block executed.
	// Every entry maps one-to-one to the corresponding live in set.
	LocationOut LocationSet

	ValueLocations *ValueLocations

	Operations []architecture.Operation
}

func (state *BlockState) GenerateConstraints(targetPlatform platform.Platform) {
	constraints := map[ast.Instruction]*architecture.InstructionConstraints{}
	for _, inst := range state.Instructions {
		constraints[inst] = targetPlatform.InstructionConstraints(inst)
	}

	state.Constraints = constraints
}

// Note: A block's preferences cannot be precomputed since the block's
// preferences could changes due to its children's LocationIn (set by a
// different parent).
func (state *BlockState) ComputeLiveRangesAndPreferences(
	blockStates map[*ast.Block]*BlockState,
) {
	state.LiveRanges = map[*ast.VariableDefinition]*LiveRange{}
	state.Preferences = map[*architecture.Register][]PreferredAllocation{}

	defStarts := map[*ast.VariableDefinition]int{}
	for def, _ := range state.LiveIn {
		defStarts[def] = 0
	}

	for idx, inst := range state.Instructions {
		dist := idx + 1

		def := inst.Destination()
		if def != nil {
			defStarts[def] = dist
		}

		constraints := state.Constraints[inst]
		for srcIdx, src := range inst.Sources() {
			var def *ast.VariableDefinition
			ref, ok := src.(*ast.VariableReference)
			if ok {
				state.updateLiveRange(ref.UseDef, defStarts[def], dist)
			}

			// Don't collect preferences from the first instruction since we only
			// care about look ahead preferences.
			if idx == 0 {
				continue
			}

			state.collectConstraintPreferences(
				dist,
				def,
				constraints.Sources[srcIdx])
		}
	}

	// Generate preference from children LocationIn populated by other parents.
	for _, childBlock := range state.Children {
		childState := blockStates[childBlock]
		if childState.LocationIn == nil {
			continue
		}
		state.computePreferencesFromChildLocationIn(childState.LocationIn)
	}

	state.computeLiveRangesAndPreferencesFromLiveOut(blockStates, defStarts)
}

func (state *BlockState) updateLiveRange(
	def *ast.VariableDefinition,
	startDist int,
	useDist int,
) {
	live, ok := state.LiveRanges[def]
	if !ok {
		live = &LiveRange{
			Start: startDist,
		}
		state.LiveRanges[def] = live
	}

	live.End = useDist
	if useDist > 1 {
		live.NextUses = append(live.NextUses, useDist)
	}
}

func (state *BlockState) computePreferencesFromChildLocationIn(
	childLocationIn LocationSet,
) {
	sortedDefs := []*ast.VariableDefinition{}
	for def, _ := range childLocationIn {
		sortedDefs = append(sortedDefs, def)
	}

	sort.Slice(
		sortedDefs,
		func(i int, j int) bool { return sortedDefs[i].Name < sortedDefs[j].Name })

	dist := len(state.Instructions) + 1 // +1 for phi
	for _, def := range sortedDefs {
		loc := childLocationIn[def]

		for chunkIdx, reg := range loc.Registers {
			state.maybeAddPreference(reg, dist, def, chunkIdx)
		}
	}
}

func (state *BlockState) computeLiveRangesAndPreferencesFromLiveOut(
	blockStates map[*ast.Block]*BlockState,
	defStarts map[*ast.VariableDefinition]int,
) {
	type usage struct {
		inst ast.Instruction
		dist int
	}

	sortedUsages := []*usage{}
	usages := map[ast.Instruction]*usage{}
	for def, info := range state.LiveOut {
		minDist := math.MaxInt32
		maxDist := 0
		for inst, dist := range info.NextUse {
			if dist > maxDist {
				maxDist = dist
			}

			if minDist > dist {
				minDist = dist
			}

			_, ok := usages[inst]
			if !ok {
				use := &usage{
					inst: inst,
					dist: dist,
				}
				usages[inst] = use
				sortedUsages = append(sortedUsages, use)
			}
		}

		// Note: global live range could be longer, but this is a good enough
		// estimate for this block.
		start := defStarts[def]
		state.updateLiveRange(def, start, minDist)
		if minDist < maxDist {
			state.updateLiveRange(def, start, maxDist)
		}
	}

	sort.Slice(
		sortedUsages,
		func(i int, j int) bool {
			return sortedUsages[i].dist < sortedUsages[j].dist
		})

	for _, use := range sortedUsages {
		inst := use.inst
		_, ok := inst.(*ast.Phi)
		if ok { // phi copy has no preferences
			continue
		}

		constraints := blockStates[inst.ParentBlock()].Constraints[inst]
		for srcIdx, src := range inst.Sources() {
			ref, ok := src.(*ast.VariableReference)
			if !ok {
				continue
			}

			def := ref.UseDef
			_, ok = state.LiveOut[def]
			if !ok {
				continue
			}

			state.collectConstraintPreferences(
				use.dist,
				def,
				constraints.Sources[srcIdx])
		}
	}
}

func (state *BlockState) collectConstraintPreferences(
	dist int,
	def *ast.VariableDefinition,
	constraint *architecture.LocationConstraint,
) {
	for chunkIdx, candidate := range constraint.Registers {
		if candidate.Require == nil {
			continue
		}

		state.maybeAddPreference(candidate.Require, dist, def, chunkIdx)
	}
}

func (state *BlockState) maybeAddPreference(
	reg *architecture.Register,
	dist int,
	def *ast.VariableDefinition,
	chunkIdx int,
) {
	list, ok := state.Preferences[reg]
	if ok && list[len(list)-1].Use == dist {
		// confliciting preferences (just keep the first preference)
		return
	}

	state.Preferences[reg] = append(
		list,
		PreferredAllocation{
			Use:      dist,
			Def:      def,
			ChunkIdx: chunkIdx,
		})
}

func (state *BlockState) AdvanceLiveRangesAndPreferences(
	currentDist int,
) {
	nextDist := currentDist + 1

	for def, liveRange := range state.LiveRanges {
		if nextDist > liveRange.End {
			delete(state.LiveRanges, def)
			continue
		}

		for len(liveRange.NextUses) > 0 && liveRange.NextUses[0] <= nextDist {
			liveRange.NextUses = liveRange.NextUses[1:]
		}
	}

	for reg, pref := range state.Preferences {
		origLen := len(pref)
		for len(pref) > 0 && pref[0].Use <= nextDist {
			pref = pref[1:]
		}

		if origLen > len(pref) {
			state.Preferences[reg] = pref
		}
	}
}

func (state *BlockState) InitializeValueLocations() {
	state.ValueLocations = NewValueLocations(
		state.Platform,
		state.StackFrame,
		state.LocationIn)
}

func (state *BlockState) FinalizeLocationOut() {
	state.LocationOut = LocationSet{}
	for def, locs := range state.ValueLocations.Values {
		var selected *architecture.DataLocation
		for _, loc := range locs {
			if loc.OnTempStack {
				panic("should never happen")
			}

			// TODO select locations that best match predetermined children locations.
			// For now, prefer register locations over fixed stack location
			selected = loc
			if !selected.OnFixedStack {
				break
			}
		}

		if selected == nil {
			panic("should never happen")
		}

		state.LocationOut[def] = selected
	}
}

// Note: srcs must be allocated/tracked by ValueLocations.  dest's location is
// allocated/tracked by ValueLocations prior to instruction execution iff the
// location is on temp stack; register locations must be allocated/tracked by
// ValueLocations after instruction execution since destination could reuse
// source registers (dest cannot be on fixed stack).
func (state *BlockState) ExecuteInstruction(
	inst ast.Instruction,
	srcs []*architecture.DataLocation,
	dest *architecture.DataLocation,
) {
	/* TODO UNCOMMENT.

	for _, src := range srcs {
		state.ValueLocations.AssertAllocated(src)
	}
	*/

	if dest == nil {
		// Do nothing. This is a control flow instruction.
	} else if dest.OnFixedStack {
		panic("should never happen")
	} else if dest.OnTempStack {
		state.ValueLocations.AssertAllocated(dest)
	} else { // register locations
		state.ValueLocations.AssertNotAllocated(dest)
	}

	state.Operations = append(
		state.Operations,
		architecture.NewExecuteInstructionOp(inst, srcs, dest))
}

func (state *BlockState) PushStackFrame() {
	state.Operations = append(
		state.Operations,
		architecture.NewPushStackFrameOp(state.StackFrame))
}

func (state *BlockState) PopStackFrame() {
	state.Operations = append(
		state.Operations,
		architecture.NewPopStackFrameOp(state.StackFrame))
}

func (state *BlockState) MoveRegister(
	src *architecture.Register,
	dest *architecture.Register,
) *architecture.DataLocation {
	newLoc := state.ValueLocations.MoveRegister(src, dest)
	state.Operations = append(
		state.Operations,
		architecture.NewMoveRegisterOp(src, dest))
	return newLoc
}

// Note: both src and dest must be allocated/tracked by ValueLocations. scratch
// register must not be in use.
func (state *BlockState) CopyLocation(
	src *architecture.DataLocation,
	dest *architecture.DataLocation,
	scratch *architecture.Register,
) {
	state.ValueLocations.AssertAllocated(src)
	state.ValueLocations.AssertAllocated(dest)
	if src.IsOnStack() && dest.IsOnStack() && scratch == nil {
		panic("should never happen")
	}

	if scratch != nil {
		state.ValueLocations.AssertFree(scratch)
	}

	state.Operations = append(
		state.Operations,
		architecture.NewCopyLocationOp(src, dest, scratch))
}

// Note: dest must be allocated/tracked by ValueLocations. scratch register
// must not be in use.
func (state *BlockState) SetConstantValue(
	value ast.Value,
	dest *architecture.DataLocation,
	scratch *architecture.Register,
) {
	state.ValueLocations.AssertAllocated(dest)
	if dest.IsOnStack() && scratch == nil {
		panic("should never happen")
	}
	if scratch != nil {
		state.ValueLocations.AssertFree(scratch)
	}

	state.Operations = append(
		state.Operations,
		architecture.NewSetConstantValueOp(value, dest, scratch))
}

// Note: dest must be allocated/tracked by ValueLocations. scratch register must
// not be in use.
func (state *BlockState) InitializeZeros(
	dest *architecture.DataLocation,
	scratch *architecture.Register,
) {
	state.ValueLocations.AssertAllocated(dest)
	if !dest.OnTempStack {
		panic("should never happen")
	}
	state.ValueLocations.AssertFree(scratch)

	state.Operations = append(
		state.Operations,
		architecture.NewInitializeZerosOp(dest, scratch))
}

func (state *BlockState) AllocateRegistersLocation(
	def *ast.VariableDefinition,
	registers ...*architecture.Register,
) *architecture.DataLocation {
	loc := state.ValueLocations.AllocateRegistersLocation(def, registers...)
	state.Operations = append(
		state.Operations,
		architecture.NewAllocateLocationOp(loc))
	return loc
}

func (state *BlockState) AllocateFixedStackLocation(
	def *ast.VariableDefinition,
) *architecture.DataLocation {
	loc := state.ValueLocations.AllocateFixedStackLocation(def)
	state.Operations = append(
		state.Operations,
		architecture.NewAllocateLocationOp(loc))
	return loc
}

func (state *BlockState) AllocateTempStackLocations(
	argDefs []*ast.VariableDefinition,
	returnDef *ast.VariableDefinition,
) (
	[]*architecture.DataLocation,
	*architecture.DataLocation,
) {
	argLocs, returnLoc := state.ValueLocations.AllocateTempStackLocations(
		argDefs,
		returnDef)

	for _, loc := range argLocs {
		state.Operations = append(
			state.Operations,
			architecture.NewAllocateLocationOp(loc))
	}

	if returnLoc != nil {
		state.Operations = append(
			state.Operations,
			architecture.NewAllocateLocationOp(returnLoc))
	}

	return argLocs, returnLoc
}

// Note: loc must be allocated/tracked by ValueLocations.
func (state *BlockState) FreeLocation(
	loc *architecture.DataLocation,
) {
	state.ValueLocations.FreeLocation(loc)
	state.Operations = append(
		state.Operations,
		architecture.NewFreeLocationOp(loc))
}

// For internal debugging only.
func (state *BlockState) printValueLocations() {
	fmt.Println("VALUE LOCATIONS:")
	for def, locs := range state.ValueLocations.Values {
		fmt.Println("  DEFINTIION:", def.Name)
		for _, loc := range locs {
			fmt.Println("    LOCATION:", loc)
		}
	}
}

package allocator

import (
	"fmt"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/analyzer/util"
	"github.com/pattyshack/chickadee/architecture"
	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/platform"
)

// A linear scan style register/stack allocator
//
// Assumptions:
// - all general and float register are usable for data storage / spilling.
// - spilling between registers is significantly faster than spilling to memory.
type Allocator struct {
	platform.Platform

	DebugMode bool

	FuncDef     *ast.FunctionDefinition
	BlockStates map[*ast.Block]*BlockState

	*architecture.StackFrame

	nextTransferBlockId int
}

func NewAllocator(
	targetPlatform platform.Platform,
	debugMode bool,
) *Allocator {
	return &Allocator{
		Platform:            targetPlatform,
		DebugMode:           debugMode,
		BlockStates:         map[*ast.Block]*BlockState{},
		StackFrame:          architecture.NewStackFrame(),
		nextTransferBlockId: 0,
	}
}

func (allocator *Allocator) Process(entry ast.SourceEntry) {
	funcDef, ok := entry.(*ast.FunctionDefinition)
	if !ok {
		return
	}
	allocator.FuncDef = funcDef

	// Note: The allocator will insert and reorder blocks, making the original
	// jump instructions invalid.  We will need to replace all jump instructions
	// after the allocation process.
	allocator.stripExplicitUnconditionalJumps()

	allocator.initializeBlockStates()
	allocator.initializeEntryBlockLocationIn()
	allocator.StartCurrentFrame()

	dfsOrder, _ := util.DFS(funcDef)
	for _, block := range dfsOrder {
		allocator.processBlock(allocator.BlockStates[block])
	}

	allocator.maybeInsertTransferBlocks()

	allocator.FinalizeFrame()

	allocator.reorderBlocksAndUpdateJumps()
}

func (allocator *Allocator) stripExplicitUnconditionalJumps() {
	for _, block := range allocator.FuncDef.Blocks {
		if len(block.Instructions) == 0 {
			continue
		}

		strip := false
		last := block.Instructions[len(block.Instructions)-1]
		switch last.(type) {
		case *ast.Jump:
			strip = true
		case *ast.ConditionalJump:
			// Both branches ends up at the next block
			strip = len(block.Children) == 1
		}

		if strip {
			block.Instructions = block.Instructions[:len(block.Instructions)-1]
		}
	}
}

func (allocator *Allocator) reorderBlocksAndUpdateJumps() {
	reorderedBlocks, _ := util.DFS(allocator.FuncDef)
	allocator.FuncDef.Blocks = reorderedBlocks

	numBlocks := len(allocator.FuncDef.Blocks)
	for idx, block := range allocator.FuncDef.Blocks {
		switch len(block.Children) {
		case 0: // terminal block
			// sanity check
			if len(block.Instructions) == 0 {
				panic("should never happen")
			}

			last := block.Instructions[len(block.Instructions)-1]
			switch inst := last.(type) {
			case *ast.Terminal: // ok
			case *ast.FuncCall:
				if !inst.IsExitTerminal {
					panic("should never happen")
				}
			default:
				panic("should never happen")
			}
		case 1: // unconditional jump
			if idx+1 < numBlocks &&
				allocator.FuncDef.Blocks[idx+1] == block.Children[0] {

				continue // implicit jump via fallthrough
			}

			// sanity check
			if len(block.Instructions) > 0 {
				last := block.Instructions[len(block.Instructions)-1]
				_, ok := last.(ast.ControlFlowInstruction)
				if ok {
					panic("should never happen")
				}
			}

			// Insert explicit unconditional jump
			jump := &ast.Jump{
				StartEndPos: parseutil.NewStartEndPos(block.End(), block.End()),
				Label:       block.Children[0].Label,
			}
			block.Instructions = append(block.Instructions, jump)
			allocator.BlockStates[block].ExecuteInstruction(jump, nil, nil)
		case 2: // conditional jump
			if len(block.Instructions) == 0 {
				panic("should never happen")
			}

			last := block.Instructions[len(block.Instructions)-1]
			jump, ok := last.(*ast.ConditionalJump)
			if !ok {
				panic("should never happen")
			}

			// The first child is always the jump child branch
			jump.Label = block.Children[0].Label

			// The second child is always the fallthrough child branch
			if idx+1 < numBlocks &&
				allocator.FuncDef.Blocks[idx+1] == block.Children[1] {
				// ok
			} else {
				panic("should never happen")
			}
		default:
			panic("should never happen")
		}
	}
}

func (allocator *Allocator) initializeBlockStates() {
	analyzer := NewLivenessAnalyzer()
	analyzer.Process(allocator.FuncDef)

	for _, block := range allocator.FuncDef.Blocks {
		state := &BlockState{
			Platform:   allocator.Platform,
			Block:      block,
			StackFrame: allocator.StackFrame,
			DebugMode:  allocator.DebugMode,
			LiveIn:     analyzer.LiveIn[block],
			LiveOut:    analyzer.LiveOut[block],
		}
		state.GenerateConstraints()

		allocator.BlockStates[block] = state
	}
}

func (allocator *Allocator) initializeEntryBlockLocationIn() {
	spec := allocator.CallConvention(allocator.FuncDef.FuncType)
	convention := spec.CallConstraints

	if convention.Destination.RequireOnStack {
		allocator.StackFrame.SetDestination(allocator.FuncDef.ReturnType)
	}

	// The first constraint is call's function location, which is not applicable
	// to the function definition
	constraints := convention.AllSources()[1:]

	locations := LocationSet{}
	for idx, param := range allocator.FuncDef.AllParameters() {
		constraint := constraints[idx]
		if constraint.RequireOnStack {
			locations[param] = allocator.StackFrame.AddParameter(
				param.Name,
				param.Type)
		} else {
			registers := []*architecture.Register{}
			for _, reg := range constraint.Registers {
				registers = append(registers, reg.Require)
			}

			locations[param] = architecture.NewRegistersDataLocation(
				param.Name,
				param.Type,
				registers)
		}
	}

	allocator.BlockStates[allocator.FuncDef.Blocks[0]].LocationIn = locations
}

func (allocator *Allocator) processBlock(block *BlockState) {
	block.ComputeLiveRangesAndPreferences(allocator.BlockStates)

	block.InitializeValueLocations()

	for idx, inst := range block.Instructions {
		currentDist := idx + 1 // +1 for phi
		scheduler := newOperationsScheduler(block, currentDist)
		scheduler.ScheduleInstructionOperations(inst, block.Constraints[inst])
		block.AdvanceLiveRangesAndPreferences(currentDist)
	}

	for _, child := range block.Children {
		allocator.maybeInitializeChildBlockLocationIn(
			block,
			allocator.BlockStates[child])
	}
}

func (allocator *Allocator) maybeInsertTransferBlocks() {
	for _, child := range allocator.FuncDef.Blocks[:] {
		if len(child.Parents) <= 1 {
			// All data must already be at the correct locations.  Note that only
			// the entry block has no parent; the rest have exactly one parent.
			continue
		}

		childState := allocator.BlockStates[child]
		for _, parent := range child.Parents {
			allocator.maybeInsertTransferBlock(
				allocator.BlockStates[parent],
				childState)
		}
	}
}

// NOTE: This deconstructs SSA PHIs for the case where value locations are
// unconstrained / do not require relocation.
func (allocator *Allocator) maybeInitializeChildBlockLocationIn(
	parent *BlockState,
	child *BlockState,
) {
	if child.LocationIn != nil {
		return
	}

	defs := make(map[string]*ast.VariableDefinition, len(child.LiveIn))
	for def, _ := range child.LiveIn {
		defs[def.Name] = def
	}

	locationIn := LocationSet{}
	for parentDef, locs := range parent.ValueLocations.Values {
		def, ok := defs[parentDef.Name]
		if !ok {
			continue
		}

		var selected *architecture.DataLocation
		for _, loc := range locs {
			if loc.OnTempStack {
				panic("should never happen")
			}

			if selected == nil || selected.OnFixedStack {
				selected = loc
			} else if len(loc.Registers) > 0 &&
				selected.Registers[0].Index > loc.Registers[0].Index {

				selected = loc
			}
		}

		locationIn[def] = selected.Copy()
	}
	child.LocationIn = locationIn
}

// If the parent's value locations does not match the child's expected
// LocationIn data locations, this inserts a transfer block between
// the parent and child, and the transfer block will move the data to the
// expected locations.
//
// This assumes that both parent are child are already processed.
//
// NOTE: This deconstructs SSA PHIs for the case where value locations are
// constrained.
func (allocator *Allocator) maybeInsertTransferBlock(
	parent *BlockState,
	child *BlockState,
) {
	transferBlock := &ast.Block{
		StartEndPos: parseutil.NewStartEndPos(parent.End(), child.Loc()),
		Label: fmt.Sprintf(
			":transfer-block-%d",
			allocator.nextTransferBlockId),
		ParentFuncDef: allocator.FuncDef,
		Parents:       []*ast.Block{parent.Block},
		Children:      []*ast.Block{child.Block},
	}
	allocator.nextTransferBlockId++

	defs := map[string]*ast.VariableDefinition{}
	for def, _ := range child.LocationIn {
		defs[def.Name] = def
	}

	// This is the SSA deconstruction "copy" step.  Note that the data may be in
	// in the wrong location.
	locations := NewValueLocations(allocator.Platform, allocator.StackFrame)
	for parentDef, locs := range parent.ValueLocations.Values {
		def, ok := defs[parentDef.Name]
		if !ok { // definition not used by this child
			continue
		}

		for _, loc := range locs {
			if loc.OnFixedStack {
				locations.AllocateFixedStackLocation(def)
			} else if loc.OnTempStack {
				panic("should never happen")
			} else {
				locations.AllocateRegistersLocation(def, loc.Registers...)
			}
		}
	}

	// NOTE: LiveIn, LiveOut, LiveRanges, LocationIn, Constraints, and
	// Preferences are not populated.
	transferState := &BlockState{
		Platform:       allocator.Platform,
		Block:          transferBlock,
		StackFrame:     allocator.StackFrame,
		DebugMode:      allocator.DebugMode,
		ValueLocations: locations,
	}

	scheduler := newOperationsScheduler(transferState, 0)
	scheduler.ScheduleTransferBlockOperations(child.LocationIn)

	hasRealOperations := false
	for _, op := range transferState.Operations {
		if op.Kind != architecture.FreeLocation {
			hasRealOperations = true
			break
		}
	}

	if !hasRealOperations {
		// All data are already in correct locations.  No Need to insert a
		// transfer block
		return
	}

	// Insert the transfer block and update control flow graph edges

	allocator.FuncDef.Blocks = append(allocator.FuncDef.Blocks, transferBlock)
	allocator.BlockStates[transferBlock] = transferState

	numModified := 0
	for idx, block := range parent.Children {
		if child.Block == block {
			parent.Children[idx] = transferBlock
			numModified++
		}
	}
	if numModified != 1 {
		panic("should never happen")
	}

	numModified = 0
	for idx, block := range child.Parents {
		if parent.Block == block {
			child.Parents[idx] = transferBlock
			numModified++
		}
	}
	if numModified != 1 {
		panic("should never happen")
	}
}

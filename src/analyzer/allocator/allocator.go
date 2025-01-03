package allocator

import (
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
}

func NewAllocator(
	targetPlatform platform.Platform,
	debugMode bool,
) *Allocator {
	return &Allocator{
		Platform:    targetPlatform,
		DebugMode:   debugMode,
		BlockStates: map[*ast.Block]*BlockState{},
		StackFrame:  architecture.NewStackFrame(),
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
	allocator.initializeFuncDefDataLocations()
	allocator.StartCurrentFrame()

	dfsOrder, _ := util.DFS(funcDef)
	for _, block := range dfsOrder {
		allocator.processBlock(allocator.BlockStates[block])
	}

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
			_, ok := last.(*ast.Terminal)
			if !ok {
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
			allocator.BlockStates[block].ExecuteInstruction(nil, nil)
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
		state.GenerateConstraints(allocator.Platform)

		allocator.BlockStates[block] = state
	}
}

func (allocator *Allocator) initializeFuncDefDataLocations() {
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

	// TODO actual allocator implementation.
	// XXX: The following is only used for debugging stack frame's implementation

	for _, inst := range block.Instructions {
		for _, src := range inst.Sources() {
			ref, ok := src.(*ast.VariableReference)
			if !ok {
				continue
			}
			allocator.StackFrame.MaybeAddLocalVariable(ref.Name, ref.Type())
		}
	}
}

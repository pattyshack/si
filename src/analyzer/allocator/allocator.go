package allocator

import (
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

	*StackFrame
}

func NewAllocator(
	targetPlatform platform.Platform,
	debugMode bool,
) *Allocator {
	return &Allocator{
		Platform:    targetPlatform,
		DebugMode:   debugMode,
		BlockStates: map[*ast.Block]*BlockState{},
		StackFrame:  NewStackFrame(),
	}
}

func (allocator *Allocator) Process(entry ast.SourceEntry) {
	funcDef, ok := entry.(*ast.FunctionDefinition)
	if !ok {
		return
	}

	allocator.FuncDef = funcDef
	allocator.initializeBlockStates()
	allocator.initializeFuncDefDataLocations()
	allocator.StartCurrentFrame()

	dfsOrder, _ := util.DFS(funcDef)
	for _, block := range dfsOrder {
		allocator.processBlock(allocator.BlockStates[block])
	}

	allocator.FinalizeFrame(allocator.Platform)
}

func (allocator *Allocator) initializeBlockStates() {
	analyzer := NewLivenessAnalyzer()
	analyzer.Process(allocator.FuncDef)

	for _, block := range allocator.FuncDef.Blocks {
		state := &BlockState{
			Platform:  allocator.Platform,
			Block:     block,
			DebugMode: allocator.DebugMode,
			LiveIn:    analyzer.LiveIn[block],
			LiveOut:   analyzer.LiveOut[block],
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

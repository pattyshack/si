package allocator

import (
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

	BlockStates map[*ast.Block]*BlockState

	*StackFrame
}

func NewAllocator(targetPlatform platform.Platform) *Allocator {
	return &Allocator{
		Platform:    targetPlatform,
		BlockStates: map[*ast.Block]*BlockState{},
		StackFrame:  NewStackFrame(),
	}
}

func (allocator *Allocator) Process(entry ast.SourceEntry) {
	funcDef, ok := entry.(*ast.FunctionDefinition)
	if !ok {
		return
	}

	allocator.analyzeLiveness(funcDef)
	allocator.initializeFuncDefDataLocations(funcDef)
	allocator.StartCurrentFrame()

	// TODO actual allocator implementation.
	// XXX: The following is only used for debugging stack frame's implementation
	for _, param := range funcDef.AllParameters() {
		allocator.StackFrame.MaybeAddLocalVariable(param.Name, param.Type)
	}

	for _, block := range funcDef.Blocks {
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

	allocator.FinalizeFrame(allocator.Platform)
}

func (allocator *Allocator) analyzeLiveness(
	funcDef *ast.FunctionDefinition,
) {
	analyzer := NewLivenessAnalyzer()
	analyzer.Process(funcDef)

	for _, block := range funcDef.Blocks {
		allocator.BlockStates[block] = &BlockState{
			LiveIn:  analyzer.LiveIn[block],
			LiveOut: analyzer.LiveOut[block],
		}
	}
}

func (allocator *Allocator) initializeFuncDefDataLocations(
	funcDef *ast.FunctionDefinition,
) {
	convention := funcDef.CallConventionSpec.CallConstraints

	if convention.Destination.RequireOnStack {
		allocator.StackFrame.SetDestination(funcDef.ReturnType)
	}

	// The first constraint is call's function location, which is not applicable
	// to the function definition
	constraints := convention.AllSources()[1:]

	locations := LocationSet{}
	for idx, param := range funcDef.AllParameters() {
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

			locations[param] = NewRegistersDataLocation(
				param.Name,
				param.Type,
				registers)
		}
	}

	allocator.BlockStates[funcDef.Blocks[0]].LocationIn = locations
}

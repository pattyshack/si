package ast

import (
	"github.com/pattyshack/gt/parseutil"
)

type FuncDefinition struct {
	sourceEntry

	parseutil.StartEndPos

	ParseError error // only set by parser

	Label      string
	Parameters []*RegisterDefinition
	ReturnType Type
	Blocks     []*Block
}

var _ SourceEntry = &FuncDefinition{}
var _ Validator = &FuncDefinition{}

func (def *FuncDefinition) Walk(visitor Visitor) {
	visitor.Enter(def)
	for _, param := range def.Parameters {
		param.Walk(visitor)
	}
	def.ReturnType.Walk(visitor)
	for _, block := range def.Blocks {
		block.Walk(visitor)
	}
	visitor.Exit(def)
}

func (def *FuncDefinition) Validate(emitter *parseutil.Emitter) {
	if def.Label == "" {
		emitter.Emit(def.Loc(), "empty function definition label string")
	}

	if len(def.Blocks) == 0 {
		emitter.Emit(def.Loc(), "function definition must have at least one block")
	}

	names := map[string]*RegisterDefinition{}
	for _, param := range def.Parameters {
		prev, ok := names[param.Name]
		if ok {
			emitter.Emit(
				param.Loc(),
				"parameter (%s) previously defined at (%s)",
				param.Name,
				prev.Loc().ShortString())
		} else {
			names[param.Name] = param
		}

		if param.Type == nil {
			emitter.Emit(
				param.Loc(),
				"function parameter (%s) must be explicitly typed",
				param.Name)
		}
	}
}

// A straight-line / basic block
type Block struct {
	parseutil.StartEndPos

	Label string

	// NOTE: only the last instruction can be a control flow instruction.  All
	// other instructions must be operation instructions.  If no control flow
	// instruction is provided, the block implicitly fallthrough to the next
	// block.
	Instructions []Instruction

	// internal

	// Populated by ControlFlowGraphInitializer.
	Parents  []*Block
	Children []*Block

	Phis map[string]*Phi
}

var _ Node = &Block{}
var _ Validator = &Block{}

func (Block) isNode() {}

func (block *Block) Walk(visitor Visitor) {
	visitor.Enter(block)
	for _, phi := range block.Phis {
		phi.Walk(visitor)
	}
	for _, instruction := range block.Instructions {
		instruction.Walk(visitor)
	}
	visitor.Exit(block)
}

func (block *Block) Validate(emitter *parseutil.Emitter) {
	if len(block.Instructions) == 0 {
		emitter.Emit(block.Loc(), "block must have at least one instruction")
		return
	}

	for idx, in := range block.Instructions {
		switch inst := in.(type) {
		case ControlFlowInstruction:
			if idx != len(block.Instructions)-1 {
				emitter.Emit(
					inst.Loc(),
					"control flow instruction must be the last instruction in the block")
			}
		case *Phi:
			emitter.Emit(inst.Loc(), "phi cannot be used as a regular instruction")
		}
	}
}

func (block *Block) AddToPhis(parent *Block, def *RegisterDefinition) {
	if block.Phis == nil {
		block.Phis = map[string]*Phi{}
	}

	phi, ok := block.Phis[def.Name]
	if !ok {
		pos := parseutil.NewStartEndPos(block.Loc(), block.Loc())
		phi = &Phi{
			StartEndPos: pos,
			Dest: &RegisterDefinition{
				StartEndPos: pos,
				Name:        def.Name,
			},
			Srcs: map[*Block]Value{},
		}
		phi.Parent = block
		block.Phis[def.Name] = phi
	}

	phi.Add(parent, def)
}

// For internal use only
type Phi struct {
	instruction

	parseutil.StartEndPos

	Dest *RegisterDefinition

	// Value is usually a register reference, but could be constant after
	// optimization.
	Srcs map[*Block]Value
}

var _ Instruction = &Phi{}

func (phi *Phi) Walk(visitor Visitor) {
	visitor.Enter(phi)
	phi.Dest.Walk(visitor)
	for _, src := range phi.Srcs {
		src.Walk(visitor)
	}
	visitor.Exit(phi)
}

func (phi *Phi) replaceSource(oldVal Value, newVal Value) {
	replaceCount := 0
	for block, src := range phi.Srcs {
		if src == oldVal {
			phi.Srcs[block] = newVal
			replaceCount++
		}
	}

	if replaceCount != 1 {
		panic("should never happen")
	}
}

func (phi *Phi) Sources() []Value {
	result := make([]Value, 0, len(phi.Srcs))
	for _, src := range phi.Srcs {
		result = append(result, src)
	}
	return result
}

func (phi *Phi) Destination() *RegisterDefinition {
	return phi.Dest
}

func (phi *Phi) Add(parent *Block, def *RegisterDefinition) {
	ref := def.NewRef(phi.StartEnd())
	ref.SetParent(phi)
	phi.Srcs[parent] = ref
}

func (phi *Phi) Discard() {
	delete(phi.Parent.Phis, phi.Dest.Name)
	for _, src := range phi.Srcs {
		src.Discard()
	}
}

package ast

import (
	"github.com/pattyshack/gt/parseutil"
)

type DeclarationKind string

const (
	DataDeclaration = DeclarationKind("data")
	FuncDeclaration = DeclarationKind("func")
)

type Declaration struct {
	sourceEntry

	parseutil.StartEndPos

	Kind DeclarationKind

	Label string
	Type  Type
}

var _ SourceEntry = &Declaration{}
var _ Validator = &Declaration{}

func (decl *Declaration) Walk(visitor Visitor) {
	visitor.Enter(decl)
	decl.Type.Walk(visitor)
	visitor.Exit(decl)
}

func (decl *Declaration) Validate(emitter *parseutil.Emitter) {
	switch decl.Kind {
	case DataDeclaration, FuncDeclaration: // ok
	default:
		emitter.Emit(decl.Loc(), "unexpected declaration kind (%s)", decl.Kind)
	}

	if decl.Label == "" {
		emitter.Emit(decl.Loc(), "empty label string")
	}
}

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

	lastBlock := def.Blocks[len(def.Blocks)-1]
	// Note: block with no instruction is also invalid, but is checked by
	// Block.Validate
	if len(lastBlock.Instructions) > 0 {
		lastInstruction := lastBlock.Instructions[len(lastBlock.Instructions)-1]
		_, ok := lastInstruction.(ControlFlowInstruction)
		if !ok {
			// There's no next block to fallthrough to
			emitter.Emit(
				lastInstruction.Loc(),
				"the last instruction in function definition must be a "+
					"control flow instruction")
		}
	}
}

// A straight-line / basic block
type Block struct {
	parseutil.StartEndPos

	Label LocalLabel

	// NOTE: only the last instruction can be a control flow instruction.  All
	// other instructions must be operation instructions.  If no control flow
	// instruction is provided, the block implicitly fallthrough to the next
	// block.
	Instructions []Instruction

	// internal

	Parents  []*Block
	Children []*Block

	// TODO phi functions
}

var _ Node = &Block{}
var _ Validator = &Block{}

func (Block) isNode() {}

func (block *Block) Walk(visitor Visitor) {
	visitor.Enter(block)
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

	for idx, instruction := range block.Instructions {
		_, ok := instruction.(ControlFlowInstruction)
		if ok && idx != len(block.Instructions)-1 {
			emitter.Emit(
				instruction.Loc(),
				"control flow instruction must be the last instruction in the block")
		}
	}
}

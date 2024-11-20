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

	Label GlobalLabel
	Type  Type
}

var _ SourceEntry = &Declaration{}

func (decl *Declaration) Walk(visitor Visitor) {
	visitor.Enter(decl)
	decl.Type.Walk(visitor)
	visitor.Exit(decl)
}

type FuncDefinition struct {
	sourceEntry

	parseutil.StartEndPos

	ParseError error // only set by parser

	Label      GlobalLabel
	Parameters []*RegisterDefinition
	ReturnType Type
	Blocks     []*Block
}

var _ SourceEntry = &FuncDefinition{}

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

func (Block) isNode() {}

func (block *Block) Walk(visitor Visitor) {
	visitor.Enter(block)
	for _, instruction := range block.Instructions {
		instruction.Walk(visitor)
	}
	visitor.Exit(block)
}

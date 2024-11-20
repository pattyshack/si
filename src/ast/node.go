package ast

import (
	"github.com/pattyshack/gt/parseutil"
)

type Node interface {
	parseutil.Locatable
	Walk(Visitor)
}

type Visitor interface {
	Enter(Node)
	Exit(Node)
}

type Line interface { // used only by the parser
	IsLine()
}

type Instruction interface {
	Node
	Line
	isInstruction()
}

type instruction struct {
}

func (instruction) IsLine()        {}
func (instruction) isInstruction() {}

type ControlFlowInstruction interface {
	Instruction
	isControlFlow()
}

type controlFlowInstruction struct {
	instruction
}

func (controlFlowInstruction) isControlFlow() {}

type Type interface {
	Node
	isTypeExpr()
}

type isType struct{}

func (isType) isTypeExpr() {}

type SourceEntry interface {
	Node
	Line
	isSourceEntry()
}

type sourceEntry struct {
}

func (sourceEntry) IsLine()        {}
func (sourceEntry) isSourceEntry() {}

// @-prefixed label for various definitions/declarations.  Note that the '@'
// prefix is not part of the name and is only used by the parser.
type GlobalLabel string

type GlobalLabelReference struct {
	parseutil.StartEndPos

	Label GlobalLabel
}

func (GlobalLabelReference) isValue() {}

func (ref *GlobalLabelReference) Walk(visitor Visitor) {
	visitor.Enter(ref)
	visitor.Exit(ref)
}

// :-prefixed block label.  Note that the ':' prefix is not part of the name
// and is only used by the parser.
type LocalLabel string

// %-prefixed local register variable definition.  Note that the '%' prefix is
// not part of the name and is only used by the parser.
type RegisterDefinition struct {
	parseutil.StartEndPos

	Name string // require

	Type Type // optional. Type is check/inferred during type checking

	DefUses map[*RegisterReference]struct{} // internal. Set by ssa construction
}

func (def *RegisterDefinition) Walk(visitor Visitor) {
	visitor.Enter(def)
	if def.Type != nil {
		def.Type.Walk(visitor)
	}
	visitor.Exit(def)
}

// Register, global label, or immediate
type Value interface {
	Node
	isValue()
}

// %-prefixed local register variable reference.  Note that the '%' prefix is
// not part of the name and is only used by the parser.
type RegisterReference struct {
	parseutil.StartEndPos

	Name string // require

	UseDef *RegisterDefinition // internal. Set by ssa construction
}

func (RegisterReference) isValue() {}

func (ref *RegisterReference) Walk(visitor Visitor) {
	visitor.Enter(ref)
	visitor.Exit(ref)
}

type Immediate struct {
	parseutil.StartEndPos

	Value   string
	IsFloat bool // false if int
}

func (Immediate) isValue() {}

func (imm *Immediate) Walk(visitor Visitor) {
	visitor.Enter(imm)
	visitor.Exit(imm)
}

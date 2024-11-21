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

type Validator interface {
	Validate(*parseutil.Emitter)
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
type GlobalLabelReference struct {
	parseutil.StartEndPos

	Label string
}

var _ Node = &GlobalLabelReference{}
var _ Validator = &GlobalLabelReference{}

func (GlobalLabelReference) isValue() {}

func (ref *GlobalLabelReference) Walk(visitor Visitor) {
	visitor.Enter(ref)
	visitor.Exit(ref)
}

func (ref *GlobalLabelReference) Validate(emitter *parseutil.Emitter) {
	if ref.Label == "" {
		emitter.Emit(ref.Loc(), "empty global label name")
	}
}

// %-prefixed local register variable definition.  Note that the '%' prefix is
// not part of the name and is only used by the parser.
type RegisterDefinition struct {
	parseutil.StartEndPos

	Name string // require

	Type Type // optional. Type is check/inferred during type checking

	DefUses map[*RegisterReference]struct{} // internal. Set by ssa construction
}

var _ Node = &RegisterDefinition{}
var _ Validator = &RegisterDefinition{}

func (def *RegisterDefinition) Walk(visitor Visitor) {
	visitor.Enter(def)
	if def.Type != nil {
		def.Type.Walk(visitor)
	}
	visitor.Exit(def)
}

func (def *RegisterDefinition) Validate(emitter *parseutil.Emitter) {
	if def.Name == "" {
		emitter.Emit(def.Loc(), "empty register definition name")
	}
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

var _ Node = &RegisterReference{}
var _ Validator = &RegisterReference{}

func (RegisterReference) isValue() {}

func (ref *RegisterReference) Walk(visitor Visitor) {
	visitor.Enter(ref)
	visitor.Exit(ref)
}

func (ref *RegisterReference) Validate(emitter *parseutil.Emitter) {
	if ref.Name == "" {
		emitter.Emit(ref.Loc(), "empty register reference name")
	}
}

type IntImmediate struct {
	parseutil.StartEndPos

	Value int64
}

func (IntImmediate) isValue() {}

func (imm *IntImmediate) Walk(visitor Visitor) {
	visitor.Enter(imm)
	visitor.Exit(imm)
}

type FloatImmediate struct {
	parseutil.StartEndPos

	Value float64
}

func (FloatImmediate) isValue() {}

func (imm *FloatImmediate) Walk(visitor Visitor) {
	visitor.Enter(imm)
	visitor.Exit(imm)
}

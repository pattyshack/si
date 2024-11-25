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

type SourceEntry interface {
	Node
	Line
	isSourceEntry()
}

type sourceEntry struct {
}

func (sourceEntry) IsLine()        {}
func (sourceEntry) isSourceEntry() {}

// %-prefixed local register variable definition.  Note that the '%' prefix is
// not part of the name and is only used by the parser.
type RegisterDefinition struct {
	parseutil.StartEndPos

	Name string // require

	Type Type // optional. Type is check/inferred during type checking

	// Internal (set during ssa construction)
	Parent  Instruction // nil for phi and func parameters
	DefUses map[*RegisterReference]struct{}
}

var _ Node = &RegisterDefinition{}
var _ Validator = &RegisterDefinition{}

func (def *RegisterDefinition) ReplaceReferencesWith(value Value) {
	for ref, _ := range def.DefUses {
		ref.ReplaceWith(value)
	}
}

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

	if def.Type != nil {
		validateUsableType(def.Type, emitter)
	}
}

func (def *RegisterDefinition) AddRef(ref *RegisterReference) {
	if def.DefUses == nil {
		def.DefUses = map[*RegisterReference]struct{}{}
	}

	def.DefUses[ref] = struct{}{}
	ref.UseDef = def
}

func (def *RegisterDefinition) NewRef(
	pos parseutil.StartEndPos,
) *RegisterReference {
	ref := &RegisterReference{
		StartEndPos: pos,
		Name:        def.Name,
	}
	def.AddRef(ref)
	return ref
}

// Register, global label, or immediate
type Value interface {
	Node
	isValue()

	// What this reference refers to.  For now:
	// - register reference returns a *RegisterDefinition
	// - global label reference returns a string
	// - immediate returns an int / float
	Definition() interface{}

	// NOTE: A copy of newVal, not newVal itself, is used to replace the
	// original value (current object).  The current object is discarded as part
	// of this call.
	ReplaceWith(newVal Value)

	Copy(pos parseutil.StartEndPos) Value
	Discard() // clear graph node references

	SetParent(Instruction)
}

type value struct {
	// Internal (set during ssa construction)
	Parent Instruction
}

func (val *value) Discard() {
	val.Parent = nil
}

func (val *value) SetParent(ins Instruction) {
	val.Parent = ins
}

// @-prefixed label for various definitions/declarations.  Note that the '@'
// prefix is not part of the name and is only used by the parser.
type GlobalLabelReference struct {
	value
	parseutil.StartEndPos

	Label string
}

var _ Node = &GlobalLabelReference{}
var _ Validator = &GlobalLabelReference{}
var _ Value = &GlobalLabelReference{}

func (GlobalLabelReference) isValue() {}

func (ref *GlobalLabelReference) Definition() interface{} {
	return ref.Label
}

func (ref *GlobalLabelReference) ReplaceWith(newVal Value) {
	newVal = newVal.Copy(ref.StartEnd())
	newVal.SetParent(ref.Parent)
	ref.Parent.replaceSource(ref, newVal)
	ref.Discard()
}

func (ref *GlobalLabelReference) Copy(pos parseutil.StartEndPos) Value {
	copied := *ref
	copied.StartEndPos = pos
	return &copied
}

func (ref *GlobalLabelReference) Walk(visitor Visitor) {
	visitor.Enter(ref)
	visitor.Exit(ref)
}

func (ref *GlobalLabelReference) Validate(emitter *parseutil.Emitter) {
	if ref.Label == "" {
		emitter.Emit(ref.Loc(), "empty global label name")
	}
}

// %-prefixed local register variable reference.  Note that the '%' prefix is
// not part of the name and is only used by the parser.
type RegisterReference struct {
	value
	parseutil.StartEndPos

	Name string // require

	// Internal (set during ssa construction)
	UseDef *RegisterDefinition
}

var _ Node = &RegisterReference{}
var _ Validator = &RegisterReference{}
var _ Value = &RegisterReference{}

func (RegisterReference) isValue() {}

func (ref *RegisterReference) Definition() interface{} {
	return ref.UseDef
}

func (ref *RegisterReference) ReplaceWith(newVal Value) {
	newVal = newVal.Copy(ref.StartEnd())
	newVal.SetParent(ref.Parent)
	ref.Parent.replaceSource(ref, newVal)
	ref.Discard()
}

func (ref *RegisterReference) Copy(pos parseutil.StartEndPos) Value {
	return ref.UseDef.NewRef(pos)
}

func (ref *RegisterReference) Discard() {
	delete(ref.UseDef.DefUses, ref)
	ref.UseDef = nil
	ref.Parent = nil
}

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
	value
	parseutil.StartEndPos

	Value int64
}

var _ Value = &IntImmediate{}

func (IntImmediate) isValue() {}

func (imm *IntImmediate) Definition() interface{} {
	return imm.Value
}

func (imm *IntImmediate) ReplaceWith(newVal Value) {
	newVal = newVal.Copy(imm.StartEnd())
	newVal.SetParent(imm.Parent)
	imm.Parent.replaceSource(imm, newVal)
	imm.Discard()
}

func (imm *IntImmediate) Copy(pos parseutil.StartEndPos) Value {
	copied := *imm
	copied.StartEndPos = pos
	return &copied
}

func (imm *IntImmediate) Walk(visitor Visitor) {
	visitor.Enter(imm)
	visitor.Exit(imm)
}

type FloatImmediate struct {
	value
	parseutil.StartEndPos

	Value float64
}

var _ Value = &FloatImmediate{}

func (FloatImmediate) isValue() {}

func (imm *FloatImmediate) Definition() interface{} {
	return imm.Value
}

func (imm *FloatImmediate) ReplaceWith(newVal Value) {
	newVal = newVal.Copy(imm.StartEnd())
	newVal.SetParent(imm.Parent)
	imm.Parent.replaceSource(imm, newVal)
	imm.Discard()
}

func (imm *FloatImmediate) Copy(pos parseutil.StartEndPos) Value {
	copied := *imm
	copied.StartEndPos = pos
	return &copied
}

func (imm *FloatImmediate) Walk(visitor Visitor) {
	visitor.Enter(imm)
	visitor.Exit(imm)
}

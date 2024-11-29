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

	// Internal

	Type() Type
}

type sourceEntry struct {
}

func (sourceEntry) IsLine()        {}
func (sourceEntry) isSourceEntry() {}

// %-prefixed local register variable definition.  Note that the '%' prefix is
// not part of the name and is only used by the parser.
type VariableDefinition struct {
	parseutil.StartEndPos

	Name string // require

	Type Type // optional. Type is check/inferred during type checking

	// Internal (set during ssa construction)
	Parent  Instruction // nil for phi and func parameters
	DefUses map[*VariableReference]struct{}
}

var _ Node = &VariableDefinition{}
var _ Validator = &VariableDefinition{}

func (def *VariableDefinition) ReplaceReferencesWith(value Value) {
	for ref, _ := range def.DefUses {
		ref.ReplaceWith(value)
	}
}

func (def *VariableDefinition) Walk(visitor Visitor) {
	visitor.Enter(def)
	if def.Type != nil {
		def.Type.Walk(visitor)
	}
	visitor.Exit(def)
}

func (def *VariableDefinition) Validate(emitter *parseutil.Emitter) {
	if def.Name == "" {
		emitter.Emit(def.Loc(), "empty register definition name")
	}

	if def.Type != nil {
		validateUsableType(def.Type, emitter)
	}
}

func (def *VariableDefinition) AddRef(ref *VariableReference) {
	if def.DefUses == nil {
		def.DefUses = map[*VariableReference]struct{}{}
	}

	def.DefUses[ref] = struct{}{}
	ref.UseDef = def
}

func (def *VariableDefinition) NewRef(
	pos parseutil.StartEndPos,
) *VariableReference {
	ref := &VariableReference{
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

	// Internal

	// What this reference refers to.  For now:
	// - register reference returns a *VariableDefinition
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

	Type() Type
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

	// Internal

	Signature SourceEntry
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

func (ref *GlobalLabelReference) Type() Type {
	if ref.Signature == nil { // failed named binding
		return NewErrorType(ref.StartEndPos)
	}
	return ref.Signature.Type()
}

// %-prefixed local register variable reference.  Note that the '%' prefix is
// not part of the name and is only used by the parser.
type VariableReference struct {
	value
	parseutil.StartEndPos

	Name string // require

	// Internal (set during ssa construction)
	UseDef *VariableDefinition
}

var _ Node = &VariableReference{}
var _ Validator = &VariableReference{}
var _ Value = &VariableReference{}

func (VariableReference) isValue() {}

func (ref *VariableReference) Definition() interface{} {
	return ref.UseDef
}

func (ref *VariableReference) ReplaceWith(newVal Value) {
	newVal = newVal.Copy(ref.StartEnd())
	newVal.SetParent(ref.Parent)
	ref.Parent.replaceSource(ref, newVal)
	ref.Discard()
}

func (ref *VariableReference) Copy(pos parseutil.StartEndPos) Value {
	return ref.UseDef.NewRef(pos)
}

func (ref *VariableReference) Discard() {
	delete(ref.UseDef.DefUses, ref)
	ref.UseDef = nil
	ref.Parent = nil
}

func (ref *VariableReference) Walk(visitor Visitor) {
	visitor.Enter(ref)
	visitor.Exit(ref)
}

func (ref *VariableReference) Validate(emitter *parseutil.Emitter) {
	if ref.Name == "" {
		emitter.Emit(ref.Loc(), "empty register reference name")
	}
}

func (ref *VariableReference) Type() Type {
	if ref.UseDef == nil { // failed named binding
		return NewErrorType(ref.StartEndPos)
	}
	return ref.UseDef.Type
}

type IntImmediate struct {
	value
	parseutil.StartEndPos

	Value      uint64
	IsNegative bool
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

func (imm *IntImmediate) Type() Type {
	if imm.IsNegative {
		return NegativeIntLiteralType{
			StartEndPos: imm.StartEndPos,
		}
	} else {
		return PositiveIntLiteralType{
			StartEndPos: imm.StartEndPos,
		}
	}
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

func (imm *FloatImmediate) Type() Type {
	return FloatLiteralType{
		StartEndPos: imm.StartEndPos,
	}
}

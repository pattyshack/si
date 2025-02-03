package ast

import (
	"strings"

	"github.com/pattyshack/gt/parseutil"
)

type CallConventionName string

const (
	DefaultCallConvention = InternalCalleeSavedCallConvention

	// NOTE: Full C / System V ABI compatibility is not a priority.  It is
	// needlessly complicated for our purpose (e.g., 128 int/float, aggregate
	// type	classification/splitting, vararg, etc.).  We'll just pick something
	// simple to implement for now.  Note that the internal/default convention
	// is unstable.
	InternalCallConvention = CallConventionName("internal")

	// All arguments and destination are pass via stack.  All registers are
	// callee saved (the first register holds the frame pointer).
	InternalCalleeSavedCallConvention = CallConventionName(
		"internal-callee-saved")

	// All registers are caller saved.  Mainly used for testing.
	InternalCallerSavedCallConvention = CallConventionName(
		"internal-caller-saved")

	// This call convention implements a subset of the SystemV ABI; specifically,
	// it only supports primitive argument and return types (ints, floats and
	// pointers).  This in theory is the bare minimal needed for accessing C
	// functions.
	//
	// The following features are not supported (the list is not exhastive):
	// - 128 byte int/float value
	// - aggregate type as argument / return value
	// - c++ conventions
	// - vararg
	SystemVLiteCallConvention = CallConventionName("SystemV-lite")
)

func (call CallConventionName) isValid() bool {
	switch call {
	case InternalCallConvention,
		InternalCalleeSavedCallConvention,
		InternalCallerSavedCallConvention,
		SystemVLiteCallConvention:
		return true
	default:
		return false
	}
}

type FunctionDefinition struct {
	sourceEntry

	parseutil.StartEndPos

	// TODO: add option to specify call convention via lr grammar
	CallConventionName

	Label      string
	Parameters []*VariableDefinition
	ReturnType Type
	Blocks     []*Block

	// Internal

	FuncType *FunctionType

	// Non-argument callee-saved registers are treated as pseudo/hidden
	// parameters that are alive for the entire function execution, and are
	// "returned" as part of the ret instruction.
	PseudoParameters []*VariableDefinition

	// All callee-saved variable definitions from Parameters and PseudoParameters
	CalleeSavedParameters []*VariableDefinition
}

var _ SourceEntry = &FunctionDefinition{}
var _ Validator = &FunctionDefinition{}

func (def *FunctionDefinition) AllParameters() []*VariableDefinition {
	result := make(
		[]*VariableDefinition,
		0,
		len(def.Parameters)+len(def.PseudoParameters))
	result = append(result, def.Parameters...)
	result = append(result, def.PseudoParameters...)
	return result
}

func (def *FunctionDefinition) Walk(visitor Visitor) {
	visitor.Enter(def)
	for _, param := range def.Parameters {
		param.Walk(visitor)
	}
	for _, param := range def.PseudoParameters {
		param.Walk(visitor)
	}
	def.ReturnType.Walk(visitor)
	for _, block := range def.Blocks {
		block.Walk(visitor)
	}
	visitor.Exit(def)
}

func (def *FunctionDefinition) Validate(emitter *parseutil.Emitter) {
	if def.Label == "" {
		emitter.Emit(def.Loc(), "empty function definition label string")
	}

	if !def.CallConventionName.isValid() {
		emitter.Emit(
			def.Loc(),
			"unsupported call convention (%s)",
			def.CallConventionName)
	}

	if len(def.Blocks) == 0 {
		emitter.Emit(def.Loc(), "function definition must have at least one block")
	}

	names := map[string]*VariableDefinition{}
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

	validateUsableType(def.ReturnType, emitter)
}

func (def *FunctionDefinition) Type() Type {
	return def.FuncType
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

	ParentFuncDef *FunctionDefinition

	// Populated by ControlFlowGraphInitializer.
	Parents []*Block
	// The jump child branch (if exist) is always before the fallthrough child
	// branch (if exist).
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
	if strings.HasPrefix(block.Label, ":") {
		emitter.Emit(block.Loc(), ":-prefixed label is reserved for internal use")
	}

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

func (block *Block) AddToPhis(parent *Block, def *VariableDefinition) {
	if block.Phis == nil {
		block.Phis = map[string]*Phi{}
	}

	phi, ok := block.Phis[def.Name]
	if !ok {
		pos := parseutil.NewStartEndPos(block.Loc(), block.Loc())
		phi = &Phi{
			StartEndPos: pos,
			Dest: &VariableDefinition{
				StartEndPos: pos,
				Name:        def.Name,
			},
			Srcs: map[*Block]Value{},
		}
		phi.parentBlock = block
		phi.Dest.ParentInstruction = phi
		block.Phis[def.Name] = phi
	}

	phi.Add(parent, def)
}

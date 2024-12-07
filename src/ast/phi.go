package ast

import (
	"github.com/pattyshack/gt/parseutil"
)

// For internal use only
type Phi struct {
	instruction

	parseutil.StartEndPos

	Dest *VariableDefinition

	// Value is usually a local variable reference, but could be constant after
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

func (phi *Phi) Destination() *VariableDefinition {
	return phi.Dest
}

func (phi *Phi) Add(parent *Block, def *VariableDefinition) {
	ref := def.NewRef(phi.StartEnd())
	ref.SetParentInstruction(phi)
	phi.Srcs[parent] = ref
}

func (phi *Phi) Discard() {
	delete(phi.ParentBlock.Phis, phi.Dest.Name)
	for _, src := range phi.Srcs {
		src.Discard()
	}
}

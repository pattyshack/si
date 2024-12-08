package analyzer

import (
	"github.com/pattyshack/chickadee/ast"
)

// Note:
// 1. Liveness can be efficiently computed via ssa / loop tree, however,
// for simplicity, we'll stick with classic back flow propagation.
//
// 2. PHI:
//  a. Liveness.  Let
//        Xi = PHI(Xj, ...)
//    where the subscript indicate which block variable X is is defined in.
//    We'll use the convention that Xi is live in block i, and Xj is live out
//    of block j.
//  b. Deconstruction. Variable copying occurs on the edge from block i to
//    block j (when dealing real registers, this corresponds to inserting a
//    block between block i and block j, and handling data transfer in the
//    inserted block).

type liveSet map[*ast.VariableDefinition]int

func (set liveSet) update(def *ast.VariableDefinition, dist int) bool {
	origDist, ok := set[def]
	if !ok || origDist > dist {
		set[def] = dist
		return true
	}
	return false
}

func (set liveSet) equals(other liveSet) bool {
	if other == nil {
		return false
	}
	if len(set) != len(other) {
		return false
	}
	for def, dist := range set {
		otherDist, ok := other[def]
		if !ok || dist != otherDist {
			return false
		}
	}
	return true
}

// A block's live in and out sets
type liveInOut struct {
	// definition -> distance to next use (in number of instructions; phi
	// instructions counts as zero, the first real instruction counts as one)
	//
	// TODO: The distance heuristic does not take into account of loops /
	// branch probability. Improve after we have a working compiler.
	// If pgo statistics is available, maybe use markov chain weighted distance.

	// updated by current block
	in liveSet

	// updated by children block
	out liveSet
}

type livenessAnalyzer struct {
	live       map[*ast.Block]*liveInOut
	funcParams map[*ast.VariableDefinition]struct{}
}

var _ Pass[ast.SourceEntry] = &livenessAnalyzer{}

func NewLivenessAnalyzer() *livenessAnalyzer {
	return &livenessAnalyzer{
		live:       map[*ast.Block]*liveInOut{},
		funcParams: map[*ast.VariableDefinition]struct{}{},
	}
}

func (analyzer *livenessAnalyzer) Process(entry ast.SourceEntry) {
	funcDef, ok := entry.(*ast.FunctionDefinition)
	if !ok {
		return
	}

	for _, param := range funcDef.Parameters {
		analyzer.funcParams[param] = struct{}{}
	}

	for _, param := range funcDef.PseudoParameters {
		analyzer.funcParams[param] = struct{}{}
	}

	workSet := map[*ast.Block]struct{}{}
	for _, block := range funcDef.Blocks {
		analyzer.live[block] = &liveInOut{
			out: map[*ast.VariableDefinition]int{},
		}

		if len(block.Children) == 0 { // Terminal block
			workSet[block] = struct{}{}
		}
	}

	for len(workSet) > 0 {
		var block *ast.Block
		for b, _ := range workSet {
			block = b
			break
		}
		delete(workSet, block)

		if analyzer.updateBlockLiveIn(block) {
			for _, parent := range block.Parents {
				if analyzer.updateParentBlockLiveOut(parent, block) {
					workSet[parent] = struct{}{}
				}
			}
		}
	}
}

func (analyzer *livenessAnalyzer) isDefinedIn(
	def *ast.VariableDefinition,
	block *ast.Block,
) bool {
	_, ok := analyzer.funcParams[def]
	if ok {
		return false
	}

	return def.ParentInstruction.ParentBlock() == block
}

// Note: no need to perform per instruction back propagation since we have
// ssa use/def and parent block information.
func (analyzer *livenessAnalyzer) updateBlockLiveIn(block *ast.Block) bool {
	liveIn := liveSet{}

	for _, phi := range block.Phis {
		liveIn.update(phi.Dest, 0) // See note 2a.
	}

	for idx, inst := range block.Instructions {
		for _, src := range inst.Sources() {
			ref, ok := src.(*ast.VariableReference)
			if !ok { // immediate or global label reference
				continue
			}

			def := ref.UseDef
			if analyzer.isDefinedIn(def, block) {
				continue
			}

			liveIn.update(def, idx+1)
		}
	}

	for def, dist := range analyzer.live[block].out {
		if analyzer.isDefinedIn(def, block) {
			continue
		}

		liveIn.update(def, dist)
	}

	if !liveIn.equals(analyzer.live[block].in) {
		analyzer.live[block].in = liveIn
		return true
	}
	return false
}

func (analyzer *livenessAnalyzer) updateParentBlockLiveOut(
	parent *ast.Block,
	child *ast.Block,
) bool {
	childLiveIn := analyzer.live[child].in
	parentLiveOut := analyzer.live[parent].out

	modified := false
	parentBlockLength := len(parent.Instructions)
	localDefs := map[ast.Instruction]int{}
	for def, childDist := range childLiveIn {
		if analyzer.isDefinedIn(def, child) { // i.e., a phi, See note 2a
			if childDist != 0 {
				panic("should never happen")
			}

			phi := def.ParentInstruction.(*ast.Phi)
			ref, ok := phi.Srcs[parent].(*ast.VariableReference)
			if !ok { // immediate or global label reference
				continue
			}

			def = ref.UseDef // Corresponding parent block definition. See note 2a.
		}

		if analyzer.isDefinedIn(def, parent) {
			_, ok := def.ParentInstruction.(*ast.Phi)
			if !ok { // The definition belongs to a real instruction in the block
				localDefs[def.ParentInstruction] = childDist
				continue
			}
		}

		if parentLiveOut.update(def, parentBlockLength+childDist) {
			modified = true
		}
	}

	for idx, inst := range parent.Instructions {
		childDist, ok := localDefs[inst]
		if !ok {
			continue
		}

		dest := inst.Destination()
		if dest == nil {
			panic("should never happen")
		}

		if parentLiveOut.update(dest, parentBlockLength-idx+childDist) {
			modified = true
		}
	}

	return modified
}

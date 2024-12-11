package analyzer

import (
	"reflect"

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

type liveInfo struct {
	// Distance to next use in number of instructions relative to current
	// location. Current block's phi instructions counts as zero; the first real
	// instruction counts as one.
	//
	// TODO: The distance heuristic does not take into account of loops /
	// branch probability. Improve after we have a working compiler.
	// If pgo statistics is available, maybe use markov chain weighted distance.
	distance int

	// There could be multiple next use instructions if they are all in children
	// branches.
	nextUse map[ast.Instruction]struct{}
}

func newLiveInfo(dist int, inst ast.Instruction) *liveInfo {
	return &liveInfo{
		distance: dist,
		nextUse: map[ast.Instruction]struct{}{
			inst: struct{}{},
		},
	}
}

func (info *liveInfo) Copy() *liveInfo {
	nextUse := map[ast.Instruction]struct{}{}
	for inst, _ := range info.nextUse {
		nextUse[inst] = struct{}{}
	}
	return &liveInfo{
		distance: info.distance,
		nextUse:  nextUse,
	}
}

func (info *liveInfo) MergeFromChild(defDist int, childInfo *liveInfo) bool {
	totalDist := defDist + childInfo.distance

	modified := false
	if info.distance > totalDist {
		modified = true
		info.distance = totalDist
	}

	for inst, _ := range childInfo.nextUse {
		_, ok := info.nextUse[inst]
		if ok {
			continue
		}
		modified = true
		info.nextUse[inst] = struct{}{}
	}

	return modified
}

type liveSet map[*ast.VariableDefinition]*liveInfo

func (liveIn liveSet) MaybeAdd(
	def *ast.VariableDefinition,
	dist int,
	inst ast.Instruction,
) {
	info, ok := liveIn[def]
	if !ok {
		liveIn[def] = newLiveInfo(dist, inst)
	} else if info.distance > dist {
		panic("should never happen")
	}
}

func (liveIn liveSet) MaybeAddInfo(
	def *ast.VariableDefinition,
	other *liveInfo,
) {
	info, ok := liveIn[def]
	if !ok {
		liveIn[def] = other.Copy()
	} else if info.distance > other.distance {
		panic("should never happen")
	}
}

func (liveOut liveSet) MergeFromChild(
	def *ast.VariableDefinition,
	defDist int,
	childInfo *liveInfo,
) bool {
	info, ok := liveOut[def]
	if !ok {
		info = childInfo.Copy()
		info.distance += defDist
		liveOut[def] = info
		return true
	}
	return info.MergeFromChild(defDist, childInfo)
}

type livenessAnalyzer struct {
	// updated by children block
	liveOut map[*ast.Block]liveSet

	// updated by current block
	liveIn map[*ast.Block]liveSet

	funcParams map[*ast.VariableDefinition]struct{}
}

var _ Pass[ast.SourceEntry] = &livenessAnalyzer{}

func NewLivenessAnalyzer() *livenessAnalyzer {
	return &livenessAnalyzer{
		liveOut:    map[*ast.Block]liveSet{},
		liveIn:     map[*ast.Block]liveSet{},
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

	workSet := newDataflowWorkSet()
	for _, block := range funcDef.Blocks {
		analyzer.liveOut[block] = liveSet{}
		if len(block.Children) == 0 { // Terminal block
			workSet.push(block)
		}
	}

	for !workSet.isEmpty() {
		block := workSet.pop()
		if analyzer.updateLiveIn(block) {
			for _, parent := range block.Parents {
				if analyzer.updateParentLiveOut(parent, block) {
					workSet.push(parent)
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
func (analyzer *livenessAnalyzer) updateLiveIn(block *ast.Block) bool {
	liveIn := liveSet{}

	for _, phi := range block.Phis {
		liveIn.MaybeAdd(phi.Dest, 0, phi) // See note 2a.
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

			liveIn.MaybeAdd(def, idx+1, inst)
		}
	}

	for def, info := range analyzer.liveOut[block] {
		if analyzer.isDefinedIn(def, block) {
			continue
		}
		liveIn.MaybeAddInfo(def, info)
	}

	if !reflect.DeepEqual(liveIn, analyzer.liveIn[block]) {
		analyzer.liveIn[block] = liveIn
		return true
	}
	return false
}

func (analyzer *livenessAnalyzer) updateParentLiveOut(
	parent *ast.Block,
	child *ast.Block,
) bool {
	childLiveIn := analyzer.liveIn[child]
	parentLiveOut := analyzer.liveOut[parent]

	modified := false
	parentBlockLength := len(parent.Instructions)
	localDefs := map[ast.Instruction]*liveInfo{}
	for def, childInfo := range childLiveIn {
		if analyzer.isDefinedIn(def, child) { // i.e., a phi, See note 2a
			if childInfo.distance != 0 {
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
				localDefs[def.ParentInstruction] = childInfo
				continue
			}
		}

		if parentLiveOut.MergeFromChild(def, parentBlockLength, childInfo) {
			modified = true
		}
	}

	for idx, inst := range parent.Instructions {
		childInfo, ok := localDefs[inst]
		if !ok {
			continue
		}

		dest := inst.Destination()
		if dest == nil {
			panic("should never happen")
		}

		if parentLiveOut.MergeFromChild(dest, parentBlockLength-idx, childInfo) {
			modified = true
		}
	}

	return modified
}

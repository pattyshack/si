package allocator

import (
	"reflect"

	"github.com/pattyshack/chickadee/analyzer/util"
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

// All distances are in number of instructions relative to the beginning of
// the current block. Current block's phi instructions counts as zero; the
// first real instruction counts as one.
//
// TODO: The distance heuristic does not take into account of loops /
// branch probability. Improve after we have a working compiler.
// If pgo statistics is available, maybe use markov chain weighted distance.
type LiveInfo struct {
	// There could be multiple next use instructions if they are all in children
	// branches.
	NextUse map[ast.Instruction]int
}

func (info *LiveInfo) Copy() *LiveInfo {
	nextUse := map[ast.Instruction]int{}
	for inst, dist := range info.NextUse {
		nextUse[inst] = dist
	}
	return &LiveInfo{
		NextUse: nextUse,
	}
}

func (liveOutInfo *LiveInfo) MergeFromChildLiveInInfo(
	parentBlockLength int,
	childLiveInInfo *LiveInfo,
) bool {
	modified := false
	for inst, childNextUseDist := range childLiveInInfo.NextUse {
		dist := parentBlockLength + childNextUseDist
		origDist, ok := liveOutInfo.NextUse[inst]
		if ok && origDist < dist {
			continue
		}
		modified = true
		liveOutInfo.NextUse[inst] = dist
	}

	return modified
}

type LiveSet map[*ast.VariableDefinition]*LiveInfo

func (liveIn LiveSet) InstructionUses(
	inst ast.Instruction,
	useDist int,
	def *ast.VariableDefinition,
) {
	_, ok := liveIn[def]
	if !ok {
		liveIn[def] = &LiveInfo{
			NextUse: map[ast.Instruction]int{
				inst: useDist,
			},
		}
	}
}

func (liveIn LiveSet) MergeFromLiveOut(
	def *ast.VariableDefinition,
	liveOutInfo *LiveInfo,
) {
	_, ok := liveIn[def]
	if !ok {
		liveIn[def] = liveOutInfo.Copy()
	}
}

func (liveOut LiveSet) MergeFromChildLiveIn(
	def *ast.VariableDefinition,
	parentBlockLength int,
	childLiveInInfo *LiveInfo,
) bool {
	info, ok := liveOut[def]
	if !ok {
		info = &LiveInfo{
			NextUse: map[ast.Instruction]int{},
		}
		liveOut[def] = info
	}
	modified := info.MergeFromChildLiveInInfo(parentBlockLength, childLiveInInfo)
	return !ok || modified
}

type LivenessAnalyzer struct {
	// updated by children block
	LiveOut map[*ast.Block]LiveSet

	// updated by current block
	LiveIn map[*ast.Block]LiveSet

	funcParams map[*ast.VariableDefinition]struct{}
}

var _ util.Pass[ast.SourceEntry] = &LivenessAnalyzer{}

func NewLivenessAnalyzer() *LivenessAnalyzer {
	return &LivenessAnalyzer{
		LiveOut:    map[*ast.Block]LiveSet{},
		LiveIn:     map[*ast.Block]LiveSet{},
		funcParams: map[*ast.VariableDefinition]struct{}{},
	}
}

func (analyzer *LivenessAnalyzer) Process(entry ast.SourceEntry) {
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

	workSet := util.NewDataflowWorkSet()
	for _, block := range funcDef.Blocks {
		analyzer.LiveOut[block] = LiveSet{}
		if len(block.Children) == 0 { // Terminal block
			workSet.Push(block)
		}
	}

	for !workSet.IsEmpty() {
		block := workSet.Pop()
		if analyzer.updateLiveIn(block) {
			for _, parent := range block.Parents {
				if analyzer.updateParentLiveOut(parent, block) {
					workSet.Push(parent)
				}
			}
		}
	}
}

func (analyzer *LivenessAnalyzer) isDefinedIn(
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
func (analyzer *LivenessAnalyzer) updateLiveIn(block *ast.Block) bool {
	liveIn := LiveSet{}

	for _, phi := range block.Phis {
		liveIn.InstructionUses(phi, 0, phi.Dest) // See note 2a.
	}

	for idx, inst := range block.Instructions {
		dist := idx + 1
		for _, src := range inst.Sources() {
			ref, ok := src.(*ast.VariableReference)
			if !ok { // immediate or global label reference
				continue
			}

			def := ref.UseDef
			if analyzer.isDefinedIn(def, block) {
				continue
			}

			liveIn.InstructionUses(inst, dist, def)
		}
	}

	for def, info := range analyzer.LiveOut[block] {
		if analyzer.isDefinedIn(def, block) {
			continue
		}
		liveIn.MergeFromLiveOut(def, info)
	}

	if !reflect.DeepEqual(liveIn, analyzer.LiveIn[block]) {
		analyzer.LiveIn[block] = liveIn
		return true
	}
	return false
}

func (analyzer *LivenessAnalyzer) updateParentLiveOut(
	parent *ast.Block,
	child *ast.Block,
) bool {
	childLiveIn := analyzer.LiveIn[child]
	parentLiveOut := analyzer.LiveOut[parent]

	modified := false
	parentBlockLength := len(parent.Instructions) + 1 // +1 for parent's phis
	localDefs := map[ast.Instruction]*LiveInfo{}
	for def, childInfo := range childLiveIn {
		if analyzer.isDefinedIn(def, child) { // i.e., a phi, See note 2a
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

		if parentLiveOut.MergeFromChildLiveIn(def, parentBlockLength, childInfo) {
			modified = true
		}
	}

	for _, inst := range parent.Instructions {
		childInfo, ok := localDefs[inst]
		if !ok {
			continue
		}

		dest := inst.Destination()
		if dest == nil {
			panic("should never happen")
		}

		if parentLiveOut.MergeFromChildLiveIn(dest, parentBlockLength, childInfo) {
			modified = true
		}
	}

	return modified
}

package analyzer

import (
	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
)

type ssaConstructor struct {
	*parseutil.Emitter

	liveOuts map[*ast.Block]map[string]*ast.RegisterDefinition
}

func ConstructSSA(emitter *parseutil.Emitter) Pass[ast.SourceEntry] {
	return &ssaConstructor{
		Emitter:  emitter,
		liveOuts: map[*ast.Block]map[string]*ast.RegisterDefinition{},
	}
}

func (constructor *ssaConstructor) Process(entry ast.SourceEntry) {
	funcDef, ok := entry.(*ast.FuncDefinition)
	if !ok {
		return
	}

	initLiveIn := map[string]*ast.RegisterDefinition{}
	for _, param := range funcDef.Parameters {
		initLiveIn[param.Name] = param
	}
	constructor.liveOuts[funcDef.Blocks[0]] = initLiveIn

	processed := map[*ast.Block]struct{}{}
	queue := []*ast.Block{funcDef.Blocks[0]}
	for len(queue) > 0 {
		block := queue[0]
		queue = queue[1:]

		_, ok := processed[block]
		if ok {
			continue
		}

		liveOut := constructor.processBlock(block)
		constructor.populateChildrenPhis(block, liveOut)

		for _, child := range block.Children {
			queue = append(queue, child)
		}
		processed[block] = struct{}{}
	}

	// In theory, this should never happen since the graph is reducible, but
	// it doesn't hurt to double check.
	for _, block := range funcDef.Blocks {
		if len(block.Parents) < 2 {
			continue
		}

		for _, phi := range block.Phis {
			if len(phi.Dest.DefUses) > 0 && len(phi.Srcs) != len(block.Parents) {
				for ref, _ := range phi.Dest.DefUses {
					constructor.Emit(
						ref.Loc(),
						"variable (%s) not defined in all parent blocks",
						ref.Name)
				}
			}
		}
	}

	// TODO prune to minimal ssa form
}

func (constructor *ssaConstructor) processBlock(
	block *ast.Block,
) map[string]*ast.RegisterDefinition {
	liveOut, ok := constructor.liveOuts[block]
	if !ok {
		liveOut = map[string]*ast.RegisterDefinition{}
		for name, phi := range block.Phis {
			liveOut[name] = phi.Dest
		}
	}

	for _, inst := range block.Instructions {
		inst.SetParentBlock(block)
		for _, src := range inst.Sources() {
			src.SetParent(inst)

			ref, ok := src.(*ast.RegisterReference)
			if !ok {
				continue
			}

			def, ok := liveOut[ref.Name]
			if !ok {
				constructor.Emit(
					ref.Loc(),
					"variable (%s) not defined in all parent blocks",
					ref.Name)
				continue
			}

			def.AddRef(ref)
		}

		dest := inst.Destination()
		if dest == nil {
			continue
		}

		dest.Parent = inst
		liveOut[dest.Name] = dest
	}

	return liveOut
}

func (constructor *ssaConstructor) populateChildrenPhis(
	block *ast.Block,
	liveOut map[string]*ast.RegisterDefinition,
) {
	for _, child := range block.Children {
		if len(child.Parents) == 1 {
			// Skip populating phis since there won't be any left after pruning
			liveIn := make(map[string]*ast.RegisterDefinition, len(liveOut))
			for name, def := range liveOut {
				liveIn[name] = def
			}
			constructor.liveOuts[child] = liveIn
			continue
		}

		for _, def := range liveOut {
			child.AddToPhis(block, def)
		}
	}
}

package analyzer

import (
	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
)

type ssaConstructor struct {
	*parseutil.Emitter
}

func ConstructSSA(emitter *parseutil.Emitter) Pass[ast.SourceEntry] {
	return &ssaConstructor{
		Emitter: emitter,
	}
}

func (constructor *ssaConstructor) Process(entry ast.SourceEntry) {
	funcDef, ok := entry.(*ast.FuncDefinition)
	if !ok {
		return
	}

	defs := map[string]*ast.RegisterDefinition{}
	for _, param := range funcDef.Parameters {
		defs[param.Name] = param
	}
	funcDef.Blocks[0].LiveIn = defs

	processed := map[*ast.Block]struct{}{}
	queue := []*ast.Block{funcDef.Blocks[0]}
	for len(queue) > 0 {
		block := queue[0]
		queue = queue[1:]

		_, ok := processed[block]
		if ok {
			continue
		}

		constructor.processBlock(block)
		constructor.updateChildrenPhis(block)

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

func (constructor *ssaConstructor) processBlock(block *ast.Block) {
	if block.LiveIn == nil { // i.e., have multiple parents
		liveIn := map[string]*ast.RegisterDefinition{}
		for name, phi := range block.Phis {
			liveIn[name] = phi.Dest
		}
		block.LiveIn = liveIn
	}

	liveOut := map[string]*ast.RegisterDefinition{}
	for name, def := range block.LiveIn {
		liveOut[name] = def
	}

	for _, inst := range block.Instructions {
		inst.SetParentBlock(block)
		for _, src := range inst.Sources() {
			ref, ok := src.(*ast.RegisterReference)
			if !ok {
				continue
			}

			ref.Parent = inst
			def, ok := liveOut[ref.Name]
			if !ok {
				constructor.Emit(
					ref.Loc(),
					"variable (%s) not defined in all parent blocks",
					ref.Name)
				continue
			}

			if def.DefUses == nil {
				def.DefUses = map[*ast.RegisterReference]struct{}{}
			}

			def.DefUses[ref] = struct{}{}
			ref.UseDef = def
		}

		dest := inst.Destination()
		if dest == nil {
			continue
		}

		dest.Parent = inst
		liveOut[dest.Name] = dest
	}

	if len(block.Children) != 0 { // nothing is live after terminal
		block.LiveOut = liveOut
	}
}

func (constructor *ssaConstructor) updateChildrenPhis(block *ast.Block) {
	for _, child := range block.Children {
		if len(child.Parents) == 1 {
			// Skip populating phis since there won't be any left after pruning
			child.LiveIn = block.LiveOut
			continue
		}

		if child.Phis == nil {
			child.Phis = map[string]*ast.Phi{}
		}

		for _, def := range block.LiveOut {
			pos := parseutil.NewStartEndPos(child.Loc(), child.Loc())
			phi, ok := child.Phis[def.Name]
			if !ok {
				phi = &ast.Phi{
					Dest: &ast.RegisterDefinition{
						StartEndPos: pos,
						Name:        def.Name,
					},
					Srcs: map[*ast.Block]ast.Value{},
				}
				child.Phis[def.Name] = phi
			}

			ref := &ast.RegisterReference{
				StartEndPos: pos,
				Name:        def.Name,
				UseDef:      def,
			}

			if def.DefUses == nil {
				def.DefUses = map[*ast.RegisterReference]struct{}{}
			}

			def.DefUses[ref] = struct{}{}
			phi.Srcs[block] = ref
		}
	}
}

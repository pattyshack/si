package analyzer

import (
	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
)

type ssaConstructor struct {
	*parseutil.Emitter

	// reachable definitions
	defOuts map[*ast.Block]map[string]*ast.VariableDefinition
}

func ConstructSSA(emitter *parseutil.Emitter) Pass[ast.SourceEntry] {
	return &ssaConstructor{
		Emitter: emitter,
		defOuts: map[*ast.Block]map[string]*ast.VariableDefinition{},
	}
}

func (constructor *ssaConstructor) Process(entry ast.SourceEntry) {
	funcDef, ok := entry.(*ast.FunctionDefinition)
	if !ok {
		return
	}

	initDefIn := map[string]*ast.VariableDefinition{}
	for _, param := range funcDef.Parameters {
		initDefIn[param.Name] = param
	}
	constructor.defOuts[funcDef.Blocks[0]] = initDefIn

	processed := map[*ast.Block]struct{}{}
	queue := []*ast.Block{funcDef.Blocks[0]}
	for len(queue) > 0 {
		block := queue[0]
		queue = queue[1:]

		_, ok := processed[block]
		if ok {
			continue
		}
		processed[block] = struct{}{}

		defOut := constructor.processBlock(block)
		constructor.populateChildrenPhis(block, defOut)

		queue = append(queue, block.Children...)
	}

	// In theory, this should never happen since the graph is reducible, but
	// it doesn't hurt to double check.
	for _, block := range funcDef.Blocks {
		for _, phi := range block.Phis {
			if len(phi.Dest.DefUses) > 0 && len(phi.Srcs) != len(block.Parents) {
				for ref, _ := range phi.Dest.DefUses {
					constructor.Emit(
						ref.Loc(),
						"register (%s) not defined in all parent blocks",
						ref.Name)
				}
			}
		}
	}

	constructor.prunePhis(funcDef.Blocks)
}

func (constructor *ssaConstructor) processBlock(
	block *ast.Block,
) map[string]*ast.VariableDefinition {
	defOut, ok := constructor.defOuts[block]
	if !ok {
		defOut = map[string]*ast.VariableDefinition{}
		for name, phi := range block.Phis {
			defOut[name] = phi.Dest
		}
	}

	for _, inst := range block.Instructions {
		for _, src := range inst.Sources() {
			ref, ok := src.(*ast.VariableReference)
			if !ok {
				continue
			}

			def, ok := defOut[ref.Name]
			if !ok {
				constructor.Emit(
					ref.Loc(),
					"register (%s) not defined in all parent blocks",
					ref.Name)
				continue
			}

			def.AddRef(ref)
		}

		dest := inst.Destination()
		if dest == nil {
			continue
		}

		dest.ParentInstruction = inst
		defOut[dest.Name] = dest
	}

	return defOut
}

func (constructor *ssaConstructor) populateChildrenPhis(
	block *ast.Block,
	defOut map[string]*ast.VariableDefinition,
) {
	for _, child := range block.Children {
		if len(child.Parents) == 1 {
			// Skip populating phis since there won't be any left after pruning
			defIn := make(map[string]*ast.VariableDefinition, len(defOut))
			for name, def := range defOut {
				defIn[name] = def
			}
			constructor.defOuts[child] = defIn
			continue
		}

		for _, def := range defOut {
			child.AddToPhis(block, def)
		}
	}
}

func (constructor *ssaConstructor) prunePhis(blocks []*ast.Block) {
	toCheck := map[*ast.Phi]struct{}{}
	for _, block := range blocks {
		for _, phi := range block.Phis {
			toCheck[phi] = struct{}{}
		}
	}

	for len(toCheck) > 0 {
		// "pop" the first toCheck item

		var phi *ast.Phi
		for p, _ := range toCheck {
			phi = p
			break
		}
		delete(toCheck, phi)

		// For xi = phi(x1, ... xn), if all x1 ..., xn are either xi or xj, replace
		// xi with xj and delete phi

		var value ast.Value
		definitions := map[interface{}]struct{}{}
		for _, src := range phi.Srcs {
			def := src.Definition()

			regDef, ok := def.(*ast.VariableDefinition)
			if ok && regDef == phi.Dest { // ignore self reference
				continue
			}

			definitions[def] = struct{}{}
			value = src
		}

		if len(definitions) == 0 {
			panic("should never happen")
		}

		if len(definitions) != 1 {
			continue
		}

		for ref, _ := range phi.Dest.DefUses {
			otherPhi, ok := ref.ParentInstruction.(*ast.Phi)
			if ok && otherPhi != phi {
				// Removing the current phi may enable us to remove the otherPhi
				toCheck[otherPhi] = struct{}{}
			}
		}
		phi.Dest.ReplaceReferencesWith(value)
		phi.Discard()
	}
}

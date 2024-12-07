package analyzer

import (
	"github.com/pattyshack/chickadee/ast"
)

type retPseudoSourcesPopulator struct {
	constraints *callRetConstraints
}

func PopulateRetPseudoSources(
	constraints *callRetConstraints,
) Pass[ast.SourceEntry] {
	return &retPseudoSourcesPopulator{
		constraints: constraints,
	}
}

func (populator retPseudoSourcesPopulator) Process(
	entry ast.SourceEntry,
) {
	funcDef, ok := entry.(*ast.FunctionDefinition)
	if !ok {
		return
	}

	populator.constraints.Ready()

	for _, block := range funcDef.Blocks {
		if len(block.Children) > 0 {
			continue
		}

		term := block.Instructions[len(block.Instructions)-1].(*ast.Terminal)
		if term.Kind != ast.Ret {
			continue
		}

		for _, def := range funcDef.PseudoParameters {
			ref := def.NewRef(term.StartEnd())
			ref.SetParentInstruction(term)
			term.PseudoSources = append(term.PseudoSources, ref)
		}
	}
}

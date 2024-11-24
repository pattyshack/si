package analyzer

import (
	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
)

type typeChecker struct {
	*parseutil.Emitter
	globalSignatures map[string]ast.SourceEntry
}

func CheckTypes(
	emitter *parseutil.Emitter,
	globalSignatures map[string]ast.SourceEntry,
) Pass[ast.SourceEntry] {
	return &typeChecker{
		Emitter:          emitter,
		globalSignatures: globalSignatures,
	}
}

func (checker *typeChecker) Process(entry ast.SourceEntry) {
	def, ok := entry.(*ast.FuncDefinition)
	if !ok {
		return
	}

	for _, block := range def.Blocks {
		for _, inst := range block.Instructions {
			switch inst.(type) {
			// TODO implement
			}
		}
	}
}

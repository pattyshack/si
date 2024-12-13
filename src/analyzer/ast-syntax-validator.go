package analyzer

import (
	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
)

type astSyntaxValidator struct {
	*parseutil.Emitter
}

func ValidateAstSyntax(emitter *parseutil.Emitter) Pass[ast.SourceEntry] {
	return &astSyntaxValidator{
		Emitter: emitter,
	}
}

func (validator *astSyntaxValidator) Process(entry ast.SourceEntry) {
	entry.Walk(validator)
}

func (validator *astSyntaxValidator) Enter(n ast.Node) {
	switch node := n.(type) {
	case ast.SourceEntry:
		if validator.Emitter.HasErrors() {
			panic("should never happen since source entries are roots")
		}
		node.Validate(validator.Emitter)
		if validator.Emitter.HasErrors() {
			node.SetHasDeclarationSyntaxError(true)
		}
	case ast.Validator:
		node.Validate(validator.Emitter)
	}
}

func (validator *astSyntaxValidator) Exit(node ast.Node) {
}

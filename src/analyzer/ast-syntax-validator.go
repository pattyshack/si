package analyzer

import (
	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
)

type astSyntaxValidator struct {
	*parseutil.Emitter
}

func ValidateAstSyntax(entry ast.SourceEntry, emitter *parseutil.Emitter) {
	validator := astSyntaxValidator{
		Emitter: emitter,
	}
	entry.Walk(validator)
}

func (validator astSyntaxValidator) Enter(n ast.Node) {
	switch node := n.(type) {
	case ast.Validator:
		node.Validate(validator.Emitter)
	}
}

func (validator astSyntaxValidator) Exit(node ast.Node) {
}

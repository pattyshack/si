package analyzer

import (
	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
)

type astSyntaxValidator struct {
	*parseutil.Emitter
}

func ValidateAstSyntax(emitter *parseutil.Emitter) Pass[[]ast.SourceEntry] {
	return &astSyntaxValidator{
		Emitter: emitter,
	}
}

func (validator *astSyntaxValidator) Process(list []ast.SourceEntry) {
	ParallelWalk(list, func(ast.SourceEntry) ast.Visitor { return validator })
}

func (validator *astSyntaxValidator) Enter(node ast.Node) {
	validatable, ok := node.(ast.Validator)
	if ok {
		validatable.Validate(validator.Emitter)
	}
}

func (validator *astSyntaxValidator) Exit(node ast.Node) {
}

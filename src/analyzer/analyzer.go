package analyzer

import (
	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
)

func Analyze(source []ast.SourceEntry, emitter *parseutil.Emitter) {
	passes := [][]Pass[[]ast.SourceEntry]{
		{
			ValidateAstSyntax(emitter),
		},
		{
			InitializeControlFlowGraph(emitter),
		},
	}

	Process(source, passes, func() bool { return emitter.HasErrors() })
}

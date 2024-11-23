package analyzer

import (
	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
)

func Analyze(sources []ast.SourceEntry, emitter *parseutil.Emitter) {
	passes := [][]Pass[[]ast.SourceEntry]{
		{
			ValidateAstSyntax(emitter),
		},
	}

	Process(sources, passes, nil)
	if emitter.HasErrors() {
		return
	}

	// TODO collect definitions / declarations before parallel process

	ParallelProcess(
		sources,
		func(ast.SourceEntry) func(ast.SourceEntry) {
			passes := [][]Pass[ast.SourceEntry]{
				{InitializeControlFlowGraph(emitter)},
			}

			return func(entry ast.SourceEntry) {
				Process(entry, passes, nil)
			}
		})
}

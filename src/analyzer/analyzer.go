package analyzer

import (
	"context"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/platform"
)

func Analyze(
	sources []ast.SourceEntry,
	targetPlatform platform.Platform,
	emitter *parseutil.Emitter,
) {
	_, abort := context.WithCancel(context.Background())
	// TODO
	//abortCtx, abort := context.WithCancel(context.Background())

	entryEmitters := map[ast.SourceEntry]*parseutil.Emitter{}
	for _, entry := range sources {
		entryEmitters[entry] = &parseutil.Emitter{}
	}

	ParallelProcess(
		sources,
		func(entry ast.SourceEntry) {
			entryEmitter := entryEmitters[entry]
			ValidateAstSyntax(entry, entryEmitter)
			if entryEmitter.HasErrors() {
				abort()
			}
		})

	collector := NewSignatureCollector(emitter)
	collector.Process(sources)

	if emitter.HasErrors() {
		abort()
	}

	signatures := collector.Signatures()

	ParallelProcess(
		sources,
		func(entry ast.SourceEntry) {
			entryEmitter := entryEmitters[entry]
			if entryEmitter.HasErrors() { // Entry has syntax error
				return
			}

			passes := [][]Pass[ast.SourceEntry]{
				{InitializeControlFlowGraph(entryEmitter)},
				{BindGlobalLabelReferences(entryEmitter, signatures)},
				{ConstructSSA(entryEmitter)},
				{CheckTypes(entryEmitter, targetPlatform)},
			}

			Process(entry, passes, nil)
			if entryEmitter.HasErrors() {
				abort()
			}
		})

	for _, entryEmitter := range entryEmitters {
		emitter.EmitErrors(entryEmitter.Errors()...)
	}
}

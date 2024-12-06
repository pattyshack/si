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

	signatureCollector := NewSignatureCollector(emitter)
	signatureCollector.Process(sources)
	signatures := signatureCollector.Signatures()

	if emitter.HasErrors() {
		abort()
	}

	callRetConstraintsCollector := NewCallRetConstraintsCollector(
		targetPlatform,
		len(sources))

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
				{
					CheckTypes(entryEmitter, targetPlatform),
					callRetConstraintsCollector,
				},
			}

			Process(entry, passes, nil)
			if entryEmitter.HasErrors() {
				abort()
			}

			callRetConstraintsCollector.Constraints()
			// TODO use in register allocation
			// callRetConstraints := callRetConstraintsCollector.Constraints()
		})

	for _, entryEmitter := range entryEmitters {
		emitter.EmitErrors(entryEmitter.Errors()...)
	}
}

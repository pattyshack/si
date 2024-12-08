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
	abortBuildCtx, abortBuild := context.WithCancel(context.Background())
	shouldAbortBuild := func() bool {
		select {
		case <-abortBuildCtx.Done():
			return true
		default:
			return false
		}
	}

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
				abortBuild()
			}
		})

	signatures, callRetConstraints := CollectSignaturesAndCallRetConstraints(
		sources,
		targetPlatform,
		emitter)
	if emitter.HasErrors() {
		abortBuild()
	}

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
				abortBuild()
			}

			// At this point, the entry is well-form and no more error could occur.
			if shouldAbortBuild() {
				return
			}

			passes = [][]Pass[ast.SourceEntry]{
				{PopulateTerminalPseudoEntries(targetPlatform, callRetConstraints)},
				{PrintLiveness()},
			}

			Process(entry, passes, shouldAbortBuild)
		})

	for _, entryEmitter := range entryEmitters {
		emitter.EmitErrors(entryEmitter.Errors()...)
	}
}

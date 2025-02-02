package analyzer

import (
	"context"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/analyzer/allocator"
	"github.com/pattyshack/chickadee/analyzer/util"
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

	util.ParallelProcess(
		sources,
		func(entry ast.SourceEntry) {
			entryEmitter := entryEmitters[entry]

			syntaxPasses := [][]util.Pass[ast.SourceEntry]{
				{ValidateAstSyntax(entryEmitter)},
				// Note: func def type must be generated prior to signature collection
				// to avoid race races during name binding.
				{GenerateFuncDefTypeAndConstraints(entryEmitter, targetPlatform)},
			}

			if !util.Process(
				entry,
				syntaxPasses,
				func() bool { return entryEmitter.HasErrors() }) {

				abortBuild()
			}
		})

	signatures := CollectSignatures(sources, emitter)
	if emitter.HasErrors() {
		abortBuild()
	}

	util.ParallelProcess(
		sources,
		func(entry ast.SourceEntry) {
			entryEmitter := entryEmitters[entry]
			if entryEmitter.HasErrors() { // Entry has syntax error
				return
			}

			setupPasses := [][]util.Pass[ast.SourceEntry]{
				{InitializeControlFlowGraph(entryEmitter)},
				{ModifyTerminals(targetPlatform)},
			}

			if !util.Process(
				entry,
				setupPasses,
				func() bool { return entryEmitter.HasErrors() }) {

				abortBuild()
				return
			}

			semanticCheckPasses := [][]util.Pass[ast.SourceEntry]{
				{BindGlobalLabelReferences(entryEmitter, signatures)},
				{ConstructSSA(entryEmitter)},
				{CheckTypes(entryEmitter, targetPlatform)},
			}

			util.Process(entry, semanticCheckPasses, nil)
			if entryEmitter.HasErrors() {
				abortBuild()
				return
			}

			// At this point, the entry is well-form and no more error could occur.
			if shouldAbortBuild() {
				return
			}

			debugMode := true
			registerStackAllocator := allocator.NewAllocator(
				targetPlatform,
				debugMode)
			backendPasses := [][]util.Pass[ast.SourceEntry]{
				{registerStackAllocator},
			}
			if debugMode {
				backendPasses = append(
					backendPasses,
					[]util.Pass[ast.SourceEntry]{
						// these passes are only used for debugging the compiler
						// implementation and should be removed or flag guarded once the
						// compiler works.
						allocator.Debug(registerStackAllocator),
					})
			}

			util.Process(entry, backendPasses, shouldAbortBuild)
		})

	for _, entryEmitter := range entryEmitters {
		emitter.EmitErrors(entryEmitter.Errors()...)
	}
}

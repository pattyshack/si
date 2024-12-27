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

			passes := [][]util.Pass[ast.SourceEntry]{
				{ValidateAstSyntax(entryEmitter)},
				{GenerateFuncDefConstraints(entryEmitter, targetPlatform)},
				{InitializeControlFlowGraph(entryEmitter)},
			}

			util.Process(entry, passes, nil)
			if entryEmitter.HasErrors() {
				abortBuild()
			}
		})

	signatures := CollectSignaturesAndCallRetConstraints(sources, emitter)
	if emitter.HasErrors() {
		abortBuild()
	}

	util.ParallelProcess(
		sources,
		func(entry ast.SourceEntry) {
			entryEmitter := entryEmitters[entry]
			if entryEmitter.HasErrors() { // Entry has syntax / func def type error
				return
			}

			passes := [][]util.Pass[ast.SourceEntry]{
				{ModifyTerminals(targetPlatform)},
				{BindGlobalLabelReferences(entryEmitter, signatures)},
				{ConstructSSA(entryEmitter)},
				{CheckTypes(entryEmitter, targetPlatform)},
			}

			util.Process(entry, passes, nil)
			if entryEmitter.HasErrors() {
				abortBuild()
			}

			// At this point, the entry is well-form and no more error could occur.
			if shouldAbortBuild() {
				return
			}

			debugMode := true
			registerStackAllocator := allocator.NewAllocator(
				targetPlatform,
				debugMode)
			passes = [][]util.Pass[ast.SourceEntry]{
				{registerStackAllocator},
			}
			if debugMode {
				passes = append(
					passes,
					[]util.Pass[ast.SourceEntry]{
						// these passes are only used for debugging the compiler
						// implementation and should be removed or flag guarded once the
						// compiler works.
						allocator.Debug(registerStackAllocator),
						allocator.ValidateInstructionConstraints(
							targetPlatform,
							registerStackAllocator),
					})
			}

			util.Process(entry, passes, shouldAbortBuild)
		})

	for _, entryEmitter := range entryEmitters {
		emitter.EmitErrors(entryEmitter.Errors()...)
	}
}

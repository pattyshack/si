package analyzer

import (
	"strconv"
	"sync"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
)

type controlFlowGraphInitializer struct {
	*parseutil.Emitter
}

func InitializeControlFlowGraph(
	emitter *parseutil.Emitter,
) Pass[[]ast.SourceEntry] {
	return &controlFlowGraphInitializer{
		Emitter: emitter,
	}
}

func (initializer *controlFlowGraphInitializer) Process(
	source []ast.SourceEntry,
) {
	wg := sync.WaitGroup{}
	for _, entry := range source {
		funcDef, ok := entry.(*ast.FuncDefinition)
		if !ok {
			continue
		}

		wg.Add(1)
		func(def *ast.FuncDefinition) {
			initializer.process(def)
			wg.Done()
		}(funcDef)
	}
	wg.Wait()
}

func (initializer *controlFlowGraphInitializer) process(
	def *ast.FuncDefinition,
) {
	labelled := map[string]*ast.Block{}
	names := map[string]struct{}{}
	for _, block := range def.Blocks {
		if block.Label != "" {
			labelled[block.Label] = block
			names[block.Label] = struct{}{}
		}
	}

	for idx, block := range def.Blocks {
		canFallthrough := true
		last := block.Instructions[len(block.Instructions)-1]
		switch jump := last.(type) {
		case *ast.Jump:
			canFallthrough = false

			child, ok := labelled[jump.Label]
			if !ok {
				initializer.Emit(jump.Loc(), "undefined block label (%s)", jump.Label)
				names[jump.Label] = struct{}{}
			} else {
				block.Children = append(block.Children, child)
				child.Parents = append(child.Parents, block)
			}
		case *ast.ConditionalJump:
			child, ok := labelled[jump.Label]
			if !ok {
				initializer.Emit(jump.Loc(), "undefined block label (%s)", jump.Label)
				names[jump.Label] = struct{}{}
			} else {
				block.Children = append(block.Children, child)
				child.Parents = append(child.Parents, block)
			}
		case *ast.Terminate:
			canFallthrough = false
		}

		if !canFallthrough {
			continue
		}

		if idx == len(def.Blocks)-1 {
			initializer.Emit(
				last.Loc(),
				"last statement in function must either exit the function or "+
					"unconditionally jump to another block")
			continue
		}

		child := def.Blocks[idx+1]

		block.Children = append(block.Children, child)
		child.Parents = append(child.Parents, block)
	}

	// Add labels for debugging purpose
	idx := 0
	for _, block := range def.Blocks {
		if block.Label != "" {
			continue
		}

		for {
			label := ":" + strconv.Itoa(idx)
			idx++

			_, ok := names[label]
			if !ok {
				block.Label = label
				break
			}
		}
	}
}

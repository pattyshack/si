package analyzer

import (
	"fmt"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/analyzer/util"
	"github.com/pattyshack/chickadee/ast"
)

type controlFlowGraphInitializer struct {
	*parseutil.Emitter
}

func InitializeControlFlowGraph(
	emitter *parseutil.Emitter,
) util.Pass[ast.SourceEntry] {
	return &controlFlowGraphInitializer{
		Emitter: emitter,
	}
}

func (initializer *controlFlowGraphInitializer) Process(
	entry ast.SourceEntry,
) {
	def, ok := entry.(*ast.FunctionDefinition)
	if !ok {
		return
	}

	nextId := 0
	labelled := map[string]*ast.Block{}
	names := map[string]struct{}{}
	for _, block := range def.Blocks {
		if block.Label != "" {
			labelled[block.Label] = block
			names[block.Label] = struct{}{}
		} else {
			// Add labels for internal debugging purpose
			block.Label = fmt.Sprintf(":unlabelled-block-%d", nextId)
			nextId++
		}
	}

	for idx, block := range def.Blocks {
		if idx == 0 {
			// The entry block was inserted during call convention generation and
			// may be empty if there are no callee-saved registers.
			firstRealBlock := def.Blocks[1]
			block.Children = []*ast.Block{firstRealBlock}
			firstRealBlock.Parents = []*ast.Block{block}
			continue
		}

		var prevChild *ast.Block
		last := block.Instructions[len(block.Instructions)-1]
		switch jump := last.(type) {
		case *ast.Jump:
			child, ok := labelled[jump.Label]
			if !ok {
				initializer.Emit(jump.Loc(), "undefined block label (%s)", jump.Label)
				names[jump.Label] = struct{}{}
			} else {
				block.Children = append(block.Children, child)
				child.Parents = append(child.Parents, block)
			}
			continue
		case *ast.ConditionalJump:
			child, ok := labelled[jump.Label]
			if !ok {
				initializer.Emit(jump.Loc(), "undefined block label (%s)", jump.Label)
				names[jump.Label] = struct{}{}
			} else {
				block.Children = append(block.Children, child)
				child.Parents = append(child.Parents, block)
				prevChild = child
			}
		case *ast.Terminal:
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

		// Conditional jump branches could both point to the same child.
		// Only include the child once in this case.
		if prevChild != child {
			block.Children = append(block.Children, child)
			child.Parents = append(child.Parents, block)
		}
	}

	initializer.checkForUnreachableBlocks(def)
}

func (initializer *controlFlowGraphInitializer) checkForUnreachableBlocks(
	def *ast.FunctionDefinition,
) {
	_, reachable := util.DFS(def)
	for _, block := range def.Blocks {
		_, ok := reachable[block]
		if !ok {
			label := ""
			if block.Label != "" {
				label = " (" + block.Label + ")"
			}
			initializer.Emit(block.Loc(), "block%s not reachable", label)
		}
	}
}

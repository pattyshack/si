package analyzer

import (
	"strconv"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
)

type controlFlowGraphInitializer struct {
	*parseutil.Emitter
}

func InitializeControlFlowGraph(
	emitter *parseutil.Emitter,
) Pass[ast.SourceEntry] {
	return &controlFlowGraphInitializer{
		Emitter: emitter,
	}
}

func (initializer *controlFlowGraphInitializer) Process(
	entry ast.SourceEntry,
) {
	def, ok := entry.(*ast.FuncDefinition)
	if !ok {
		return
	}

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
		case *ast.Terminal:
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

	// Insert a pseudo entry block if the real entry block is also a loop header.
	if len(def.Blocks[0].Parents) != 0 {
		realBlock := def.Blocks[0]

		entryBlock := &ast.Block{
			StartEndPos: parseutil.NewStartEndPos(realBlock.Loc(), realBlock.Loc()),
			Children:    []*ast.Block{realBlock},
		}
		realBlock.Parents = append([]*ast.Block{entryBlock}, realBlock.Parents...)

		def.Blocks = append([]*ast.Block{entryBlock}, def.Blocks...)
	}

	initializer.checkForUnreachableBlocks(def)

	// Add labels for internal debugging purpose
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

func (initializer *controlFlowGraphInitializer) checkForUnreachableBlocks(
	def *ast.FuncDefinition,
) {
	reachable := map[*ast.Block]struct{}{
		def.Blocks[0]: struct{}{},
	}

	queue := []*ast.Block{def.Blocks[0]}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		for _, child := range node.Children {
			_, ok := reachable[child]
			if !ok {
				reachable[child] = struct{}{}
				queue = append(queue, child)
			}
		}
	}

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

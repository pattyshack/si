package util

import (
	"sync"

	"github.com/pattyshack/chickadee/ast"
)

type Pass[T any] interface {
	Process(T)
}

func Process[T any](
	node T,
	passes [][]Pass[T], // sequence of parallelizable passes
	shouldEarlyExit func() bool, // optional
) {
	for _, parallelPasses := range passes {
		wg := sync.WaitGroup{}
		wg.Add(len(parallelPasses))
		for _, pass := range parallelPasses {
			go func(pass Pass[T]) {
				pass.Process(node)
				wg.Done()
			}(pass)
		}

		wg.Wait()

		if shouldEarlyExit != nil && shouldEarlyExit() {
			return
		}
	}
}

func ParallelProcess[Node ast.Node](
	list []Node,
	process func(Node),
) {
	wg := sync.WaitGroup{}
	wg.Add(len(list))
	for _, item := range list {
		go func(item Node) {
			process(item)
			wg.Done()
		}(item)
	}
	wg.Wait()
}

type DataFlowWorkSet struct {
	queue []*ast.Block
	set   map[*ast.Block]struct{}
}

func NewDataflowWorkSet() *DataFlowWorkSet {
	return &DataFlowWorkSet{
		set: map[*ast.Block]struct{}{},
	}
}

func (set *DataFlowWorkSet) IsEmpty() bool {
	return len(set.queue) == 0
}

func (set *DataFlowWorkSet) Push(block *ast.Block) {
	_, ok := set.set[block]
	if ok {
		return
	}
	set.set[block] = struct{}{}
	set.queue = append(set.queue, block)
}

func (set *DataFlowWorkSet) Pop() *ast.Block {
	head := set.queue[0]
	set.queue = set.queue[1:]
	delete(set.set, head)
	return head
}

func DFS(
	funcDef *ast.FunctionDefinition,
) (
	[]*ast.Block,
	map[*ast.Block]struct{},
) {
	stack := make([]*ast.Block, 0, len(funcDef.Blocks))
	stack = append(stack, funcDef.Blocks[0])

	visited := make(map[*ast.Block]struct{}, len(funcDef.Blocks))
	order := make([]*ast.Block, 0, len(funcDef.Blocks))
	for len(stack) > 0 {
		idx := len(stack) - 1
		top := stack[idx]
		stack = stack[:idx]

		_, ok := visited[top]
		if ok {
			continue
		}

		visited[top] = struct{}{}
		order = append(order, top)

		// NOTE: This visits the fallthrough child before the branch child due to
		// the way control flow graph is initialized.
		stack = append(stack, top.Children...)
	}

	return order, visited
}

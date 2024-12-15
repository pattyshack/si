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

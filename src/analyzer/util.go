package analyzer

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

func ParallelWalk[Node ast.Node](
	list []Node,
	newVisitor func(Node) ast.Visitor,
) {
	wg := sync.WaitGroup{}
	wg.Add(len(list))

	for _, n := range list {
		v := newVisitor(n)
		go func(node Node, visitor ast.Visitor) {
			node.Walk(visitor)
			wg.Done()
		}(n, v)
	}

	wg.Wait()
}

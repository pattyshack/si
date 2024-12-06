package analyzer

import (
	"sync"

	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/platform"
)

type callRetConstraintsCollector struct {
	platform platform.Platform

	wg sync.WaitGroup

	mutex            sync.Mutex
	entryConstraints map[ast.SourceEntry]*platform.InstructionConstraints
}

func NewCallRetConstraintsCollector(
	targetPlatform platform.Platform,
	numEntries int,
) *callRetConstraintsCollector {
	collector := &callRetConstraintsCollector{
		platform:         targetPlatform,
		entryConstraints: map[ast.SourceEntry]*platform.InstructionConstraints{},
	}
	collector.wg.Add(numEntries)
	return collector
}

func (collector *callRetConstraintsCollector) Process(entry ast.SourceEntry) {
	defer collector.wg.Done()

	funcDef, ok := entry.(*ast.FunctionDefinition)
	if !ok {
		return
	}

	callSpec := collector.platform.CallSpec(funcDef.CallConvention)
	constraints := callSpec.CallRetConstraints(funcDef.Type().(ast.FunctionType))

	collector.mutex.Lock()
	defer collector.mutex.Unlock()

	collector.entryConstraints[entry] = constraints
}

func (collector *callRetConstraintsCollector) Constraints() map[ast.SourceEntry]*platform.InstructionConstraints {
	collector.wg.Wait()
	return collector.entryConstraints
}

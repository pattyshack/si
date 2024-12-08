package analyzer

import (
	"fmt"
	"strings"

	"github.com/pattyshack/chickadee/ast"
)

type livenessPrinter struct {
	livenessAnalyzer
}

// This is only for debugging purpose.
func PrintLiveness() Pass[ast.SourceEntry] {
	return &livenessPrinter{
		livenessAnalyzer: livenessAnalyzer{
			live: map[*ast.Block]*liveInOut{},
		},
	}
}

func (printer *livenessPrinter) Process(entry ast.SourceEntry) {
	funcDef, ok := entry.(*ast.FunctionDefinition)
	if !ok {
		return
	}

	printer.livenessAnalyzer.Process(funcDef)

	result := fmt.Sprintf("Definition: %s\n", funcDef.Label)
	result += fmt.Sprintf(
		"  # of callee saved registers: %d\n",
		len(funcDef.PseudoParameters))
	for idx, block := range funcDef.Blocks {
		result += fmt.Sprintf("  Block %d (%s):\n", idx, block.Label)

		result += fmt.Sprintf("    LiveIn:\n")
		calleeSavedCount := 0
		for def, dist := range printer.live[block].in {
			if strings.HasPrefix(def.Name, "%") {
				calleeSavedCount++
				continue
			}
			result += fmt.Sprintf("      %s %d (%s)\n", def.Name, dist, def.Loc())
		}
		if calleeSavedCount != len(funcDef.PseudoParameters) {
			panic("should never happen")
		}

		result += fmt.Sprintf("    LiveOut:\n")
		calleeSavedCount = 0
		for def, dist := range printer.live[block].out {
			if strings.HasPrefix(def.Name, "%") {
				calleeSavedCount++
				continue
			}
			result += fmt.Sprintf("      %s %d (%s)\n", def.Name, dist, def.Loc())
		}
		if len(block.Children) > 0 &&
			calleeSavedCount != len(funcDef.PseudoParameters) {
			panic("should never happen")
		}
	}

	fmt.Println(result)
}

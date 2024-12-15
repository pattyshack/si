package allocator

import (
	"fmt"
	"strings"

	"github.com/pattyshack/chickadee/analyzer/util"
	"github.com/pattyshack/chickadee/ast"
)

type LivenessPrinter struct {
	*LivenessAnalyzer
}

// This is only for debugging purpose.
func PrintLiveness() util.Pass[ast.SourceEntry] {
	return &LivenessPrinter{
		LivenessAnalyzer: NewLivenessAnalyzer(),
	}
}

func (printer *LivenessPrinter) Process(entry ast.SourceEntry) {
	funcDef, ok := entry.(*ast.FunctionDefinition)
	if !ok {
		return
	}

	printer.LivenessAnalyzer.Process(funcDef)

	result := fmt.Sprintf("Definition: %s\n", funcDef.Label)
	result += fmt.Sprintf(
		"  # of callee saved registers: %d\n",
		len(funcDef.PseudoParameters))
	for idx, block := range funcDef.Blocks {
		result += fmt.Sprintf("  Block %d (%s):\n", idx, block.Label)

		result += fmt.Sprintf("    LiveIn:\n")
		calleeSavedCount := 0
		for def, dist := range printer.LiveIn[block] {
			if strings.HasPrefix(def.Name, "%") {
				calleeSavedCount++
				continue
			}
			result += fmt.Sprintf("      %s %d (%s)\n", def.Name, dist, def.Loc())
		}

		result += fmt.Sprintf("    LiveOut:\n")
		calleeSavedCount = 0
		for def, dist := range printer.LiveOut[block] {
			if strings.HasPrefix(def.Name, "%") {
				calleeSavedCount++
				continue
			}
			result += fmt.Sprintf("      %s %d (%s)\n", def.Name, dist, def.Loc())
		}
	}

	fmt.Println(result)
}

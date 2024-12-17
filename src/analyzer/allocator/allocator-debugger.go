package allocator

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pattyshack/chickadee/analyzer/util"
	"github.com/pattyshack/chickadee/ast"
)

type AllocatorDebugger struct {
	*Allocator
}

func Debug(allocator *Allocator) util.Pass[ast.SourceEntry] {
	return &AllocatorDebugger{
		Allocator: allocator,
	}
}

func (debugger *AllocatorDebugger) Process(entry ast.SourceEntry) {
	funcDef, ok := entry.(*ast.FunctionDefinition)
	if !ok {
		return
	}

	buffer := &bytes.Buffer{}
	printf := func(template string, args ...interface{}) {
		fmt.Fprintf(buffer, template, args...)
	}

	printf("Definition: %s\n", funcDef.Label)
	printf("------------------------------------------\n")
	printf("Liveness:\n")
	printf("  # of callee saved registers: %d\n", len(funcDef.PseudoParameters))

	for idx, block := range funcDef.Blocks {
		blockState := debugger.BlockStates[block]

		printf("  Block %d (%s):\n", idx, block.Label)

		printf("    LiveIn:\n")
		calleeSavedCount := 0
		for def, info := range blockState.LiveIn {
			if strings.HasPrefix(def.Name, "%") {
				calleeSavedCount++
				continue
			}
			printf("      %s : %d (%s)\n", def.Name, info.Distance, def.Loc())
		}

		printf("    LiveOut:\n")
		calleeSavedCount = 0
		for def, info := range blockState.LiveOut {
			if strings.HasPrefix(def.Name, "%") {
				calleeSavedCount++
				continue
			}
			printf("      %s : %d (%s)\n", def.Name, info.Distance, def.Loc())
		}
	}

	printf("------------------------------------------\n")
	printf("Live Ranges:\n")
	for idx, block := range funcDef.Blocks {
		blockState := debugger.BlockStates[block]

		printf("  Block %d (%s):\n", idx, block.Label)
		for def, liveRange := range blockState.LiveRanges {
			printf(
				"    %s: [%d %d] (%s)\n",
				def.Name,
				liveRange.Start,
				liveRange.End,
				def.Loc())
		}
	}

	printf("------------------------------------------\n")
	printf("Data Locations:\n")
	for idx, block := range funcDef.Blocks {
		blockState := debugger.BlockStates[block]

		printf("  Block %d (%s):\n", idx, block.Label)
		printf("    LocationIn:\n")
		for _, loc := range blockState.LocationIn {
			printf("      %s\n", loc)
		}

		printf("    LocationOut:\n")
		for _, loc := range blockState.LocationOut {
			printf("      %s\n", loc)
		}
	}

	printf("------------------------------------------\n")
	printf("Stack Frame (Size = %d):\n", debugger.StackFrame.FrameSize)
	printf("  Layout (bottom to top):\n")
	for _, entry := range debugger.StackFrame.Layout {
		printf("    %s\n", entry)
	}
	printf("==========================================\n")

	fmt.Println(buffer.String())
}

package analyzer

import (
	"fmt"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
)

func CollectSignatures(
	entries []ast.SourceEntry,
	emitter *parseutil.Emitter,
) map[string]ast.SourceEntry {
	result := map[string]ast.SourceEntry{}
	for _, source := range entries {
		if source.HasDeclarationSyntaxError() {
			continue
		}

		switch entry := source.(type) {
		case *ast.FunctionDefinition:
			prev, ok := result[entry.Label]
			if ok {
				emitter.Emit(
					entry.Loc(),
					"definition (%s) previously defined at (%s)",
					entry.Label,
					prev.Loc())
				continue
			}

			result[entry.Label] = entry
		default:
			panic(fmt.Sprintf("%s: unhandled SourceEntry", source.Loc()))
		}
	}

	return result
}

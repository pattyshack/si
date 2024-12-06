package analyzer

import (
	"fmt"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
)

type SignatureCollector struct {
	*parseutil.Emitter
	signatures map[string]ast.SourceEntry
}

func NewSignatureCollector(emitter *parseutil.Emitter) *SignatureCollector {
	return &SignatureCollector{
		Emitter:    emitter,
		signatures: map[string]ast.SourceEntry{},
	}
}

func (collector *SignatureCollector) Signatures() map[string]ast.SourceEntry {
	return collector.signatures
}

func (collector *SignatureCollector) Process(entries []ast.SourceEntry) {
	for _, source := range entries {
		if source.HasDeclarationSyntaxError() {
			continue
		}

		switch entry := source.(type) {
		case *ast.FunctionDefinition:
			prev, ok := collector.signatures[entry.Label]
			if ok {
				collector.Emit(
					entry.Loc(),
					"definition (%s) previously defined at (%s)",
					entry.Label,
					prev.Loc())
				continue
			}

			collector.signatures[entry.Label] = entry
		default:
			panic(fmt.Sprintf("%s: unhandled SourceEntry", source.Loc()))
		}
	}
}

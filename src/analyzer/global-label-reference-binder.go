package analyzer

import (
	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/analyzer/util"
	"github.com/pattyshack/chickadee/ast"
)

type globalLabelReferenceBinder struct {
	*parseutil.Emitter

	signatures map[string]ast.SourceEntry
}

func BindGlobalLabelReferences(
	emitter *parseutil.Emitter,
	signatures map[string]ast.SourceEntry,
) util.Pass[ast.SourceEntry] {
	return &globalLabelReferenceBinder{
		Emitter:    emitter,
		signatures: signatures,
	}
}

func (binder *globalLabelReferenceBinder) Process(entry ast.SourceEntry) {
	funcDef, ok := entry.(*ast.FunctionDefinition)
	if !ok {
		return
	}

	for _, block := range funcDef.Blocks {
		block.ParentFuncDef = funcDef
		for _, inst := range block.Instructions {
			inst.SetParentBlock(block)
			for _, src := range inst.Sources() {
				src.SetParentInstruction(inst)

				ref, ok := src.(*ast.GlobalLabelReference)
				if !ok {
					continue
				}

				sig, ok := binder.signatures[ref.Label]
				if !ok {
					binder.Emit(ref.Loc(), "global label (%s) not defined", ref.Label)
					continue
				}

				ref.Signature = sig
			}

			dest := inst.Destination()
			if dest != nil {
				dest.ParentInstruction = inst
			}
		}
	}
}

package analyzer

import (
	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/platform"
)

type terminalPseudoEntriesPopulator struct {
	platform    platform.Platform
	constraints *callRetConstraints
}

func PopulateTerminalPseudoEntries(
	targetPlatform platform.Platform,
	constraints *callRetConstraints,
) Pass[ast.SourceEntry] {
	return &terminalPseudoEntriesPopulator{
		platform:    targetPlatform,
		constraints: constraints,
	}
}

func (populator terminalPseudoEntriesPopulator) Process(
	entry ast.SourceEntry,
) {
	funcDef, ok := entry.(*ast.FunctionDefinition)
	if !ok {
		return
	}

	populator.constraints.Ready()

	for _, block := range funcDef.Blocks {
		if len(block.Children) > 0 {
			continue
		}

		term := block.Instructions[len(block.Instructions)-1].(*ast.Terminal)
		switch term.Kind {
		case ast.Ret:
			for _, def := range funcDef.PseudoParameters {
				ref := def.NewRef(term.StartEnd())
				ref.SetParentInstruction(term)
				term.PseudoSources = append(term.PseudoSources, ref)
			}
		case ast.Exit:
			sysCallSpec := populator.platform.SysCallSpec()
			term.ExitSysCall = &ast.FuncCall{
				StartEndPos: term.StartEndPos,
				Kind:        ast.SysCall,
				Dest: &ast.VariableDefinition{
					StartEndPos: term.StartEndPos,
					// It's unlike this name will conflict with any real register name
					Name: "%%ignore-exit-syscall-return-value%%",
					Type: ast.NewU32(term.StartEndPos),
				},
				Func: sysCallSpec.ExitSysCallFuncValue(term.StartEndPos),
				Args: []ast.Value{term.Src},
			}
			term.ExitSysCall.SetParentBlock(term.ParentBlock())
			// XXX: maybe use term.ExitSysCall as parent instruction instead?
			term.ExitSysCall.Dest.ParentInstruction = term
		default:
			panic("unhandled terminal kind: " + term.Kind)
		}
	}
}

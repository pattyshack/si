package analyzer

import (
	"fmt"

	"github.com/pattyshack/chickadee/analyzer/util"
	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/platform"
)

type terminalModifier struct {
	platform platform.Platform
}

func ModifyTerminals(
	targetPlatform platform.Platform,
) util.Pass[ast.SourceEntry] {
	return &terminalModifier{
		platform: targetPlatform,
	}
}

func (modifier *terminalModifier) Process(
	entry ast.SourceEntry,
) {
	funcDef, ok := entry.(*ast.FunctionDefinition)
	if !ok {
		return
	}

	sysCallSpec := modifier.platform.SysCallSpec()
	for _, block := range funcDef.Blocks {
		if len(block.Children) > 0 {
			continue
		}

		term := block.Instructions[len(block.Instructions)-1].(*ast.Terminal)
		switch term.Kind {
		case ast.Ret:
			for _, def := range funcDef.CalleeSavedParameters {
				ref := def.NewRef(term.StartEnd())
				ref.SetParentInstruction(term)
				term.CalleeSavedSources = append(term.CalleeSavedSources, ref)
			}
		case ast.Exit:
			exitSysCall := &ast.FuncCall{
				StartEndPos: term.StartEndPos,
				Kind:        ast.SysCall,
				Dest: &ast.VariableDefinition{
					StartEndPos: term.StartEndPos,
					// It's unlike this name will conflict with any real register name
					Name: "%%ignore-exit-syscall-return-value%%",
					Type: sysCallSpec.ReturnType(term.StartEndPos),
				},
				Func: sysCallSpec.ExitSysCallFuncValue(term.StartEndPos),
				Args: []ast.Value{term.RetVal},
			}

			block.Instructions[len(block.Instructions)-1] = exitSysCall
		default:
			panic(fmt.Sprintf("unhandled terminal kind: %s", term.Loc()))
		}
	}
}

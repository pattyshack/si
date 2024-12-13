package analyzer

import (
	"fmt"
	"strings"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/platform"
)

type funcDefConstraintsGenerator struct {
	*parseutil.Emitter

	platform platform.Platform
}

func GenerateFuncDefConstraints(
	emitter *parseutil.Emitter,
	platform platform.Platform,
) Pass[ast.SourceEntry] {
	return &funcDefConstraintsGenerator{
		Emitter:  emitter,
		platform: platform,
	}
}

func (generator *funcDefConstraintsGenerator) Process(entry ast.SourceEntry) {
	funcDef, ok := entry.(*ast.FunctionDefinition)
	if !ok {
		return
	}

	hasCallConventionError := false

	callSpec := generator.platform.CallSpec(funcDef.CallConvention)
	for _, def := range funcDef.Parameters {
		if def.Type == nil {
			hasCallConventionError = true
			continue // error previously emitted
		}

		if !callSpec.IsValidArgType(def.Type) {
			hasCallConventionError = true
			generator.Emit(
				def.Type.Loc(),
				"%s call convention does not support %s argument type",
				funcDef.CallConvention,
				def.Type)
		}
	}

	if !callSpec.IsValidReturnType(funcDef.ReturnType) {
		hasCallConventionError = true
		generator.Emit(
			funcDef.ReturnType.Loc(),
			"%s call convention does not support %s return type",
			funcDef.CallConvention,
			funcDef.ReturnType)
	}

	if hasCallConventionError {
		return
	}

	constraints, pseudoParams := callSpec.CallRetConstraints(
		funcDef.Type().(ast.FunctionType))

	for _, param := range pseudoParams {
		if !strings.HasPrefix(param.Name, "%") {
			// call spec implementation error
			panic(fmt.Sprintf(
				"(%s) call spec implementation error",
				funcDef.CallConvention))
		}
	}

	funcDef.PseudoParameters = pseudoParams
	funcDef.CallRetConstraints = constraints
}

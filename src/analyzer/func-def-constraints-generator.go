package analyzer

import (
	"sort"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/architecture"
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

	callSpec := generator.platform.CallSpec(funcDef.CallConvention)
	if generator.failCallTypeRestriction(funcDef, callSpec) {
		return
	}

	convention := callSpec.CallRetConstraints(
		funcDef.Type().(ast.FunctionType))
	funcDef.CallConventionSpec = convention

	funcDef.PseudoParameters = generator.generatePseudoParameters(
		convention,
		funcDef.StartEnd())

	// Always insert a pseudo entry block to copy callee-saved argument variables
	// and ensures the first block is not a loop header.
	entryBlock := &ast.Block{
		StartEndPos: parseutil.NewStartEndPos(funcDef.Loc(), funcDef.Loc()),
	}
	funcDef.Blocks = append([]*ast.Block{entryBlock}, funcDef.Blocks...)

	calleeSavedParameters := []*ast.VariableDefinition{}
	for _, idx := range convention.CalleeSavedSourceIndices {
		// Rename the callee-saved parameter to keep the value throughout the
		// function, and copy the callee-saved parameter.
		param := funcDef.Parameters[idx]
		calleeSavedParameters = append(calleeSavedParameters, param)

		origParamName := param.Name
		param.Name = "%%" + origParamName

		entryBlock.Instructions = append(
			entryBlock.Instructions,
			&ast.CopyOperation{
				StartEndPos: param.StartEnd(),
				Dest: &ast.VariableDefinition{
					StartEndPos: param.StartEnd(),
					Name:        origParamName,
				},
				Src: &ast.VariableReference{
					StartEndPos: param.StartEnd(),
					Name:        param.Name,
				},
			})
	}

	calleeSavedParameters = append(
		calleeSavedParameters,
		funcDef.PseudoParameters...)

	funcDef.CalleeSavedParameters = calleeSavedParameters
}

func (generator *funcDefConstraintsGenerator) generatePseudoParameters(
	convention *architecture.CallConvention,
	pos parseutil.StartEndPos,
) []*ast.VariableDefinition {
	pseudoSourceRegisters := map[*architecture.Register]struct{}{}
	for reg, clobbered := range convention.CallConstraints.RequiredRegisters {
		if clobbered {
			continue
		}
		pseudoSourceRegisters[reg] = struct{}{}
	}

	for _, src := range convention.CallConstraints.Sources {
		for _, reg := range src.Registers {
			delete(pseudoSourceRegisters, reg.Require)
		}
	}

	sorted := []*architecture.Register{}
	for reg, _ := range pseudoSourceRegisters {
		sorted = append(sorted, reg)
	}
	sort.Slice(
		sorted,
		func(i int, j int) bool { return sorted[i].Name < sorted[j].Name })

	pseudoParameters := []*ast.VariableDefinition{}
	for _, reg := range sorted {
		convention.CallConstraints.AddPseudoSource(
			convention.CallConstraints.Require(false, reg))
		convention.RetConstraints.AddPseudoSource(
			convention.RetConstraints.Require(false, reg))

		var regType ast.Type
		if reg.AllowGeneralOp {
			regType = ast.NewU64(pos)
		} else {
			regType = ast.NewF64(pos)
		}
		pseudoParam := &ast.VariableDefinition{
			StartEndPos: pos,
			Name:        "%" + reg.Name,
			Type:        regType,
			DefUses:     map[*ast.VariableReference]struct{}{},
		}

		pseudoParameters = append(pseudoParameters, pseudoParam)
	}

	return pseudoParameters
}

func (generator *funcDefConstraintsGenerator) failCallTypeRestriction(
	funcDef *ast.FunctionDefinition,
	callSpec platform.CallSpec,
) bool {
	hasCallConventionError := false

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

	return hasCallConventionError
}

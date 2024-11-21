package reducer

import (
	"strconv"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/parser/lr"
)

type Reducer struct{}

var _ lr.Reducer = Reducer{}

func (Reducer) DataToDeclaration(
	declare *lr.TokenValue,
	label *ast.GlobalLabelReference,
	typeExpr ast.Type,
) (
	ast.Line,
	error,
) {
	return &ast.Declaration{
		StartEndPos: parseutil.NewStartEndPos(declare.Loc(), typeExpr.End()),
		Kind:        ast.DataDeclaration,
		Label:       label.Label,
		Type:        typeExpr,
	}, nil
}

func (Reducer) FuncToDeclaration(
	declare *lr.TokenValue,
	funcKW *lr.TokenValue,
	label *ast.GlobalLabelReference,
	lparen *lr.TokenValue,
	parameters []ast.Type,
	rparen *lr.TokenValue,
	retType ast.Type,
) (
	ast.Line,
	error,
) {
	return &ast.Declaration{
		StartEndPos: parseutil.NewStartEndPos(declare.Loc(), retType.End()),
		Kind:        ast.FuncDeclaration,
		Label:       label.Label,
		Type: ast.FunctionType{
			ParameterTypes: parameters,
			ReturnType:     retType,
		},
	}, nil
}

func (Reducer) FuncToDefinition(
	define *lr.TokenValue,
	funcKW *lr.TokenValue,
	label *ast.GlobalLabelReference,
	lparen *lr.TokenValue,
	parameters []*ast.RegisterDefinition,
	rparen *lr.TokenValue,
	retType ast.Type,
	lbrace *lr.TokenValue,
) (
	ast.Line,
	error,
) {
	return &ast.FuncDefinition{
		StartEndPos: parseutil.NewStartEndPos(define.Loc(), lbrace.End()),
		Label:       label.Label,
		Parameters:  parameters,
		ReturnType:  retType,
	}, nil
}

func (Reducer) ToRbrace(
	rbrace *lr.TokenValue,
) (
	ast.Line,
	error,
) {
	return lr.ParsedRbrace{
		StartEndPos: rbrace.StartEndPos,
	}, nil
}

func (Reducer) InferredToRegisterDefinition(
	ref *ast.RegisterReference,
) (
	*ast.RegisterDefinition,
	error,
) {
	return &ast.RegisterDefinition{
		StartEndPos: ref.StartEndPos,
		Name:        ref.Name,
	}, nil
}

func (Reducer) ToTypedRegisterDefinition(
	ref *ast.RegisterReference,
	typeExpr ast.Type,
) (
	*ast.RegisterDefinition,
	error,
) {
	return &ast.RegisterDefinition{
		StartEndPos: parseutil.NewStartEndPos(ref.Loc(), typeExpr.End()),
		Name:        ref.Name,
		Type:        typeExpr,
	}, nil
}

func (Reducer) ToGlobalLabel(
	at *lr.TokenValue,
	identifier *lr.TokenValue,
) (
	*ast.GlobalLabelReference,
	error,
) {
	return &ast.GlobalLabelReference{
		StartEndPos: parseutil.NewStartEndPos(at.StartPos, identifier.EndPos),
		Label:       identifier.Value,
	}, nil
}

func (Reducer) ToLocalLabel(
	pound *lr.TokenValue,
	identifier *lr.TokenValue,
) (
	lr.ParsedLocalLabel,
	error,
) {
	return lr.ParsedLocalLabel{
		StartEndPos: parseutil.NewStartEndPos(pound.StartPos, identifier.EndPos),
		Label:       identifier.Value,
	}, nil
}

func (Reducer) ToRegisterReference(
	percent *lr.TokenValue,
	identifier *lr.TokenValue,
) (
	*ast.RegisterReference,
	error,
) {
	return &ast.RegisterReference{
		StartEndPos: parseutil.NewStartEndPos(percent.StartPos, identifier.EndPos),
		Name:        identifier.Value,
	}, nil
}

func (Reducer) ToIntImmediate(
	token *lr.TokenValue,
) (
	ast.Value,
	error,
) {
	value, err := strconv.ParseInt(token.Value, 0, 64)
	if err != nil {
		return nil, parseutil.NewLocationError(
			token.Loc(),
			"failed to parse int (%s): %w",
			token.Value,
			err)
	}

	return &ast.IntImmediate{
		StartEndPos: token.StartEndPos,
		Value:       value,
	}, nil
}

func (Reducer) ToFloatImmediate(
	token *lr.TokenValue,
) (
	ast.Value,
	error,
) {
	value, err := strconv.ParseFloat(token.Value, 64)
	if err != nil {
		return nil, parseutil.NewLocationError(
			token.Loc(),
			"failed to parse float (%s): %w",
			token.Value,
			err)
	}

	return &ast.FloatImmediate{
		StartEndPos: token.StartEndPos,
		Value:       value,
	}, nil
}

func (Reducer) StringToIdentifier(
	token *lr.TokenValue,
) (
	*lr.TokenValue,
	error,
) {
	token.Value = parseutil.Unescape(token.Value[1 : len(token.Value)-1])
	return token, nil
}

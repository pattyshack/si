package reducer

import (
	"strconv"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/parser/lr"
)

type Reducer struct{}

var _ lr.Reducer = Reducer{}

func (Reducer) FuncToDefinition(
	define *lr.TokenValue,
	funcKW *lr.TokenValue,
	label *ast.GlobalLabelReference,
	lparen *lr.TokenValue,
	parameters []*ast.VariableDefinition,
	rparen *lr.TokenValue,
	retType ast.Type,
	lbrace *lr.TokenValue,
) (
	ast.Line,
	error,
) {
	return &ast.FunctionDefinition{
		StartEndPos: parseutil.NewStartEndPos(define.Loc(), lbrace.End()),
		// TODO: make call convention configurable
		CallConventionName: ast.DefaultCallConvention,
		Label:              label.Label,
		Parameters:         parameters,
		ReturnType:         retType,
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

func (Reducer) InferredToVariableDefinition(
	ref *ast.VariableReference,
) (
	*ast.VariableDefinition,
	error,
) {
	return &ast.VariableDefinition{
		StartEndPos: ref.StartEndPos,
		Name:        ref.Name,
	}, nil
}

func (Reducer) ToTypedVariableDefinition(
	ref *ast.VariableReference,
	typeExpr ast.Type,
) (
	*ast.VariableDefinition,
	error,
) {
	return &ast.VariableDefinition{
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

func (Reducer) ToVariableReference(
	percent *lr.TokenValue,
	identifier *lr.TokenValue,
) (
	*ast.VariableReference,
	error,
) {
	return &ast.VariableReference{
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
	isNegative := false
	bytes := []byte(token.Value)
	if len(bytes) > 1 && bytes[0] == '-' {
		isNegative = true
		bytes = bytes[1:]
	}

	value, err := strconv.ParseUint(string(bytes), 0, 64)
	if err != nil {
		return nil, parseutil.NewLocationError(
			token.Loc(),
			"failed to parse int (%s): %w",
			token.Value,
			err)
	}

	return ast.NewIntImmediate(token.StartEnd(), value, isNegative), nil
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

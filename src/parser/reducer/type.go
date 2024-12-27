package reducer

import (
	"strings"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/parser/lr"
)

func (Reducer) ToNumberType(
	token *lr.TokenValue,
) (
	ast.Type,
	error,
) {
	if strings.HasPrefix(token.Value, "F") {
		return &ast.FloatType{
			StartEndPos: token.StartEndPos,
			Kind:        ast.FloatTypeKind(token.Value),
		}, nil
	} else if strings.HasPrefix(token.Value, "U") {
		return &ast.UnsignedIntType{
			StartEndPos: token.StartEndPos,
			Kind:        ast.UnsignedIntTypeKind(token.Value),
		}, nil
	} else {
		return &ast.SignedIntType{
			StartEndPos: token.StartEndPos,
			Kind:        ast.SignedIntTypeKind(token.Value),
		}, nil
	}
}

func (Reducer) ToFuncType(
	funcKW *lr.TokenValue,
	lparen *lr.TokenValue,
	parameterTypes []ast.Type,
	rparen *lr.TokenValue,
	retType ast.Type,
) (
	ast.Type,
	error,
) {
	return &ast.FunctionType{
		StartEndPos: parseutil.NewStartEndPos(funcKW.Loc(), retType.End()),
		// TODO: make call convention configurable
		CallConvention: ast.DefaultCallConvention,
		ParameterTypes: parameterTypes,
		ReturnType:     retType,
	}, nil
}

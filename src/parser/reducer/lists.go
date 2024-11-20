package reducer

import (
	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/parser/lr"
)

func (Reducer) ImproperToParameters(
	list []*ast.RegisterDefinition,
	comma *lr.TokenValue,
) (
	[]*ast.RegisterDefinition,
	error,
) {
	return list, nil
}

func (Reducer) NilToParameters() (
	[]*ast.RegisterDefinition,
	error,
) {
	return nil, nil
}

func (Reducer) AddToProperParameters(
	list []*ast.RegisterDefinition,
	comma *lr.TokenValue,
	def *ast.RegisterDefinition,
) (
	[]*ast.RegisterDefinition,
	error,
) {
	return append(list, def), nil
}

func (Reducer) NewToProperParameters(
	def *ast.RegisterDefinition,
) (
	[]*ast.RegisterDefinition,
	error,
) {
	return []*ast.RegisterDefinition{def}, nil
}

func (Reducer) ImproperToArguments(
	list []ast.Value,
	comma *lr.TokenValue,
) (
	[]ast.Value,
	error,
) {
	return list, nil
}

func (Reducer) NilToArguments() (
	[]ast.Value,
	error,
) {
	return nil, nil
}

func (Reducer) AddToProperArguments(
	list []ast.Value,
	comma *lr.TokenValue,
	arg ast.Value,
) (
	[]ast.Value,
	error,
) {
	return append(list, arg), nil
}

func (Reducer) NewToProperArguments(
	arg ast.Value,
) (
	[]ast.Value,
	error,
) {
	return []ast.Value{arg}, nil
}

func (Reducer) ImproperToTypes(
	list []ast.Type,
	comma *lr.TokenValue,
) (
	[]ast.Type,
	error,
) {
	return list, nil
}

func (Reducer) NilToTypes() (
	[]ast.Type,
	error,
) {
	return nil, nil
}

func (Reducer) AddToProperTypes(
	list []ast.Type,
	comma *lr.TokenValue,
	typeExpr ast.Type,
) (
	[]ast.Type,
	error,
) {
	return append(list, typeExpr), nil
}

func (Reducer) NewToProperTypes(
	typeExpr ast.Type,
) (
	[]ast.Type,
	error,
) {
	return []ast.Type{typeExpr}, nil
}

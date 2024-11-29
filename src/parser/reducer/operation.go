package reducer

import (
	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/parser/lr"
)

func (Reducer) AssignToOperationInstruction(
	dest *ast.VariableDefinition,
	equal *lr.TokenValue,
	src ast.Value,
) (
	ast.Instruction,
	error,
) {
	return &ast.AssignOperation{
		StartEndPos: parseutil.NewStartEndPos(dest.Loc(), src.End()),
		Dest:        dest,
		Src:         src,
	}, nil
}

func (Reducer) UnaryToOperationInstruction(
	dest *ast.VariableDefinition,
	equal *lr.TokenValue,
	op *lr.TokenValue,
	src ast.Value,
) (
	ast.Instruction,
	error,
) {
	return &ast.UnaryOperation{
		StartEndPos: parseutil.NewStartEndPos(dest.Loc(), src.End()),
		Dest:        dest,
		Kind:        ast.UnaryOperationKind(op.Value),
		Src:         src,
	}, nil
}

func (Reducer) BinaryToOperationInstruction(
	dest *ast.VariableDefinition,
	equal *lr.TokenValue,
	op *lr.TokenValue,
	src1 ast.Value,
	comma *lr.TokenValue,
	src2 ast.Value,
) (
	ast.Instruction,
	error,
) {
	return &ast.BinaryOperation{
		StartEndPos: parseutil.NewStartEndPos(dest.Loc(), src2.End()),
		Dest:        dest,
		Kind:        ast.BinaryOperationKind(op.Value),
		Src1:        src1,
		Src2:        src2,
	}, nil
}

func (Reducer) CallToOperationInstruction(
	dest *ast.VariableDefinition,
	equal *lr.TokenValue,
	callKind *lr.TokenValue,
	funcLoc ast.Value,
	lparen *lr.TokenValue,
	args []ast.Value,
	rparen *lr.TokenValue,
) (
	ast.Instruction,
	error,
) {
	return &ast.FuncCall{
		StartEndPos: parseutil.NewStartEndPos(dest.Loc(), rparen.End()),
		Kind:        ast.FuncCallKind(callKind.Value),
		Dest:        dest,
		Func:        funcLoc,
		Args:        args,
	}, nil
}

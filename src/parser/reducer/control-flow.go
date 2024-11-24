package reducer

import (
	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/parser/lr"
)

func (Reducer) UnconditionalToControlFlowInstruction(
	op *lr.TokenValue,
	label lr.ParsedLocalLabel,
) (
	ast.Instruction,
	error,
) {
	return &ast.Jump{
		StartEndPos: parseutil.NewStartEndPos(op.Loc(), label.End()),
		Kind:        ast.JumpKind(op.Value),
		Label:       label.Label,
	}, nil
}

func (Reducer) ConditionalToControlFlowInstruction(
	op *lr.TokenValue,
	label lr.ParsedLocalLabel,
	comma1 *lr.TokenValue,
	src1 ast.Value,
	comma2 *lr.TokenValue,
	src2 ast.Value,
) (
	ast.Instruction,
	error,
) {
	return &ast.ConditionalJump{
		StartEndPos: parseutil.NewStartEndPos(op.Loc(), label.End()),
		Kind:        ast.ConditionalJumpKind(op.Value),
		Src1:        src1,
		Src2:        src2,
		Label:       label.Label,
	}, nil
}

func (Reducer) TerminalToControlFlowInstruction(
	op *lr.TokenValue,
	args []ast.Value,
) (
	ast.Instruction,
	error,
) {
	return &ast.Terminal{
		StartEndPos: parseutil.NewStartEndPos(op.Loc(), args[len(args)-1].End()),
		Kind:        ast.TerminalKind(op.Value),
		Srcs:        args,
	}, nil
}

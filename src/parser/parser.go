package parser

import (
	"io"

	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/ast"
	"github.com/pattyshack/chickadee/parser/lexer"
	"github.com/pattyshack/chickadee/parser/lr"
	"github.com/pattyshack/chickadee/parser/reducer"
)

type parser struct {
	lexer   lr.Lexer
	reducer reducer.Reducer
	emitter *parseutil.Emitter
}

func newParser(
	reader parseutil.BufferedByteLocationReader,
	emitter *parseutil.Emitter,
) *parser {
	return &parser{
		lexer:   lexer.NewLexer(reader),
		reducer: reducer.Reducer{},
		emitter: emitter,
	}
}

func (parser *parser) readLine() ([]lr.Token, error) {
	result := []lr.Token{}
	for {
		token, err := parser.lexer.Next()
		if err != nil {
			return result, err
		}

		if token.Id() == lr.NewlinesToken {
			if len(result) == 0 {
				continue
			}
			return result, nil
		}
		result = append(result, token)
	}
}

func (parser *parser) parse() []ast.SourceEntry {
	var currentFuncDef *ast.FuncDefinition
	var currentBlock *ast.Block
	result := []ast.SourceEntry{}
	for {
		segment, err := parser.readLine()
		if err != nil {
			if currentFuncDef != nil {
				parser.emitter.Emit(
					currentFuncDef.Loc(),
					"function definition not terminated by RBRACE")
			}

			if err != io.EOF {
				parser.emitter.EmitErrors(err)
			}

			return result
		}

		line, err := lr.Parse(
			parseutil.NewSubSegmentLexer(segment, parser.lexer.CurrentLocation()),
			parser.reducer)
		if err != nil {
			parser.emitter.EmitErrors(err)
			continue
		}

		switch stmt := line.(type) {
		case ast.SourceEntry:
			if currentFuncDef != nil {
				parser.emitter.Emit(
					currentFuncDef.Loc(),
					"function definition not terminated by RBRACE")
				currentFuncDef = nil
				currentBlock = nil
			}
			result = append(result, stmt)

			funcDef, ok := stmt.(*ast.FuncDefinition)
			if ok {
				currentFuncDef = funcDef
			}
		case lr.ParsedLocalLabel:
			if currentFuncDef == nil {
				parser.emitter.Emit(
					stmt.Loc(),
					"block label defined outside of function definition")
			} else {
				currentBlock = &ast.Block{
					StartEndPos: stmt.StartEndPos,
					Label:       stmt.Label,
				}
				currentFuncDef.Blocks = append(currentFuncDef.Blocks, currentBlock)
				currentFuncDef.EndPos = stmt.End()
			}
		case ast.Instruction:
			if currentFuncDef == nil {
				parser.emitter.Emit(
					stmt.Loc(),
					"instruction defined outside of function definition")
			} else {
				if currentBlock == nil {
					currentBlock = &ast.Block{
						StartEndPos: stmt.StartEnd(),
					}
					currentFuncDef.Blocks = append(currentFuncDef.Blocks, currentBlock)
				}
				currentBlock.Instructions = append(currentBlock.Instructions, stmt)
				currentBlock.EndPos = stmt.End()
				currentFuncDef.EndPos = stmt.End()

				_, ok := line.(ast.ControlFlowInstruction)
				if ok {
					currentBlock = nil
				}
			}
		case lr.ParsedRbrace:
			if currentFuncDef == nil {
				parser.emitter.Emit(
					stmt.Loc(),
					"RBRACE not part of function definition")
			} else {
				currentFuncDef.EndPos = stmt.End()
				currentFuncDef = nil
				currentBlock = nil
			}
		default:
			panic("unhandled line")
		}
	}
}

func Parse(
	reader parseutil.BufferedByteLocationReader,
	emitter *parseutil.Emitter,
) []ast.SourceEntry {
	parser := newParser(reader, emitter)
	return parser.parse()
}

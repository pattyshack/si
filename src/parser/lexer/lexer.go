package lexer

import (
	"github.com/pattyshack/gt/parseutil"

	"github.com/pattyshack/chickadee/parser/lr"
)

func NewLexer(
	reader parseutil.BufferedByteLocationReader,
) lr.Lexer {
	return parseutil.NewImplicitTerminalLexer(
		parseutil.NewMergeTokenCountLexer(
			parseutil.NewTrimTokenLexer(
				NewRawLexer(reader),
				lr.SpacesToken,
				lr.LineCommentToken,
				lr.BlockCommentToken),
			lr.NewlinesToken),
		lr.NewlinesToken,
		[]lr.SymbolId{
			lr.IdentifierToken,
			lr.IntegerLiteralToken, lr.FloatLiteralToken, lr.StringLiteralToken,
			lr.RparenToken,
			lr.LbraceToken, lr.RbraceToken,
		})
}

package lr

import (
	"github.com/pattyshack/gt/parseutil"
)

type Token = parseutil.Token[SymbolId]
type TokenValue = parseutil.TokenValue[SymbolId]
type TokenCount = parseutil.TokenCount[SymbolId]

type Lexer = parseutil.Lexer[Token]

const (
	SpacesToken       = SymbolId(' ')
	NewlinesToken     = SymbolId('\n')
	LineCommentToken  = SymbolId(-2)
	BlockCommentToken = SymbolId(-3)
	LexErrorToken     = SymbolId(-4)
)

// The following temporary structs are used only for parsing

type ParsedLocalLabel struct {
	parseutil.StartEndPos
	Label string
}

func (ParsedLocalLabel) IsLine() {}

type ParsedRbrace struct {
	parseutil.StartEndPos
}

func (ParsedRbrace) IsLine() {}

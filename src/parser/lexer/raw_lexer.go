package lexer

import (
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/pattyshack/gt/parseutil"
	"github.com/pattyshack/gt/stringutil"

	"github.com/pattyshack/chickadee/parser/lr"
)

const (
	initialPeekWindowSize = 64
)

var (
	keywords = map[string]lr.SymbolId{
		"define":  lr.DefineToken,
		"declare": lr.DeclareToken,
		"func":    lr.FuncToken,
	}
)

type RawLexer struct {
	parseutil.BufferedByteLocationReader
	*stringutil.InternPool
}

func NewRawLexer(
	reader parseutil.BufferedByteLocationReader,
) *RawLexer {
	return &RawLexer{
		BufferedByteLocationReader: reader,
		InternPool:                 stringutil.NewInternPool(),
	}
}

func (lexer *RawLexer) CurrentLocation() parseutil.Location {
	return lexer.Location
}

func (lexer *RawLexer) peekNextToken() (lr.SymbolId, string, error) {
	peeked, err := lexer.Peek(utf8.UTFMax)
	if len(peeked) > 0 && err == io.EOF {
		err = nil
	}
	if err != nil {
		return 0, "", err
	}

	char := peeked[0]

	if ('a' <= char && char <= 'z') ||
		('A' <= char && char <= 'Z') ||
		char == '_' {

		return lr.IdentifierToken, "", nil
	}

	if '0' <= char && char <= '9' || char == '-' {
		return lr.IntegerLiteralToken, "", nil
	}

	switch char {
	case ' ', '\t':
		return lr.SpacesToken, "", nil
	case '\r', '\n':
		return lr.NewlinesToken, "", nil
	case '.':
		return lr.FloatLiteralToken, "", nil
	case '"':
		return lr.StringLiteralToken, "", nil
	case '%':
		return lr.PercentToken, "%", nil
	case ',':
		return lr.CommaToken, ",", nil
	case '=':
		return lr.EqualToken, "=", nil
	case '@':
		return lr.AtToken, "@", nil
	case ':':
		return lr.ColonToken, ":", nil
	case '(':
		return lr.LparenToken, "(", nil
	case ')':
		return lr.RparenToken, ")", nil
	case '{':
		return lr.LbraceToken, "{", nil
	case '}':
		return lr.RbraceToken, "}", nil
	case '/':
		if len(peeked) > 1 {
			if peeked[1] == '/' {
				return lr.LineCommentToken, "", nil
			} else if peeked[1] == '*' {
				return lr.BlockCommentToken, "", nil
			}
		}
	}

	utf8Char, size := utf8.DecodeRune(peeked)
	if size == 1 || utf8Char == utf8.RuneError {
		return 0, "", parseutil.NewLocationError(
			lexer.Location,
			"unexpected utf8 rune")
	}

	return lr.IdentifierToken, "", nil
}

func (lexer *RawLexer) lexSpacesToken() (lr.Token, error) {
	token, err := parseutil.MaybeTokenizeSpaces(
		lexer.BufferedByteLocationReader,
		initialPeekWindowSize,
		lr.SpacesToken)
	if err != nil {
		return nil, err
	}

	if token == nil {
		panic("This should never happen")
	}

	return token, nil
}

func (lexer *RawLexer) lexNewlinesToken() (lr.Token, error) {
	token, foundInvalidNewline, err := parseutil.MaybeTokenizeNewlines(
		lexer.BufferedByteLocationReader,
		initialPeekWindowSize,
		lr.NewlinesToken)
	if err != nil {
		return nil, err
	}

	if token == nil {
		panic("This should never happen")
	}

	if foundInvalidNewline {
		return nil, parseutil.NewLocationError(
			token.StartPos,
			"unexpected utf8 rune")
	}

	return token, nil
}

func (lexer *RawLexer) lexLineCommentToken() (lr.Token, error) {
	token, err := parseutil.MaybeTokenizeLineComment(
		lexer.BufferedByteLocationReader,
		initialPeekWindowSize,
		lr.LineCommentToken,
		false) // preserve content
	if err != nil {
		return nil, err
	}

	if token == nil {
		panic("should never happend")
	}

	return token, nil
}

func (lexer *RawLexer) lexBlockCommentToken() (lr.Token, error) {
	token, notTerminated, err := parseutil.MaybeTokenizeBlockComment(
		lexer.BufferedByteLocationReader,
		true,
		initialPeekWindowSize,
		lr.BlockCommentToken,
		false) // preserve content
	if err != nil {
		return nil, err
	}

	if token == nil {
		panic("should never happend")
	}

	if notTerminated {
		return nil, parseutil.NewLocationError(
			token.StartPos,
			"block comment not terminated")
	}

	return token, nil
}

func (lexer *RawLexer) lexIntegerOrFloatLiteralToken() (lr.Token, error) {
	token, hasNoDigits, err := parseutil.MaybeTokenizeIntegerOrFloatLiteral(
		lexer.BufferedByteLocationReader,
		initialPeekWindowSize,
		lexer.InternPool,
		lr.IntegerLiteralToken,
		lr.FloatLiteralToken)
	if err != nil {
		return nil, err
	}

	if token == nil {
		panic("should never happen")
	}

	if hasNoDigits {
		return nil, parseutil.NewLocationError(
			token.StartPos,
			"%s has no digits",
			token.SubType)
	}

	return token, nil
}

func (lexer *RawLexer) lexStringLiteralToken(
	subType parseutil.LiteralSubType,
	useBacktickMarker bool,
) (
	lr.Token,
	error,
) {
	token, errMsg, err := parseutil.MaybeTokenizeStringLiteral(
		lexer.BufferedByteLocationReader,
		initialPeekWindowSize,
		lexer.InternPool,
		lr.StringLiteralToken,
		subType,
		useBacktickMarker)
	if err != nil {
		return nil, err
	}

	if token == nil {
		panic("should never happen")
	}

	if errMsg != "" {
		return nil, parseutil.NewLocationError(token.StartPos, errMsg)
	}

	return token, nil
}

func (lexer *RawLexer) lexIdentifierOrKeywords() (
	lr.Token,
	error,
) {
	token, err := parseutil.MaybeTokenizeIdentifier(
		lexer.BufferedByteLocationReader,
		initialPeekWindowSize,
		lexer.InternPool,
		lr.IdentifierToken)
	if err != nil {
		return nil, err
	}

	if token == nil {
		panic("Should never hapapen")
	}

	kwSymbolId, ok := keywords[token.Value]
	if ok {
		token.SymbolId = kwSymbolId
	}

	return token, nil
}

func (lexer *RawLexer) Next() (lr.Token, error) {
	symbolId, value, err := lexer.peekNextToken()
	if err != nil {
		return nil, err
	}

	// fixed length token
	size := len(value)
	if size > 0 {
		loc := lexer.Location

		_, err := lexer.Discard(size)
		if err != nil {
			panic("should never happen")
		}

		return &lr.TokenValue{
			SymbolId:    symbolId,
			StartEndPos: parseutil.NewStartEndPos(loc, lexer.Location),
			Value:       value,
		}, nil
	}

	// variable length token
	switch symbolId {
	case lr.SpacesToken:
		return lexer.lexSpacesToken()
	case lr.NewlinesToken:
		return lexer.lexNewlinesToken()
	case lr.LineCommentToken:
		return lexer.lexLineCommentToken()
	case lr.BlockCommentToken:
		return lexer.lexBlockCommentToken()
	case lr.IntegerLiteralToken:
		return lexer.lexIntegerOrFloatLiteralToken()
	case lr.FloatLiteralToken:
		return lexer.lexIntegerOrFloatLiteralToken()
	case lr.StringLiteralToken:
		return lexer.lexStringLiteralToken(
			parseutil.SingleLineString, // XXX: maybe allow multiline string for data?
			false)
	case lr.IdentifierToken:
		return lexer.lexIdentifierOrKeywords()
	}

	panic(fmt.Sprintf("unhandled variable length token: %v", symbolId))
}

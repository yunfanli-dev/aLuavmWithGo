package lexer

import (
	"fmt"
	"unicode"
)

// Scanner tokenizes Lua 5.1 subset source code into a flat token stream.
type Scanner struct {
	source string
	input  []rune
	index  int
	line   int
	column int
}

// NewScanner creates a scanner for the provided Lua source payload.
func NewScanner(sourceName, input string) *Scanner {
	return &Scanner{
		source: sourceName,
		input:  []rune(input),
		line:   1,
		column: 1,
	}
}

// ScanAll consumes the entire source and returns all lexical tokens ending with EOF.
func (s *Scanner) ScanAll() ([]Token, error) {
	tokens := make([]Token, 0)
	for {
		token, err := s.NextToken()
		if err != nil {
			return nil, err
		}

		tokens = append(tokens, token)
		if token.Type == TokenEOF {
			return tokens, nil
		}
	}
}

// NextToken returns the next lexical token from the source stream.
func (s *Scanner) NextToken() (Token, error) {
	if err := s.skipWhitespaceAndComments(); err != nil {
		return Token{}, err
	}

	start := s.position()
	ch, ok := s.peek()
	if !ok {
		return Token{Type: TokenEOF, Start: start, End: start}, nil
	}

	if isIdentifierStart(ch) {
		return s.scanIdentifier()
	}

	if isDigit(ch) {
		return s.scanNumber()
	}

	switch ch {
	case '\'', '"':
		return s.scanString()
	case '[':
		if s.peekNext() == '[' || s.peekNext() == '=' {
			return s.scanLongString()
		}

		return s.singleToken(TokenLeftBracket), nil
	case '=':
		return s.scanFixedOrDouble(TokenAssign, TokenEqual, '=')
	case '~':
		return s.scanRequiredPair(TokenNotEqual, '=', "unexpected '~', Lua 5.1 only accepts '~='")
	case '<':
		return s.scanFixedOrDouble(TokenLess, TokenLessEqual, '=')
	case '>':
		return s.scanFixedOrDouble(TokenGreater, TokenGreaterEqual, '=')
	case '+':
		return s.singleToken(TokenPlus), nil
	case '-':
		return s.singleToken(TokenMinus), nil
	case '*':
		return s.singleToken(TokenStar), nil
	case '/':
		return s.singleToken(TokenSlash), nil
	case '%':
		return s.singleToken(TokenPercent), nil
	case '^':
		return s.singleToken(TokenCaret), nil
	case '#':
		return s.singleToken(TokenHash), nil
	case '(':
		return s.singleToken(TokenLeftParen), nil
	case ')':
		return s.singleToken(TokenRightParen), nil
	case '{':
		return s.singleToken(TokenLeftBrace), nil
	case '}':
		return s.singleToken(TokenRightBrace), nil
	case ']':
		return s.singleToken(TokenRightBracket), nil
	case ';':
		return s.singleToken(TokenSemicolon), nil
	case ':':
		return s.singleToken(TokenColon), nil
	case ',':
		return s.singleToken(TokenComma), nil
	case '.':
		return s.scanDots()
	default:
		return Token{}, s.errorAt(start, fmt.Sprintf("unexpected character %q", ch))
	}
}

// Tokenize is a convenience helper for scanning a complete Lua source payload.
func Tokenize(sourceName, input string) ([]Token, error) {
	return NewScanner(sourceName, input).ScanAll()
}

func (s *Scanner) scanIdentifier() (Token, error) {
	start := s.position()
	literal := s.consumeWhile(func(ch rune) bool {
		return isIdentifierPart(ch)
	})

	tokenType := TokenIdentifier
	if keywordType, ok := keywords[literal]; ok {
		tokenType = keywordType
	}

	return Token{
		Type:    tokenType,
		Literal: literal,
		Start:   start,
		End:     s.position(),
	}, nil
}

func (s *Scanner) scanNumber() (Token, error) {
	start := s.position()
	if s.peekNext() == 'x' || s.peekNext() == 'X' {
		return s.scanHexNumber(start)
	}

	literal := s.consumeWhile(func(ch rune) bool {
		return isDigit(ch)
	})

	if s.match('.') {
		next, ok := s.peek()
		if ok && isDigit(next) {
			literal += "."
			literal += s.consumeWhile(func(ch rune) bool {
				return isDigit(ch)
			})
		} else {
			s.backtrack()
		}
	}

	exponent, err := s.scanExponentPart(start)
	if err != nil {
		return Token{}, err
	}

	literal += exponent

	// TODO: Extend number scanning with Lua 5.1 hexadecimal literal support.
	return Token{
		Type:    TokenNumber,
		Literal: literal,
		Start:   start,
		End:     s.position(),
	}, nil
}

// scanHexNumber parses Lua 5.1 integer hexadecimal literals like `0xff`.
func (s *Scanner) scanHexNumber(start Position) (Token, error) {
	literal := s.consumeWhile(func(ch rune) bool {
		return isDigit(ch)
	})

	marker, _ := s.advance()
	literal += string(marker)

	if next, ok := s.peek(); !ok || !isHexDigit(next) {
		return Token{}, s.errorAt(start, "malformed hexadecimal literal")
	}

	literal += s.consumeWhile(func(ch rune) bool {
		return isHexDigit(ch)
	})

	return Token{
		Type:    TokenNumber,
		Literal: literal,
		Start:   start,
		End:     s.position(),
	}, nil
}

// scanExponentPart parses the optional exponent suffix in Lua decimal literals like `1e3` or `1.5e-2`.
func (s *Scanner) scanExponentPart(start Position) (string, error) {
	ch, ok := s.peek()
	if !ok || (ch != 'e' && ch != 'E') {
		return "", nil
	}

	s.advance()
	literal := string(ch)

	if sign, ok := s.peek(); ok && (sign == '+' || sign == '-') {
		s.advance()
		literal += string(sign)
	}

	if next, ok := s.peek(); !ok || !isDigit(next) {
		return "", s.errorAt(start, "malformed exponent literal")
	}

	literal += s.consumeWhile(func(ch rune) bool {
		return isDigit(ch)
	})

	return literal, nil
}

func (s *Scanner) scanString() (Token, error) {
	start := s.position()
	quote, _ := s.advance()
	value := make([]rune, 0)

	for {
		ch, ok := s.advance()
		if !ok {
			return Token{}, s.errorAt(start, "unterminated string literal")
		}

		if ch == '\n' {
			return Token{}, s.errorAt(start, "newline in string literal")
		}

		if ch == quote {
			return Token{
				Type:    TokenString,
				Literal: string(value),
				Start:   start,
				End:     s.position(),
			}, nil
		}

		if ch == '\\' {
			escaped, err := s.scanEscape(start)
			if err != nil {
				return Token{}, err
			}

			value = append(value, escaped)
			continue
		}

		value = append(value, ch)
	}
}

// scanLongString parses Lua 5.1 long-bracket strings like `[[...]]` and `[=[...]=]`.
func (s *Scanner) scanLongString() (Token, error) {
	start := s.position()
	level, err := s.expectLongBracketStart(start)
	if err != nil {
		return Token{}, err
	}

	value, end, err := s.scanLongBracketBody(start, level)
	if err != nil {
		return Token{}, err
	}

	return Token{
		Type:    TokenString,
		Literal: value,
		Start:   start,
		End:     end,
	}, nil
}

func (s *Scanner) scanEscape(start Position) (rune, error) {
	ch, ok := s.advance()
	if !ok {
		return 0, s.errorAt(start, "unterminated escape sequence")
	}

	switch ch {
	case 'a':
		return '\a', nil
	case 'b':
		return '\b', nil
	case 'f':
		return '\f', nil
	case 'n':
		return '\n', nil
	case 'r':
		return '\r', nil
	case 't':
		return '\t', nil
	case 'v':
		return '\v', nil
	case '\\':
		return '\\', nil
	case '"':
		return '"', nil
	case '\'':
		return '\'', nil
	default:
		// TODO: Extend escape handling with Lua 5.1 numeric escapes.
		return 0, s.errorAt(start, fmt.Sprintf("unsupported escape sequence \\%c", ch))
	}
}

func (s *Scanner) scanDots() (Token, error) {
	start := s.position()
	s.advance()

	if s.match('.') {
		if s.match('.') {
			return Token{Type: TokenVararg, Literal: "...", Start: start, End: s.position()}, nil
		}

		return Token{Type: TokenConcat, Literal: "..", Start: start, End: s.position()}, nil
	}

	return Token{Type: TokenDot, Literal: ".", Start: start, End: s.position()}, nil
}

func (s *Scanner) scanFixedOrDouble(single TokenType, pair TokenType, pairRune rune) (Token, error) {
	start := s.position()
	literalRune, _ := s.advance()
	literal := string(literalRune)

	if s.match(pairRune) {
		return Token{Type: pair, Literal: literal + string(pairRune), Start: start, End: s.position()}, nil
	}

	return Token{Type: single, Literal: literal, Start: start, End: s.position()}, nil
}

func (s *Scanner) scanRequiredPair(tokenType TokenType, expected rune, msg string) (Token, error) {
	start := s.position()
	first, _ := s.advance()
	if !s.match(expected) {
		return Token{}, s.errorAt(start, msg)
	}

	return Token{
		Type:    tokenType,
		Literal: string(first) + string(expected),
		Start:   start,
		End:     s.position(),
	}, nil
}

func (s *Scanner) singleToken(tokenType TokenType) Token {
	start := s.position()
	ch, _ := s.advance()

	return Token{
		Type:    tokenType,
		Literal: string(ch),
		Start:   start,
		End:     s.position(),
	}
}

func (s *Scanner) skipWhitespaceAndComments() error {
	for {
		ch, ok := s.peek()
		if !ok {
			return nil
		}

		if unicode.IsSpace(ch) {
			s.advance()
			continue
		}

		if ch == '-' && s.peekNext() == '-' {
			s.advance()
			s.advance()

			next, ok := s.peek()
			if ok && next == '[' {
				commentStart := s.position()
				level, err := s.expectLongBracketStart(commentStart)
				if err == nil {
					if _, _, err := s.scanLongBracketBody(commentStart, level); err != nil {
						return err
					}

					continue
				}
			}

			for {
				commentRune, commentOK := s.peek()
				if !commentOK || commentRune == '\n' {
					break
				}

				s.advance()
			}

			continue
		}

		return nil
	}
}

func (s *Scanner) expectLongBracketStart(start Position) (int, error) {
	if !s.match('[') {
		return 0, s.errorAt(start, "expected long bracket start")
	}

	level := 0
	for s.match('=') {
		level++
	}

	if !s.match('[') {
		return 0, s.errorAt(start, "expected long bracket start")
	}

	if ch, ok := s.peek(); ok && ch == '\n' {
		s.advance()
	}

	return level, nil
}

func (s *Scanner) scanLongBracketBody(start Position, level int) (string, Position, error) {
	value := make([]rune, 0)
	for {
		ch, ok := s.peek()
		if !ok {
			return "", Position{}, s.errorAt(start, "unterminated long string")
		}

		if ch == ']' {
			matched, end := s.matchLongBracketClose(level)
			if matched {
				return string(value), end, nil
			}
		}

		s.advance()
		value = append(value, ch)
	}
}

func (s *Scanner) matchLongBracketClose(level int) (bool, Position) {
	startIndex := s.index
	startLine := s.line
	startColumn := s.column

	if !s.match(']') {
		return false, Position{}
	}

	for index := 0; index < level; index++ {
		if !s.match('=') {
			s.index = startIndex
			s.line = startLine
			s.column = startColumn
			return false, Position{}
		}
	}

	if !s.match(']') {
		s.index = startIndex
		s.line = startLine
		s.column = startColumn
		return false, Position{}
	}

	return true, s.position()
}

func (s *Scanner) consumeWhile(fn func(rune) bool) string {
	runes := make([]rune, 0)
	for {
		ch, ok := s.peek()
		if !ok || !fn(ch) {
			return string(runes)
		}

		runes = append(runes, ch)
		s.advance()
	}
}

func (s *Scanner) match(target rune) bool {
	ch, ok := s.peek()
	if !ok || ch != target {
		return false
	}

	s.advance()
	return true
}

func (s *Scanner) backtrack() {
	if s.index == 0 {
		return
	}

	s.index--
	s.column--
}

func (s *Scanner) peek() (rune, bool) {
	if s.index >= len(s.input) {
		return 0, false
	}

	return s.input[s.index], true
}

func (s *Scanner) peekNext() rune {
	return s.peekNextN(1)
}

func (s *Scanner) peekNextN(offset int) rune {
	nextIndex := s.index + offset
	if nextIndex >= len(s.input) {
		return 0
	}

	return s.input[nextIndex]
}

func (s *Scanner) advance() (rune, bool) {
	ch, ok := s.peek()
	if !ok {
		return 0, false
	}

	s.index++
	if ch == '\n' {
		s.line++
		s.column = 1
	} else {
		s.column++
	}

	return ch, true
}

func (s *Scanner) position() Position {
	return Position{
		Offset: s.index,
		Line:   s.line,
		Column: s.column,
	}
}

func (s *Scanner) errorAt(pos Position, msg string) error {
	return &Error{
		Source: s.source,
		Pos:    pos,
		Msg:    msg,
	}
}

func isIdentifierStart(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch)
}

func isIdentifierPart(ch rune) bool {
	return isIdentifierStart(ch) || isDigit(ch)
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isHexDigit(ch rune) bool {
	return isDigit(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

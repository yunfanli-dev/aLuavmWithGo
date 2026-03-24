package lexer

import (
	"fmt"
	"unicode"
)

// Scanner 负责把 Lua 5.1 子集源码切分成线性的 token 流。
// 它维护当前位置、行列号和 rune 级输入，以便支持更准确的错误定位。
type Scanner struct {
	source string
	input  []rune
	index  int
	line   int
	column int
}

// NewScanner 基于给定源码名称和内容创建扫描器。
// 初始化后行列号从 1 开始计数，便于与常见编辑器的定位习惯保持一致。
func NewScanner(sourceName, input string) *Scanner {
	return &Scanner{
		source: sourceName,
		input:  []rune(input),
		line:   1,
		column: 1,
	}
}

// ScanAll 持续扫描直到读完整份源码，并返回包含 EOF 在内的完整 token 列表。
// 调用方通常在 parser 之前使用它一次性拿到完整词法结果。
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

// NextToken 从当前游标位置读取下一个 token。
// 它会先跳过空白和注释，再根据当前字符分派到具体的扫描逻辑。
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

// Tokenize 是面向外部调用的便捷函数。
// 它会创建扫描器并一次性完成整份源码的扫描。
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

	// TODO: 后续把数字扫描进一步扩展到 Lua 5.1 更完整的十六进制数字能力，
	// 当前仍以最小可用子集为主。
	return Token{
		Type:    TokenNumber,
		Literal: literal,
		Start:   start,
		End:     s.position(),
	}, nil
}

// scanHexNumber 解析 Lua 5.1 子集里的整数十六进制字面量，例如 `0xff`。
// 当前只覆盖最小整数形式，不处理更复杂的十六进制浮点等扩展语义。
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

// scanExponentPart 解析十进制数字后面可选的指数部分。
// 例如 `1e3`、`1.5e-2` 这类字面量都会经过这里补全指数后缀。
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

// scanLongString 解析 Lua 5.1 的长括号字符串。
// 当前支持 `[[...]]`、`[=[...]=]` 这类最常见的长字符串形式。
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
		// TODO: 后续补齐 Lua 5.1 更完整的转义序列处理，
		// 例如数字转义等目前尚未支持的分支。
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

package lexer

// TokenType identifies the lexical category produced by the Lua 5.1 subset scanner.
type TokenType string

const (
	// TokenEOF marks the logical end of the source stream.
	TokenEOF TokenType = "eof"
	// TokenIdentifier marks a Lua identifier.
	TokenIdentifier TokenType = "identifier"
	// TokenNumber marks a numeric literal.
	TokenNumber TokenType = "number"
	// TokenString marks a quoted string literal.
	TokenString TokenType = "string"

	// Keywords.
	TokenAnd      TokenType = "and"
	TokenBreak    TokenType = "break"
	TokenDo       TokenType = "do"
	TokenElse     TokenType = "else"
	TokenElseIf   TokenType = "elseif"
	TokenEnd      TokenType = "end"
	TokenFalse    TokenType = "false"
	TokenFor      TokenType = "for"
	TokenFunction TokenType = "function"
	TokenIf       TokenType = "if"
	TokenIn       TokenType = "in"
	TokenLocal    TokenType = "local"
	TokenNil      TokenType = "nil"
	TokenNot      TokenType = "not"
	TokenOr       TokenType = "or"
	TokenRepeat   TokenType = "repeat"
	TokenReturn   TokenType = "return"
	TokenThen     TokenType = "then"
	TokenTrue     TokenType = "true"
	TokenUntil    TokenType = "until"
	TokenWhile    TokenType = "while"

	// Punctuation and operators.
	TokenAssign       TokenType = "="
	TokenPlus         TokenType = "+"
	TokenMinus        TokenType = "-"
	TokenStar         TokenType = "*"
	TokenSlash        TokenType = "/"
	TokenPercent      TokenType = "%"
	TokenCaret        TokenType = "^"
	TokenHash         TokenType = "#"
	TokenEqual        TokenType = "=="
	TokenNotEqual     TokenType = "~="
	TokenLess         TokenType = "<"
	TokenLessEqual    TokenType = "<="
	TokenGreater      TokenType = ">"
	TokenGreaterEqual TokenType = ">="
	TokenLeftParen    TokenType = "("
	TokenRightParen   TokenType = ")"
	TokenLeftBrace    TokenType = "{"
	TokenRightBrace   TokenType = "}"
	TokenLeftBracket  TokenType = "["
	TokenRightBracket TokenType = "]"
	TokenSemicolon    TokenType = ";"
	TokenColon        TokenType = ":"
	TokenComma        TokenType = ","
	TokenDot          TokenType = "."
	TokenConcat       TokenType = ".."
	TokenVararg       TokenType = "..."
)

var keywords = map[string]TokenType{
	"and":      TokenAnd,
	"break":    TokenBreak,
	"do":       TokenDo,
	"else":     TokenElse,
	"elseif":   TokenElseIf,
	"end":      TokenEnd,
	"false":    TokenFalse,
	"for":      TokenFor,
	"function": TokenFunction,
	"if":       TokenIf,
	"in":       TokenIn,
	"local":    TokenLocal,
	"nil":      TokenNil,
	"not":      TokenNot,
	"or":       TokenOr,
	"repeat":   TokenRepeat,
	"return":   TokenReturn,
	"then":     TokenThen,
	"true":     TokenTrue,
	"until":    TokenUntil,
	"while":    TokenWhile,
}

// Position describes a token location in the original source stream.
type Position struct {
	Offset int
	Line   int
	Column int
}

// Token holds a lexical unit produced by the scanner.
type Token struct {
	Type    TokenType
	Literal string
	Start   Position
	End     Position
}

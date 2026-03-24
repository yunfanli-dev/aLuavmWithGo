package lexer

// TokenType 表示扫描器产出的词法单元类别。
// 解析器会根据它判断当前 token 在 Lua 5.1 子集语法中的角色。
type TokenType string

const (
	// TokenEOF 表示源码流已经到达逻辑结尾。
	TokenEOF TokenType = "eof"
	// TokenIdentifier 表示 Lua 标识符。
	TokenIdentifier TokenType = "identifier"
	// TokenNumber 表示数值字面量。
	TokenNumber TokenType = "number"
	// TokenString 表示字符串字面量。
	TokenString TokenType = "string"

	// 关键字。
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

	// 标点和运算符。
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

// Position 描述 token 在原始源码中的位置。
// Offset 是整体偏移量，Line 和 Column 则用于面向用户的报错信息。
type Position struct {
	Offset int
	Line   int
	Column int
}

// Token 表示扫描器产出的一项词法单元。
// 除了类别和字面量之外，还会保留起止位置，便于后续解析和错误定位。
type Token struct {
	Type    TokenType
	Literal string
	Start   Position
	End     Position
}

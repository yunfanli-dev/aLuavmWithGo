package parser

import "github.com/yunfanli-dev/aLuavmWithGo/internal/lexer"

// Node is the common AST contract shared by all parsed syntax nodes.
type Node interface {
	node()
	Span() Span
}

// Statement marks AST nodes that can appear in a chunk body.
type Statement interface {
	Node
	statement()
}

// Expression marks AST nodes that can appear in expression positions.
type Expression interface {
	Node
	expression()
}

// Span tracks the source range occupied by an AST node.
type Span struct {
	Start lexer.Position
	End   lexer.Position
}

// Chunk is the root AST node for a parsed Lua source file.
type Chunk struct {
	Statements []Statement
	span       Span
}

// LocalAssignStatement represents `local name = expr` and `local a, b = ...`.
type LocalAssignStatement struct {
	Names  []Identifier
	Values []Expression
	span   Span
}

// ReturnStatement represents a Lua `return` statement.
type ReturnStatement struct {
	Values []Expression
	span   Span
}

// Identifier represents a named variable reference.
type Identifier struct {
	Name string
	span Span
}

// NilExpression represents the Lua `nil` literal.
type NilExpression struct {
	span Span
}

// BooleanExpression represents `true` or `false`.
type BooleanExpression struct {
	Value bool
	span  Span
}

// NumberExpression represents a numeric literal.
type NumberExpression struct {
	Literal string
	span    Span
}

// StringExpression represents a quoted string literal.
type StringExpression struct {
	Value string
	span  Span
}

// UnaryExpression represents a unary operator expression.
type UnaryExpression struct {
	Operator lexer.TokenType
	Operand  Expression
	span     Span
}

// BinaryExpression represents a binary operator expression.
type BinaryExpression struct {
	Left     Expression
	Operator lexer.TokenType
	Right    Expression
	span     Span
}

func (*Chunk) node()                {}
func (*LocalAssignStatement) node() {}
func (*ReturnStatement) node()      {}
func (*Identifier) node()           {}
func (*NilExpression) node()        {}
func (*BooleanExpression) node()    {}
func (*NumberExpression) node()     {}
func (*StringExpression) node()     {}
func (*UnaryExpression) node()      {}
func (*BinaryExpression) node()     {}

func (*LocalAssignStatement) statement() {}
func (*ReturnStatement) statement()      {}

func (*Identifier) expression()        {}
func (*NilExpression) expression()     {}
func (*BooleanExpression) expression() {}
func (*NumberExpression) expression()  {}
func (*StringExpression) expression()  {}
func (*UnaryExpression) expression()   {}
func (*BinaryExpression) expression()  {}

// Span reports the source range for a chunk node.
func (c *Chunk) Span() Span { return c.span }

// Span reports the source range for a local assignment statement.
func (s *LocalAssignStatement) Span() Span { return s.span }

// Span reports the source range for a return statement.
func (s *ReturnStatement) Span() Span { return s.span }

// Span reports the source range for an identifier expression.
func (e *Identifier) Span() Span { return e.span }

// Span reports the source range for a nil expression.
func (e *NilExpression) Span() Span { return e.span }

// Span reports the source range for a boolean expression.
func (e *BooleanExpression) Span() Span { return e.span }

// Span reports the source range for a number expression.
func (e *NumberExpression) Span() Span { return e.span }

// Span reports the source range for a string expression.
func (e *StringExpression) Span() Span { return e.span }

// Span reports the source range for a unary expression.
func (e *UnaryExpression) Span() Span { return e.span }

// Span reports the source range for a binary expression.
func (e *BinaryExpression) Span() Span { return e.span }

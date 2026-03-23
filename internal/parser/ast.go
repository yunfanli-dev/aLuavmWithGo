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

// CallStatement represents a function call used as a statement.
type CallStatement struct {
	Call *CallExpression
	span Span
}

// AssignStatement represents `name = expr` and `a, b = ...`.
type AssignStatement struct {
	Targets []Expression
	Values  []Expression
	span    Span
}

// FunctionDeclarationStatement represents `function name(args) ... end`.
type FunctionDeclarationStatement struct {
	Name       Identifier
	Parameters []Identifier
	IsVararg   bool
	Body       []Statement
	span       Span
}

// LocalFunctionDeclarationStatement represents `local function name(args) ... end`.
type LocalFunctionDeclarationStatement struct {
	Name       Identifier
	Parameters []Identifier
	IsVararg   bool
	Body       []Statement
	span       Span
}

// LocalAssignStatement represents `local name = expr` and `local a, b = ...`.
type LocalAssignStatement struct {
	Names  []Identifier
	Values []Expression
	span   Span
}

// DoStatement represents `do ... end`.
type DoStatement struct {
	Body []Statement
	span Span
}

// BreakStatement represents `break`.
type BreakStatement struct {
	span Span
}

// IfClause represents one condition/body branch of an if statement.
type IfClause struct {
	Condition Expression
	Body      []Statement
	span      Span
}

// IfStatement represents `if` / `elseif` / `else`.
type IfStatement struct {
	Clauses  []IfClause
	ElseBody []Statement
	span     Span
}

// WhileStatement represents a Lua `while` loop.
type WhileStatement struct {
	Condition Expression
	Body      []Statement
	span      Span
}

// RepeatStatement represents a Lua `repeat ... until` loop.
type RepeatStatement struct {
	Body      []Statement
	Condition Expression
	span      Span
}

// NumericForStatement represents a Lua numeric for-loop.
type NumericForStatement struct {
	Name  Identifier
	Start Expression
	Limit Expression
	Step  Expression
	Body  []Statement
	span  Span
}

// GenericForStatement represents a Lua generic for-in loop.
type GenericForStatement struct {
	Names     []Identifier
	Iterators []Expression
	Body      []Statement
	span      Span
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

// CallExpression represents `callee(args)`.
type CallExpression struct {
	Callee    Expression
	Receiver  Expression
	Method    string
	Arguments []Expression
	span      Span
}

// FunctionExpression represents `function(args) ... end`.
type FunctionExpression struct {
	Parameters []Identifier
	IsVararg   bool
	Body       []Statement
	span       Span
}

// VarargExpression represents the Lua `...` expression.
type VarargExpression struct {
	span Span
}

// ParenthesizedExpression represents `(expr)` and preserves grouping semantics.
type ParenthesizedExpression struct {
	Inner Expression
	span  Span
}

// IndexExpression represents `target[index]` and `target.name`.
type IndexExpression struct {
	Target Expression
	Index  Expression
	span   Span
}

// TableField represents one field inside a table constructor.
type TableField struct {
	Key         Expression
	Value       Expression
	IsListField bool
	span        Span
}

// TableConstructorExpression represents `{ ... }`.
type TableConstructorExpression struct {
	Fields []TableField
	span   Span
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

func (*Chunk) node()                             {}
func (*CallStatement) node()                     {}
func (*AssignStatement) node()                   {}
func (*FunctionDeclarationStatement) node()      {}
func (*LocalFunctionDeclarationStatement) node() {}
func (*LocalAssignStatement) node()              {}
func (*DoStatement) node()                       {}
func (*BreakStatement) node()                    {}
func (*IfStatement) node()                       {}
func (*WhileStatement) node()                    {}
func (*RepeatStatement) node()                   {}
func (*NumericForStatement) node()               {}
func (*GenericForStatement) node()               {}
func (*ReturnStatement) node()                   {}
func (*Identifier) node()                        {}
func (*CallExpression) node()                    {}
func (*FunctionExpression) node()                {}
func (*IndexExpression) node()                   {}
func (*TableConstructorExpression) node()        {}
func (*VarargExpression) node()                  {}
func (*ParenthesizedExpression) node()           {}
func (*NilExpression) node()                     {}
func (*BooleanExpression) node()                 {}
func (*NumberExpression) node()                  {}
func (*StringExpression) node()                  {}
func (*UnaryExpression) node()                   {}
func (*BinaryExpression) node()                  {}

func (*CallStatement) statement()                     {}
func (*AssignStatement) statement()                   {}
func (*FunctionDeclarationStatement) statement()      {}
func (*LocalFunctionDeclarationStatement) statement() {}
func (*LocalAssignStatement) statement()              {}
func (*DoStatement) statement()                       {}
func (*BreakStatement) statement()                    {}
func (*IfStatement) statement()                       {}
func (*WhileStatement) statement()                    {}
func (*RepeatStatement) statement()                   {}
func (*NumericForStatement) statement()               {}
func (*GenericForStatement) statement()               {}
func (*ReturnStatement) statement()                   {}

func (*Identifier) expression()                 {}
func (*CallExpression) expression()             {}
func (*FunctionExpression) expression()         {}
func (*IndexExpression) expression()            {}
func (*TableConstructorExpression) expression() {}
func (*VarargExpression) expression()           {}
func (*ParenthesizedExpression) expression()    {}
func (*NilExpression) expression()              {}
func (*BooleanExpression) expression()          {}
func (*NumberExpression) expression()           {}
func (*StringExpression) expression()           {}
func (*UnaryExpression) expression()            {}
func (*BinaryExpression) expression()           {}

// Span reports the source range for a chunk node.
func (c *Chunk) Span() Span { return c.span }

// Span reports the source range for a call statement.
func (s *CallStatement) Span() Span { return s.span }

// Span reports the source range for an assignment statement.
func (s *AssignStatement) Span() Span { return s.span }

// Span reports the source range for a function declaration statement.
func (s *FunctionDeclarationStatement) Span() Span { return s.span }

// Span reports the source range for a local function declaration statement.
func (s *LocalFunctionDeclarationStatement) Span() Span { return s.span }

// Span reports the source range for a local assignment statement.
func (s *LocalAssignStatement) Span() Span { return s.span }

// Span reports the source range for a do statement.
func (s *DoStatement) Span() Span { return s.span }

// Span reports the source range for a break statement.
func (s *BreakStatement) Span() Span { return s.span }

// Span reports the source range for an if clause.
func (c *IfClause) Span() Span { return c.span }

// Span reports the source range for an if statement.
func (s *IfStatement) Span() Span { return s.span }

// Span reports the source range for a while statement.
func (s *WhileStatement) Span() Span { return s.span }

// Span reports the source range for a repeat statement.
func (s *RepeatStatement) Span() Span { return s.span }

// Span reports the source range for a numeric for statement.
func (s *NumericForStatement) Span() Span { return s.span }

// Span reports the source range for a generic for statement.
func (s *GenericForStatement) Span() Span { return s.span }

// Span reports the source range for a return statement.
func (s *ReturnStatement) Span() Span { return s.span }

// Span reports the source range for an identifier expression.
func (e *Identifier) Span() Span { return e.span }

// Span reports the source range for a call expression.
func (e *CallExpression) Span() Span { return e.span }

// Span reports the source range for a function expression.
func (e *FunctionExpression) Span() Span { return e.span }

// Span reports the source range for a vararg expression.
func (e *VarargExpression) Span() Span { return e.span }

// Span reports the source range for a parenthesized expression.
func (e *ParenthesizedExpression) Span() Span { return e.span }

// Span reports the source range for an index expression.
func (e *IndexExpression) Span() Span { return e.span }

// Span reports the source range for a table field.
func (f *TableField) Span() Span { return f.span }

// Span reports the source range for a table constructor expression.
func (e *TableConstructorExpression) Span() Span { return e.span }

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

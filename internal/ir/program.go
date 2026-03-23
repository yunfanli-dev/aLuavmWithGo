package ir

// Node is the common IR contract shared by all intermediate representation nodes.
type Node interface {
	node()
}

// Statement marks IR nodes that can appear in a program body.
type Statement interface {
	Node
	statement()
}

// Expression marks IR nodes that can appear in expression positions.
type Expression interface {
	Node
	expression()
}

// Program is the compiled top-level IR unit produced from a parsed Lua chunk.
type Program struct {
	Statements []Statement
}

// LocalAssignStatement represents an IR local assignment statement.
type LocalAssignStatement struct {
	Names  []string
	Values []Expression
}

// ReturnStatement represents an IR return statement.
type ReturnStatement struct {
	Values []Expression
}

// IdentifierExpression represents an IR identifier reference.
type IdentifierExpression struct {
	Name string
}

// NilExpression represents the IR nil literal.
type NilExpression struct{}

// BooleanExpression represents the IR boolean literal.
type BooleanExpression struct {
	Value bool
}

// NumberExpression represents the IR number literal.
type NumberExpression struct {
	Literal string
}

// StringExpression represents the IR string literal.
type StringExpression struct {
	Value string
}

// UnaryExpression represents an IR unary operation.
type UnaryExpression struct {
	Operator string
	Operand  Expression
}

// BinaryExpression represents an IR binary operation.
type BinaryExpression struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (*Program) node()                    {}
func (*LocalAssignStatement) node()       {}
func (*ReturnStatement) node()            {}
func (*IdentifierExpression) node()       {}
func (*NilExpression) node()              {}
func (*BooleanExpression) node()          {}
func (*NumberExpression) node()           {}
func (*StringExpression) node()           {}
func (*UnaryExpression) node()            {}
func (*BinaryExpression) node()           {}
func (*LocalAssignStatement) statement()  {}
func (*ReturnStatement) statement()       {}
func (*IdentifierExpression) expression() {}
func (*NilExpression) expression()        {}
func (*BooleanExpression) expression()    {}
func (*NumberExpression) expression()     {}
func (*StringExpression) expression()     {}
func (*UnaryExpression) expression()      {}
func (*BinaryExpression) expression()     {}

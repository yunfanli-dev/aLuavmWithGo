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

// CallStatement represents a function call used as a statement.
type CallStatement struct {
	Call *CallExpression
}

// AssignStatement represents an IR assignment statement.
type AssignStatement struct {
	Names  []string
	Values []Expression
}

// FunctionDeclarationStatement represents an IR named function declaration.
type FunctionDeclarationStatement struct {
	Name       string
	Parameters []string
	Body       []Statement
}

// LocalAssignStatement represents an IR local assignment statement.
type LocalAssignStatement struct {
	Names  []string
	Values []Expression
}

// IfClause represents one IR conditional branch.
type IfClause struct {
	Condition Expression
	Body      []Statement
}

// IfStatement represents an IR if statement.
type IfStatement struct {
	Clauses  []IfClause
	ElseBody []Statement
}

// WhileStatement represents an IR while loop.
type WhileStatement struct {
	Condition Expression
	Body      []Statement
}

// ReturnStatement represents an IR return statement.
type ReturnStatement struct {
	Values []Expression
}

// IdentifierExpression represents an IR identifier reference.
type IdentifierExpression struct {
	Name string
}

// CallExpression represents an IR function call.
type CallExpression struct {
	Callee    Expression
	Arguments []Expression
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

func (*Program) node()                           {}
func (*CallStatement) node()                     {}
func (*AssignStatement) node()                   {}
func (*FunctionDeclarationStatement) node()      {}
func (*LocalAssignStatement) node()              {}
func (*IfStatement) node()                       {}
func (*WhileStatement) node()                    {}
func (*ReturnStatement) node()                   {}
func (*IdentifierExpression) node()              {}
func (*CallExpression) node()                    {}
func (*NilExpression) node()                     {}
func (*BooleanExpression) node()                 {}
func (*NumberExpression) node()                  {}
func (*StringExpression) node()                  {}
func (*UnaryExpression) node()                   {}
func (*BinaryExpression) node()                  {}
func (*CallStatement) statement()                {}
func (*AssignStatement) statement()              {}
func (*FunctionDeclarationStatement) statement() {}
func (*LocalAssignStatement) statement()         {}
func (*IfStatement) statement()                  {}
func (*WhileStatement) statement()               {}
func (*ReturnStatement) statement()              {}
func (*IdentifierExpression) expression()        {}
func (*CallExpression) expression()              {}
func (*NilExpression) expression()               {}
func (*BooleanExpression) expression()           {}
func (*NumberExpression) expression()            {}
func (*StringExpression) expression()            {}
func (*UnaryExpression) expression()             {}
func (*BinaryExpression) expression()            {}

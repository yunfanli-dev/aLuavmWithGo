package ir

// Node 是所有 IR 节点共享的最小接口。
type Node interface {
	node()
}

// Statement 标记可以出现在 IR 程序语句列表中的节点。
type Statement interface {
	Node
	statement()
}

// Expression 标记可以出现在 IR 表达式位置中的节点。
type Expression interface {
	Node
	expression()
}

// Program 是由 AST 编译得到的顶层 IR 单元。
type Program struct {
	Statements []Statement
}

// CallStatement 表示作为语句存在的函数调用 IR 节点。
type CallStatement struct {
	Call *CallExpression
}

// AssignStatement 表示赋值语句对应的 IR 节点。
type AssignStatement struct {
	Targets []Expression
	Values  []Expression
}

// FunctionDeclarationStatement 表示具名函数声明对应的 IR 节点。
type FunctionDeclarationStatement struct {
	Name       string
	Parameters []string
	IsVararg   bool
	Body       []Statement
}

// LocalFunctionDeclarationStatement 表示局部函数声明对应的 IR 节点。
type LocalFunctionDeclarationStatement struct {
	Name       string
	Parameters []string
	IsVararg   bool
	Body       []Statement
}

// LocalAssignStatement 表示局部赋值语句对应的 IR 节点。
type LocalAssignStatement struct {
	Names  []string
	Values []Expression
}

// DoStatement 表示作用域块对应的 IR 节点。
type DoStatement struct {
	Body []Statement
}

// BreakStatement 表示 break 语句对应的 IR 节点。
type BreakStatement struct{}

// IfClause 表示 IR 中的一条条件分支。
type IfClause struct {
	Condition Expression
	Body      []Statement
}

// IfStatement 表示完整 if 结构对应的 IR 节点。
type IfStatement struct {
	Clauses  []IfClause
	ElseBody []Statement
}

// WhileStatement 表示 while 循环对应的 IR 节点。
type WhileStatement struct {
	Condition Expression
	Body      []Statement
}

// RepeatStatement 表示 repeat-until 循环对应的 IR 节点。
type RepeatStatement struct {
	Body      []Statement
	Condition Expression
}

// NumericForStatement 表示数值 for 循环对应的 IR 节点。
type NumericForStatement struct {
	Name  string
	Start Expression
	Limit Expression
	Step  Expression
	Body  []Statement
}

// GenericForStatement 表示 generic for-in 循环对应的 IR 节点。
type GenericForStatement struct {
	Names     []string
	Iterators []Expression
	Body      []Statement
}

// ReturnStatement 表示 return 语句对应的 IR 节点。
type ReturnStatement struct {
	Values []Expression
}

// IdentifierExpression 表示标识符引用对应的 IR 节点。
type IdentifierExpression struct {
	Name string
}

// CallExpression 表示函数调用对应的 IR 表达式节点。
type CallExpression struct {
	Callee    Expression
	Receiver  Expression
	Method    string
	Arguments []Expression
}

// FunctionExpression 表示匿名函数对应的 IR 表达式节点。
type FunctionExpression struct {
	Parameters []string
	IsVararg   bool
	Body       []Statement
}

// VarargExpression 表示 IR 中的 `...` 表达式。
type VarargExpression struct{}

// ParenthesizedExpression 表示带括号的 IR 表达式，并保留单值语义。
type ParenthesizedExpression struct {
	Inner Expression
}

// IndexExpression 表示 table 索引访问对应的 IR 节点。
type IndexExpression struct {
	Target Expression
	Index  Expression
}

// TableField 表示 IR table 构造器中的一个字段。
type TableField struct {
	Key         Expression
	Value       Expression
	IsListField bool
}

// TableConstructorExpression 表示 IR table 构造器节点。
type TableConstructorExpression struct {
	Fields []TableField
}

// NilExpression 表示 IR 中的 nil 字面量。
type NilExpression struct{}

// BooleanExpression 表示 IR 中的布尔字面量。
type BooleanExpression struct {
	Value bool
}

// NumberExpression 表示 IR 中的数字字面量。
type NumberExpression struct {
	Literal string
}

// StringExpression 表示 IR 中的字符串字面量。
type StringExpression struct {
	Value string
}

// UnaryExpression 表示 IR 中的一元运算。
type UnaryExpression struct {
	Operator string
	Operand  Expression
}

// BinaryExpression 表示 IR 中的二元运算。
type BinaryExpression struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (*Program) node()                                {}
func (*CallStatement) node()                          {}
func (*AssignStatement) node()                        {}
func (*FunctionDeclarationStatement) node()           {}
func (*LocalFunctionDeclarationStatement) node()      {}
func (*LocalAssignStatement) node()                   {}
func (*DoStatement) node()                            {}
func (*BreakStatement) node()                         {}
func (*IfStatement) node()                            {}
func (*WhileStatement) node()                         {}
func (*RepeatStatement) node()                        {}
func (*NumericForStatement) node()                    {}
func (*GenericForStatement) node()                    {}
func (*ReturnStatement) node()                        {}
func (*IdentifierExpression) node()                   {}
func (*CallExpression) node()                         {}
func (*FunctionExpression) node()                     {}
func (*VarargExpression) node()                       {}
func (*ParenthesizedExpression) node()                {}
func (*IndexExpression) node()                        {}
func (*TableConstructorExpression) node()             {}
func (*NilExpression) node()                          {}
func (*BooleanExpression) node()                      {}
func (*NumberExpression) node()                       {}
func (*StringExpression) node()                       {}
func (*UnaryExpression) node()                        {}
func (*BinaryExpression) node()                       {}
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
func (*IdentifierExpression) expression()             {}
func (*CallExpression) expression()                   {}
func (*FunctionExpression) expression()               {}
func (*VarargExpression) expression()                 {}
func (*ParenthesizedExpression) expression()          {}
func (*IndexExpression) expression()                  {}
func (*TableConstructorExpression) expression()       {}
func (*NilExpression) expression()                    {}
func (*BooleanExpression) expression()                {}
func (*NumberExpression) expression()                 {}
func (*StringExpression) expression()                 {}
func (*UnaryExpression) expression()                  {}
func (*BinaryExpression) expression()                 {}

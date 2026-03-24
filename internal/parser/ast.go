package parser

import "github.com/yunfanli-dev/aLuavmWithGo/internal/lexer"

// Node 是所有 AST 节点共享的最小接口。
// 每个节点都必须能标记自身类别，并返回对应的源码跨度。
type Node interface {
	node()
	Span() Span
}

// Statement 标记可以出现在 chunk / block 语句列表里的 AST 节点。
type Statement interface {
	Node
	statement()
}

// Expression 标记可以出现在表达式位置的 AST 节点。
type Expression interface {
	Node
	expression()
}

// Span 记录一个 AST 节点在原始源码中的起止范围。
type Span struct {
	Start lexer.Position
	End   lexer.Position
}

// Chunk 是一份 Lua 源码解析完成后的根节点。
type Chunk struct {
	Statements []Statement
	span       Span
}

// CallStatement 表示作为独立语句出现的函数调用。
type CallStatement struct {
	Call *CallExpression
	span Span
}

// AssignStatement 表示赋值语句，例如 `name = expr` 或 `a, b = ...`。
type AssignStatement struct {
	Targets []Expression
	Values  []Expression
	span    Span
}

// FunctionDeclarationStatement 表示具名函数声明语句 `function name(args) ... end`。
type FunctionDeclarationStatement struct {
	Name       Identifier
	Parameters []Identifier
	IsVararg   bool
	Body       []Statement
	span       Span
}

// LocalFunctionDeclarationStatement 表示局部函数声明 `local function name(args) ... end`。
type LocalFunctionDeclarationStatement struct {
	Name       Identifier
	Parameters []Identifier
	IsVararg   bool
	Body       []Statement
	span       Span
}

// LocalAssignStatement 表示局部变量赋值语句，例如 `local name = expr`。
type LocalAssignStatement struct {
	Names  []Identifier
	Values []Expression
	span   Span
}

// DoStatement 表示 `do ... end` 块语句。
type DoStatement struct {
	Body []Statement
	span Span
}

// BreakStatement 表示 `break` 语句。
type BreakStatement struct {
	span Span
}

// IfClause 表示 if / elseif 结构中的一个条件分支。
type IfClause struct {
	Condition Expression
	Body      []Statement
	span      Span
}

// IfStatement 表示完整的 `if` / `elseif` / `else` 结构。
type IfStatement struct {
	Clauses  []IfClause
	ElseBody []Statement
	span     Span
}

// WhileStatement 表示 Lua `while` 循环。
type WhileStatement struct {
	Condition Expression
	Body      []Statement
	span      Span
}

// RepeatStatement 表示 Lua `repeat ... until` 循环。
type RepeatStatement struct {
	Body      []Statement
	Condition Expression
	span      Span
}

// NumericForStatement 表示 Lua 数值 for 循环。
type NumericForStatement struct {
	Name  Identifier
	Start Expression
	Limit Expression
	Step  Expression
	Body  []Statement
	span  Span
}

// GenericForStatement 表示 Lua generic for-in 循环。
type GenericForStatement struct {
	Names     []Identifier
	Iterators []Expression
	Body      []Statement
	span      Span
}

// ReturnStatement 表示 Lua `return` 语句。
type ReturnStatement struct {
	Values []Expression
	span   Span
}

// Identifier 表示一个具名变量引用。
type Identifier struct {
	Name string
	span Span
}

// CallExpression 表示一次函数或方法调用表达式。
type CallExpression struct {
	Callee    Expression
	Receiver  Expression
	Method    string
	Arguments []Expression
	span      Span
}

// FunctionExpression 表示匿名函数表达式 `function(args) ... end`。
type FunctionExpression struct {
	Parameters []Identifier
	IsVararg   bool
	Body       []Statement
	span       Span
}

// VarargExpression 表示 Lua 的 `...` 表达式。
type VarargExpression struct {
	span Span
}

// ParenthesizedExpression 表示带括号的表达式，并显式保留分组与单值语义。
type ParenthesizedExpression struct {
	Inner Expression
	span  Span
}

// IndexExpression 表示索引访问，例如 `target[index]` 或点语法 `target.name`。
type IndexExpression struct {
	Target Expression
	Index  Expression
	span   Span
}

// TableField 表示 table 构造器中的一个字段。
type TableField struct {
	Key         Expression
	Value       Expression
	IsListField bool
	span        Span
}

// TableConstructorExpression 表示 `{ ... }` table 构造器。
type TableConstructorExpression struct {
	Fields []TableField
	span   Span
}

// NilExpression 表示 Lua `nil` 字面量。
type NilExpression struct {
	span Span
}

// BooleanExpression 表示布尔字面量 `true` 或 `false`。
type BooleanExpression struct {
	Value bool
	span  Span
}

// NumberExpression 表示数值字面量。
type NumberExpression struct {
	Literal string
	span    Span
}

// StringExpression 表示字符串字面量。
type StringExpression struct {
	Value string
	span  Span
}

// UnaryExpression 表示一元运算表达式。
type UnaryExpression struct {
	Operator lexer.TokenType
	Operand  Expression
	span     Span
}

// BinaryExpression 表示二元运算表达式。
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

// Span 返回 Chunk 对应的源码范围。
func (c *Chunk) Span() Span { return c.span }

// Span 返回函数调用语句对应的源码范围。
func (s *CallStatement) Span() Span { return s.span }

// Span 返回赋值语句对应的源码范围。
func (s *AssignStatement) Span() Span { return s.span }

// Span 返回函数声明语句对应的源码范围。
func (s *FunctionDeclarationStatement) Span() Span { return s.span }

// Span 返回局部函数声明语句对应的源码范围。
func (s *LocalFunctionDeclarationStatement) Span() Span { return s.span }

// Span 返回局部赋值语句对应的源码范围。
func (s *LocalAssignStatement) Span() Span { return s.span }

// Span 返回 do 块语句对应的源码范围。
func (s *DoStatement) Span() Span { return s.span }

// Span 返回 break 语句对应的源码范围。
func (s *BreakStatement) Span() Span { return s.span }

// Span 返回单个 if 分支对应的源码范围。
func (c *IfClause) Span() Span { return c.span }

// Span 返回完整 if 语句对应的源码范围。
func (s *IfStatement) Span() Span { return s.span }

// Span 返回 while 语句对应的源码范围。
func (s *WhileStatement) Span() Span { return s.span }

// Span 返回 repeat 语句对应的源码范围。
func (s *RepeatStatement) Span() Span { return s.span }

// Span 返回数值 for 语句对应的源码范围。
func (s *NumericForStatement) Span() Span { return s.span }

// Span 返回 generic for 语句对应的源码范围。
func (s *GenericForStatement) Span() Span { return s.span }

// Span 返回 return 语句对应的源码范围。
func (s *ReturnStatement) Span() Span { return s.span }

// Span 返回标识符表达式对应的源码范围。
func (e *Identifier) Span() Span { return e.span }

// Span 返回调用表达式对应的源码范围。
func (e *CallExpression) Span() Span { return e.span }

// Span 返回匿名函数表达式对应的源码范围。
func (e *FunctionExpression) Span() Span { return e.span }

// Span 返回 vararg 表达式对应的源码范围。
func (e *VarargExpression) Span() Span { return e.span }

// Span 返回括号表达式对应的源码范围。
func (e *ParenthesizedExpression) Span() Span { return e.span }

// Span 返回索引表达式对应的源码范围。
func (e *IndexExpression) Span() Span { return e.span }

// Span 返回 table 字段对应的源码范围。
func (f *TableField) Span() Span { return f.span }

// Span 返回 table 构造器表达式对应的源码范围。
func (e *TableConstructorExpression) Span() Span { return e.span }

// Span 返回 nil 表达式对应的源码范围。
func (e *NilExpression) Span() Span { return e.span }

// Span 返回布尔表达式对应的源码范围。
func (e *BooleanExpression) Span() Span { return e.span }

// Span 返回数字表达式对应的源码范围。
func (e *NumberExpression) Span() Span { return e.span }

// Span 返回字符串表达式对应的源码范围。
func (e *StringExpression) Span() Span { return e.span }

// Span 返回一元表达式对应的源码范围。
func (e *UnaryExpression) Span() Span { return e.span }

// Span 返回二元表达式对应的源码范围。
func (e *BinaryExpression) Span() Span { return e.span }

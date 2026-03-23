package parser

import "testing"

func TestParseLocalAssignAndReturnChunk(t *testing.T) {
	chunk, err := ParseString("sample.lua", "local value = 1 + 2\nreturn value\n")
	if err != nil {
		t.Fatalf("parse chunk: %v", err)
	}

	if len(chunk.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(chunk.Statements))
	}

	localStmt, ok := chunk.Statements[0].(*LocalAssignStatement)
	if !ok {
		t.Fatalf("expected first statement to be local assign, got %T", chunk.Statements[0])
	}

	if len(localStmt.Names) != 1 || localStmt.Names[0].Name != "value" {
		t.Fatalf("unexpected local names: %+v", localStmt.Names)
	}

	if len(localStmt.Values) != 1 {
		t.Fatalf("expected 1 local assignment value, got %d", len(localStmt.Values))
	}

	if _, ok := localStmt.Values[0].(*BinaryExpression); !ok {
		t.Fatalf("expected binary expression value, got %T", localStmt.Values[0])
	}

	returnStmt, ok := chunk.Statements[1].(*ReturnStatement)
	if !ok {
		t.Fatalf("expected second statement to be return, got %T", chunk.Statements[1])
	}

	if len(returnStmt.Values) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnStmt.Values))
	}
}

func TestParseExpressionPrecedence(t *testing.T) {
	chunk, err := ParseString("precedence.lua", "return 1 + 2 * 3\n")
	if err != nil {
		t.Fatalf("parse chunk: %v", err)
	}

	returnStmt := chunk.Statements[0].(*ReturnStatement)
	root, ok := returnStmt.Values[0].(*BinaryExpression)
	if !ok {
		t.Fatalf("expected binary expression root, got %T", returnStmt.Values[0])
	}

	if root.Operator != "+" {
		t.Fatalf("expected root operator '+', got %q", root.Operator)
	}

	right, ok := root.Right.(*BinaryExpression)
	if !ok {
		t.Fatalf("expected nested binary expression on right, got %T", root.Right)
	}

	if right.Operator != "*" {
		t.Fatalf("expected nested operator '*', got %q", right.Operator)
	}
}

func TestParseMultipleLocalNamesAndValues(t *testing.T) {
	chunk, err := ParseString("multi.lua", "local a, b = 1, foo\n")
	if err != nil {
		t.Fatalf("parse chunk: %v", err)
	}

	localStmt := chunk.Statements[0].(*LocalAssignStatement)
	if len(localStmt.Names) != 2 {
		t.Fatalf("expected 2 local names, got %d", len(localStmt.Names))
	}

	if len(localStmt.Values) != 2 {
		t.Fatalf("expected 2 local values, got %d", len(localStmt.Values))
	}
}

func TestParseReturnsHelpfulErrorForUnsupportedStatement(t *testing.T) {
	_, err := ParseString("unsupported.lua", "elseif true then return 1 end")
	if err == nil {
		t.Fatal("expected parser error")
	}

	parseErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected parser error type, got %T", err)
	}

	if parseErr.Token.Type != "elseif" {
		t.Fatalf("expected failing token to be 'elseif', got %q", parseErr.Token.Type)
	}
}

func TestParseIfElseAndWhile(t *testing.T) {
	chunk, err := ParseString("control.lua", `
local n = 0
while n < 3 do
	if n == 1 then
		n = n + 2
	else
		n = n + 1
	end
end
return n
`)
	if err != nil {
		t.Fatalf("parse control flow: %v", err)
	}

	if len(chunk.Statements) != 3 {
		t.Fatalf("expected 3 top-level statements, got %d", len(chunk.Statements))
	}

	if _, ok := chunk.Statements[1].(*WhileStatement); !ok {
		t.Fatalf("expected second statement to be while, got %T", chunk.Statements[1])
	}
}

func TestParseFunctionDeclarationAndCall(t *testing.T) {
	chunk, err := ParseString("functions.lua", `
function add(a, b)
	return a + b
end
return add(1, 2)
`)
	if err != nil {
		t.Fatalf("parse functions: %v", err)
	}

	if len(chunk.Statements) != 2 {
		t.Fatalf("expected 2 top-level statements, got %d", len(chunk.Statements))
	}

	if _, ok := chunk.Statements[0].(*FunctionDeclarationStatement); !ok {
		t.Fatalf("expected first statement to be function declaration, got %T", chunk.Statements[0])
	}

	returnStmt, ok := chunk.Statements[1].(*ReturnStatement)
	if !ok {
		t.Fatalf("expected second statement to be return, got %T", chunk.Statements[1])
	}

	if _, ok := returnStmt.Values[0].(*CallExpression); !ok {
		t.Fatalf("expected return value to be call expression, got %T", returnStmt.Values[0])
	}
}

func TestParseMethodDefinitionAndCall(t *testing.T) {
	chunk, err := ParseString("method.lua", `
function counter:inc(step)
	return self.value + step
end
return counter:inc(2)
`)
	if err != nil {
		t.Fatalf("parse method syntax: %v", err)
	}

	assignStmt, ok := chunk.Statements[0].(*AssignStatement)
	if !ok {
		t.Fatalf("expected first statement to be lowered assign, got %T", chunk.Statements[0])
	}

	if _, ok := assignStmt.Targets[0].(*IndexExpression); !ok {
		t.Fatalf("expected method definition target to be index expression, got %T", assignStmt.Targets[0])
	}

	fn, ok := assignStmt.Values[0].(*FunctionExpression)
	if !ok {
		t.Fatalf("expected lowered method definition value to be function expression, got %T", assignStmt.Values[0])
	}

	if len(fn.Parameters) == 0 || fn.Parameters[0].Name != "self" {
		t.Fatalf("expected implicit self parameter, got %+v", fn.Parameters)
	}

	returnStmt := chunk.Statements[1].(*ReturnStatement)
	call, ok := returnStmt.Values[0].(*CallExpression)
	if !ok {
		t.Fatalf("expected method call expression, got %T", returnStmt.Values[0])
	}

	if call.Receiver == nil || call.Method != "inc" {
		t.Fatalf("expected method receiver and name, got %#v", call)
	}
}

func TestParseTableAndStringCallSugar(t *testing.T) {
	chunk, err := ParseString("call_sugar.lua", `
return id{ answer = 42 }, id"hello"
`)
	if err != nil {
		t.Fatalf("parse call sugar: %v", err)
	}

	returnStmt := chunk.Statements[0].(*ReturnStatement)
	if len(returnStmt.Values) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnStmt.Values))
	}

	tableCall, ok := returnStmt.Values[0].(*CallExpression)
	if !ok {
		t.Fatalf("expected first return value to be call expression, got %T", returnStmt.Values[0])
	}

	if len(tableCall.Arguments) != 1 {
		t.Fatalf("expected table call to have 1 argument, got %d", len(tableCall.Arguments))
	}

	if _, ok := tableCall.Arguments[0].(*TableConstructorExpression); !ok {
		t.Fatalf("expected table call argument to be table constructor, got %T", tableCall.Arguments[0])
	}

	stringCall, ok := returnStmt.Values[1].(*CallExpression)
	if !ok {
		t.Fatalf("expected second return value to be call expression, got %T", returnStmt.Values[1])
	}

	if len(stringCall.Arguments) != 1 {
		t.Fatalf("expected string call to have 1 argument, got %d", len(stringCall.Arguments))
	}

	if literal, ok := stringCall.Arguments[0].(*StringExpression); !ok || literal.Value != "hello" {
		t.Fatalf("expected string call literal argument, got %#v", stringCall.Arguments[0])
	}
}

func TestParseMethodTableAndStringCallSugar(t *testing.T) {
	chunk, err := ParseString("method_call_sugar.lua", `
return obj:run{ answer = 42 }, obj:run"hello"
`)
	if err != nil {
		t.Fatalf("parse method call sugar: %v", err)
	}

	returnStmt := chunk.Statements[0].(*ReturnStatement)
	if len(returnStmt.Values) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnStmt.Values))
	}

	for index, value := range returnStmt.Values {
		call, ok := value.(*CallExpression)
		if !ok {
			t.Fatalf("expected return value %d to be call expression, got %T", index, value)
		}

		if call.Receiver == nil || call.Method != "run" {
			t.Fatalf("expected method call sugar to preserve receiver and method, got %#v", call)
		}
	}
}

func TestParseTableConstructorAndIndexing(t *testing.T) {
	chunk, err := ParseString("table.lua", `
local t = { answer = 42, ["name"] = "lua" }
t.answer = t["answer"] + 1
return t.name
`)
	if err != nil {
		t.Fatalf("parse table script: %v", err)
	}

	if len(chunk.Statements) != 3 {
		t.Fatalf("expected 3 top-level statements, got %d", len(chunk.Statements))
	}

	localStmt := chunk.Statements[0].(*LocalAssignStatement)
	if _, ok := localStmt.Values[0].(*TableConstructorExpression); !ok {
		t.Fatalf("expected local assignment value to be table constructor, got %T", localStmt.Values[0])
	}

	assignStmt := chunk.Statements[1].(*AssignStatement)
	if _, ok := assignStmt.Targets[0].(*IndexExpression); !ok {
		t.Fatalf("expected assignment target to be index expression, got %T", assignStmt.Targets[0])
	}
}

func TestParseTableConstructorTracksListField(t *testing.T) {
	chunk, err := ParseString("table_list.lua", `return { 1, pair() }`)
	if err != nil {
		t.Fatalf("parse table list: %v", err)
	}

	returnStmt := chunk.Statements[0].(*ReturnStatement)
	tableExpr := returnStmt.Values[0].(*TableConstructorExpression)
	if len(tableExpr.Fields) != 2 {
		t.Fatalf("expected 2 table fields, got %d", len(tableExpr.Fields))
	}

	if !tableExpr.Fields[0].IsListField || !tableExpr.Fields[1].IsListField {
		t.Fatalf("expected list fields to be marked, got %#v", tableExpr.Fields)
	}
}

func TestParseLocalAndAnonymousFunction(t *testing.T) {
	chunk, err := ParseString("closures.lua", `
local function addOne(n)
	return n + 1
end
local makeAdder = function(step)
	return function(value)
		return value + step
	end
end
return addOne(1), makeAdder(2)(3)
`)
	if err != nil {
		t.Fatalf("parse functions: %v", err)
	}

	if _, ok := chunk.Statements[0].(*LocalFunctionDeclarationStatement); !ok {
		t.Fatalf("expected first statement to be local function, got %T", chunk.Statements[0])
	}

	localAssign := chunk.Statements[1].(*LocalAssignStatement)
	if _, ok := localAssign.Values[0].(*FunctionExpression); !ok {
		t.Fatalf("expected local assignment value to be function expression, got %T", localAssign.Values[0])
	}
}

func TestParseVarargFunctionAndExpression(t *testing.T) {
	chunk, err := ParseString("vararg.lua", `
function pick(first, ...)
	return first, ...
end
`)
	if err != nil {
		t.Fatalf("parse vararg function: %v", err)
	}

	fn, ok := chunk.Statements[0].(*FunctionDeclarationStatement)
	if !ok {
		t.Fatalf("expected first statement to be function declaration, got %T", chunk.Statements[0])
	}

	if !fn.IsVararg {
		t.Fatal("expected function to be vararg")
	}

	returnStmt := fn.Body[0].(*ReturnStatement)
	if _, ok := returnStmt.Values[1].(*VarargExpression); !ok {
		t.Fatalf("expected second return expression to be vararg, got %T", returnStmt.Values[1])
	}
}

func TestParseRejectsVarargOutsideVarargFunction(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{
			name:  "top_level",
			input: `return ...`,
		},
		{
			name: "non_vararg_function",
			input: `
function pick(a)
	return ...
end
`,
		},
		{
			name: "nested_non_vararg_function",
			input: `
function outer(...)
	return function(a)
		return ...
	end
end
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseString("bad_vararg.lua", tc.input)
			if err == nil {
				t.Fatal("expected parser error")
			}

			parseErr, ok := err.(*Error)
			if !ok {
				t.Fatalf("expected parser error type, got %T", err)
			}

			if parseErr.Token.Type != "..." {
				t.Fatalf("expected failing token to be '...', got %q", parseErr.Token.Type)
			}
		})
	}
}

func TestParseParenthesizedExpressionPreservesNode(t *testing.T) {
	chunk, err := ParseString("paren.lua", `return (pair())`)
	if err != nil {
		t.Fatalf("parse parenthesized expression: %v", err)
	}

	returnStmt := chunk.Statements[0].(*ReturnStatement)
	if _, ok := returnStmt.Values[0].(*ParenthesizedExpression); !ok {
		t.Fatalf("expected return expression to be parenthesized, got %T", returnStmt.Values[0])
	}
}

func TestParseRepeatUntilAndNumericFor(t *testing.T) {
	chunk, err := ParseString("loops.lua", `
local total = 0
repeat
	total = total + 1
until total == 2
for i = 1, 3, 1 do
	total = total + i
end
return total
`)
	if err != nil {
		t.Fatalf("parse loops: %v", err)
	}

	if _, ok := chunk.Statements[1].(*RepeatStatement); !ok {
		t.Fatalf("expected second statement to be repeat, got %T", chunk.Statements[1])
	}

	if _, ok := chunk.Statements[2].(*NumericForStatement); !ok {
		t.Fatalf("expected third statement to be numeric for, got %T", chunk.Statements[2])
	}
}

func TestParseDoAndBreak(t *testing.T) {
	chunk, err := ParseString("do_break.lua", `
do
	local n = 1
end
while true do
	break
end
`)
	if err != nil {
		t.Fatalf("parse do/break: %v", err)
	}

	if _, ok := chunk.Statements[0].(*DoStatement); !ok {
		t.Fatalf("expected first statement to be do, got %T", chunk.Statements[0])
	}

	whileStmt, ok := chunk.Statements[1].(*WhileStatement)
	if !ok {
		t.Fatalf("expected second statement to be while, got %T", chunk.Statements[1])
	}

	if _, ok := whileStmt.Body[0].(*BreakStatement); !ok {
		t.Fatalf("expected while body to contain break, got %T", whileStmt.Body[0])
	}
}

func TestParseGenericFor(t *testing.T) {
	chunk, err := ParseString("generic_for.lua", `
for key, value in pairs({ answer = 42 }) do
	return key, value
end
`)
	if err != nil {
		t.Fatalf("parse generic for: %v", err)
	}

	loop, ok := chunk.Statements[0].(*GenericForStatement)
	if !ok {
		t.Fatalf("expected first statement to be generic for, got %T", chunk.Statements[0])
	}

	if len(loop.Names) != 2 || loop.Names[0].Name != "key" || loop.Names[1].Name != "value" {
		t.Fatalf("unexpected generic for names: %+v", loop.Names)
	}

	if len(loop.Iterators) != 1 {
		t.Fatalf("expected 1 iterator expression, got %d", len(loop.Iterators))
	}
}

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
	_, err := ParseString("unsupported.lua", "do return 1 end")
	if err == nil {
		t.Fatal("expected parser error")
	}

	parseErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected parser error type, got %T", err)
	}

	if parseErr.Token.Type != "do" {
		t.Fatalf("expected failing token to be 'do', got %q", parseErr.Token.Type)
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

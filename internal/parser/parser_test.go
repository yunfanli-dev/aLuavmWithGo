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
	_, err := ParseString("for.lua", "for i = 1, 3 do return i end")
	if err == nil {
		t.Fatal("expected parser error")
	}

	parseErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected parser error type, got %T", err)
	}

	if parseErr.Token.Type != "for" {
		t.Fatalf("expected failing token to be 'for', got %q", parseErr.Token.Type)
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

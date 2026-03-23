package ir

import (
	"testing"

	"github.com/yunfanli-dev/aLuavmWithGo/internal/parser"
)

func TestCompileChunkBuildsProgram(t *testing.T) {
	chunk, err := parser.ParseString("sample.lua", "local value = 1 + 2\nreturn value\n")
	if err != nil {
		t.Fatalf("parse chunk: %v", err)
	}

	program, err := CompileChunk(chunk)
	if err != nil {
		t.Fatalf("compile chunk: %v", err)
	}

	if len(program.Statements) != 2 {
		t.Fatalf("expected 2 IR statements, got %d", len(program.Statements))
	}

	localStmt, ok := program.Statements[0].(*LocalAssignStatement)
	if !ok {
		t.Fatalf("expected local assign statement, got %T", program.Statements[0])
	}

	if len(localStmt.Names) != 1 || localStmt.Names[0] != "value" {
		t.Fatalf("unexpected local names: %#v", localStmt.Names)
	}

	if _, ok := localStmt.Values[0].(*BinaryExpression); !ok {
		t.Fatalf("expected binary expression value, got %T", localStmt.Values[0])
	}
}

func TestCompileChunkBuildsControlFlowIR(t *testing.T) {
	chunk, err := parser.ParseString("control.lua", `
local n = 0
while n < 2 do
	if n == 0 then
		n = n + 1
	else
		n = n + 2
	end
end
return n
`)
	if err != nil {
		t.Fatalf("parse chunk: %v", err)
	}

	program, err := CompileChunk(chunk)
	if err != nil {
		t.Fatalf("compile chunk: %v", err)
	}

	if len(program.Statements) != 3 {
		t.Fatalf("expected 3 IR statements, got %d", len(program.Statements))
	}

	if _, ok := program.Statements[1].(*WhileStatement); !ok {
		t.Fatalf("expected second statement to be while IR, got %T", program.Statements[1])
	}
}

func TestCompileChunkBuildsFunctionIR(t *testing.T) {
	chunk, err := parser.ParseString("functions.lua", `
function add(a, b)
	return a + b
end
return add(1, 2)
`)
	if err != nil {
		t.Fatalf("parse chunk: %v", err)
	}

	program, err := CompileChunk(chunk)
	if err != nil {
		t.Fatalf("compile chunk: %v", err)
	}

	if _, ok := program.Statements[0].(*FunctionDeclarationStatement); !ok {
		t.Fatalf("expected first IR statement to be function declaration, got %T", program.Statements[0])
	}

	returnStmt, ok := program.Statements[1].(*ReturnStatement)
	if !ok {
		t.Fatalf("expected second IR statement to be return, got %T", program.Statements[1])
	}

	if _, ok := returnStmt.Values[0].(*CallExpression); !ok {
		t.Fatalf("expected return value to be call expression, got %T", returnStmt.Values[0])
	}
}

func TestCompileChunkBuildsMethodCallIR(t *testing.T) {
	chunk, err := parser.ParseString("method.lua", `
function counter:inc(step)
	return self.value + step
end
return counter:inc(2)
`)
	if err != nil {
		t.Fatalf("parse chunk: %v", err)
	}

	program, err := CompileChunk(chunk)
	if err != nil {
		t.Fatalf("compile chunk: %v", err)
	}

	assignStmt, ok := program.Statements[0].(*AssignStatement)
	if !ok {
		t.Fatalf("expected first IR statement to be lowered assign, got %T", program.Statements[0])
	}

	fn, ok := assignStmt.Values[0].(*FunctionExpression)
	if !ok {
		t.Fatalf("expected lowered method definition value to be function expression, got %T", assignStmt.Values[0])
	}

	if len(fn.Parameters) == 0 || fn.Parameters[0] != "self" {
		t.Fatalf("expected implicit self parameter, got %#v", fn.Parameters)
	}

	returnStmt := program.Statements[1].(*ReturnStatement)
	call, ok := returnStmt.Values[0].(*CallExpression)
	if !ok {
		t.Fatalf("expected method call IR expression, got %T", returnStmt.Values[0])
	}

	if call.Receiver == nil || call.Method != "inc" {
		t.Fatalf("expected IR method receiver and name, got %#v", call)
	}
}

func TestCompileChunkBuildsTableIR(t *testing.T) {
	chunk, err := parser.ParseString("table.lua", `
local t = { answer = 42 }
t.answer = 43
return t["answer"]
`)
	if err != nil {
		t.Fatalf("parse chunk: %v", err)
	}

	program, err := CompileChunk(chunk)
	if err != nil {
		t.Fatalf("compile chunk: %v", err)
	}

	localStmt := program.Statements[0].(*LocalAssignStatement)
	if _, ok := localStmt.Values[0].(*TableConstructorExpression); !ok {
		t.Fatalf("expected local IR value to be table constructor, got %T", localStmt.Values[0])
	}

	assignStmt := program.Statements[1].(*AssignStatement)
	if _, ok := assignStmt.Targets[0].(*IndexExpression); !ok {
		t.Fatalf("expected assignment target to be index expression, got %T", assignStmt.Targets[0])
	}
}

func TestCompileChunkPreservesTableListFieldMarkers(t *testing.T) {
	chunk, err := parser.ParseString("table_list.lua", `return { 1, pair() }`)
	if err != nil {
		t.Fatalf("parse chunk: %v", err)
	}

	program, err := CompileChunk(chunk)
	if err != nil {
		t.Fatalf("compile chunk: %v", err)
	}

	returnStmt := program.Statements[0].(*ReturnStatement)
	tableExpr := returnStmt.Values[0].(*TableConstructorExpression)
	if len(tableExpr.Fields) != 2 {
		t.Fatalf("expected 2 table fields, got %d", len(tableExpr.Fields))
	}

	if !tableExpr.Fields[0].IsListField || !tableExpr.Fields[1].IsListField {
		t.Fatalf("expected IR list fields to be marked, got %#v", tableExpr.Fields)
	}
}

func TestCompileChunkBuildsLocalAndAnonymousFunctionIR(t *testing.T) {
	chunk, err := parser.ParseString("closures.lua", `
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
		t.Fatalf("parse chunk: %v", err)
	}

	program, err := CompileChunk(chunk)
	if err != nil {
		t.Fatalf("compile chunk: %v", err)
	}

	if _, ok := program.Statements[0].(*LocalFunctionDeclarationStatement); !ok {
		t.Fatalf("expected first IR statement to be local function declaration, got %T", program.Statements[0])
	}

	localAssign := program.Statements[1].(*LocalAssignStatement)
	if _, ok := localAssign.Values[0].(*FunctionExpression); !ok {
		t.Fatalf("expected local assignment value to be function expression, got %T", localAssign.Values[0])
	}
}

func TestCompileChunkBuildsVarargFunctionIR(t *testing.T) {
	chunk, err := parser.ParseString("vararg.lua", `
function pick(first, ...)
	return first, ...
end
`)
	if err != nil {
		t.Fatalf("parse chunk: %v", err)
	}

	program, err := CompileChunk(chunk)
	if err != nil {
		t.Fatalf("compile chunk: %v", err)
	}

	fn, ok := program.Statements[0].(*FunctionDeclarationStatement)
	if !ok {
		t.Fatalf("expected first IR statement to be function declaration, got %T", program.Statements[0])
	}

	if !fn.IsVararg {
		t.Fatal("expected IR function to be vararg")
	}

	returnStmt := fn.Body[0].(*ReturnStatement)
	if _, ok := returnStmt.Values[1].(*VarargExpression); !ok {
		t.Fatalf("expected second IR return expression to be vararg, got %T", returnStmt.Values[1])
	}
}

func TestCompileChunkPreservesParenthesizedExpressionIR(t *testing.T) {
	chunk, err := parser.ParseString("paren.lua", `return (pair())`)
	if err != nil {
		t.Fatalf("parse chunk: %v", err)
	}

	program, err := CompileChunk(chunk)
	if err != nil {
		t.Fatalf("compile chunk: %v", err)
	}

	returnStmt := program.Statements[0].(*ReturnStatement)
	if _, ok := returnStmt.Values[0].(*ParenthesizedExpression); !ok {
		t.Fatalf("expected IR return expression to be parenthesized, got %T", returnStmt.Values[0])
	}
}

func TestCompileChunkBuildsRepeatAndNumericForIR(t *testing.T) {
	chunk, err := parser.ParseString("loops.lua", `
local total = 0
repeat
	total = total + 1
until total == 2
for i = 1, 3 do
	total = total + i
end
return total
`)
	if err != nil {
		t.Fatalf("parse chunk: %v", err)
	}

	program, err := CompileChunk(chunk)
	if err != nil {
		t.Fatalf("compile chunk: %v", err)
	}

	if _, ok := program.Statements[1].(*RepeatStatement); !ok {
		t.Fatalf("expected second IR statement to be repeat, got %T", program.Statements[1])
	}

	if _, ok := program.Statements[2].(*NumericForStatement); !ok {
		t.Fatalf("expected third IR statement to be numeric for, got %T", program.Statements[2])
	}
}

func TestCompileChunkBuildsDoAndBreakIR(t *testing.T) {
	chunk, err := parser.ParseString("do_break.lua", `
do
	local n = 1
end
while true do
	break
end
`)
	if err != nil {
		t.Fatalf("parse chunk: %v", err)
	}

	program, err := CompileChunk(chunk)
	if err != nil {
		t.Fatalf("compile chunk: %v", err)
	}

	if _, ok := program.Statements[0].(*DoStatement); !ok {
		t.Fatalf("expected first IR statement to be do, got %T", program.Statements[0])
	}

	whileStmt, ok := program.Statements[1].(*WhileStatement)
	if !ok {
		t.Fatalf("expected second IR statement to be while, got %T", program.Statements[1])
	}

	if _, ok := whileStmt.Body[0].(*BreakStatement); !ok {
		t.Fatalf("expected while body to contain break, got %T", whileStmt.Body[0])
	}
}

func TestCompileChunkBuildsGenericForIR(t *testing.T) {
	chunk, err := parser.ParseString("generic_for.lua", `
for key, value in pairs({ answer = 42 }) do
	return key, value
end
`)
	if err != nil {
		t.Fatalf("parse chunk: %v", err)
	}

	program, err := CompileChunk(chunk)
	if err != nil {
		t.Fatalf("compile chunk: %v", err)
	}

	loop, ok := program.Statements[0].(*GenericForStatement)
	if !ok {
		t.Fatalf("expected first IR statement to be generic for, got %T", program.Statements[0])
	}

	if len(loop.Names) != 2 || loop.Names[0] != "key" || loop.Names[1] != "value" {
		t.Fatalf("unexpected generic for IR names: %#v", loop.Names)
	}

	if len(loop.Iterators) != 1 {
		t.Fatalf("expected 1 iterator expression, got %d", len(loop.Iterators))
	}
}

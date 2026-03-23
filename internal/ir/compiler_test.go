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

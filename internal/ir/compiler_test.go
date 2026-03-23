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

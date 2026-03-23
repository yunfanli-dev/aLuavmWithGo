package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewVMCreatesBootstrapState(t *testing.T) {
	vm := NewVM()
	if vm == nil {
		t.Fatal("expected vm instance")
	}

	if vm.state == nil {
		t.Fatal("expected vm state")
	}

	if vm.state.StackSize() != 0 {
		t.Fatalf("expected empty stack, got %d", vm.state.StackSize())
	}
}

func TestExecStringAllowsEmptyBootstrapScript(t *testing.T) {
	vm := NewVM()

	if err := vm.ExecString(""); err != nil {
		t.Fatalf("expected empty source to succeed, got %v", err)
	}
}

func TestExecSourceAllowsEmptyNamedBootstrapScript(t *testing.T) {
	vm := NewVM()

	if err := vm.ExecSource(NewStringSource("empty.lua", "   ")); err != nil {
		t.Fatalf("expected empty named source to succeed, got %v", err)
	}
}

func TestExecFileLoadsScriptFromDisk(t *testing.T) {
	vm := NewVM()
	scriptPath := filepath.Join(t.TempDir(), "empty.lua")

	if err := os.WriteFile(scriptPath, []byte(" \n\t"), 0o644); err != nil {
		t.Fatalf("write script file: %v", err)
	}

	if err := vm.ExecFile(scriptPath); err != nil {
		t.Fatalf("expected file execution bootstrap to succeed, got %v", err)
	}
}

func TestExecFileReturnsReadError(t *testing.T) {
	vm := NewVM()

	if err := vm.ExecFile(filepath.Join(t.TempDir(), "missing.lua")); err == nil {
		t.Fatal("expected missing file error")
	}
}

func TestExecStringExecutesCompiledSource(t *testing.T) {
	vm := NewVM()

	err := vm.ExecString("local value = 1\nreturn value\n")
	if err != nil {
		t.Fatalf("expected execution success, got %v", err)
	}

	if vm.state.LastProgram() == nil {
		t.Fatal("expected compiled frontend result to be stored")
	}

	if len(vm.state.LastProgram().Program.Statements) != 2 {
		t.Fatalf("expected 2 compiled statements, got %d", len(vm.state.LastProgram().Program.Statements))
	}

	returnValues := vm.state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != "number" || returnValues[0].Data != float64(1) {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

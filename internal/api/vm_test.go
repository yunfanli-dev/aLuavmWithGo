package api

import (
	"bytes"
	"context"
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

func TestRegisterFunctionExposesGoHandler(t *testing.T) {
	vm := NewVM()

	err := vm.RegisterFunction("double", func(args []Value) ([]Value, error) {
		number := args[0].Data.(float64)
		return []Value{{Type: "number", Data: number * 2}}, nil
	})
	if err != nil {
		t.Fatalf("register function: %v", err)
	}

	if err := vm.ExecString("return double(5)"); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := vm.state.LastReturnValues()
	if len(returnValues) != 1 || returnValues[0].Data != float64(10) {
		t.Fatalf("unexpected return values: %#v", returnValues)
	}
}

func TestRegisterPreloadFunctionExposesHostModule(t *testing.T) {
	vm := NewVM()

	err := vm.RegisterPreloadFunction("hostmod", func(args []Value) ([]Value, error) {
		if len(args) != 1 || args[0].Type != "string" || args[0].Data != "hostmod" {
			t.Fatalf("unexpected preload args: %#v", args)
		}

		return []Value{{Type: "number", Data: float64(42)}}, nil
	})
	if err != nil {
		t.Fatalf("register preload function: %v", err)
	}

	if err := vm.ExecString("return require(\"hostmod\")"); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := vm.state.LastReturnValues()
	if len(returnValues) != 1 || returnValues[0].Data != float64(42) {
		t.Fatalf("unexpected return values: %#v", returnValues)
	}
}

func TestRegisterLoadedModuleExposesCachedHostValue(t *testing.T) {
	vm := NewVM()

	err := vm.RegisterLoadedModule("cached", Value{Type: "string", Data: "ready"})
	if err != nil {
		t.Fatalf("register loaded module: %v", err)
	}

	if err := vm.ExecString(`return require("cached")`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := vm.state.LastReturnValues()
	if len(returnValues) != 1 || returnValues[0].Type != "string" || returnValues[0].Data != "ready" {
		t.Fatalf("unexpected return values: %#v", returnValues)
	}
}

func TestRegisterSearcherFunctionExposesHostSearcher(t *testing.T) {
	vm := NewVM()

	err := vm.RegisterSearcherFunction(func(moduleName string) (ModuleLoader, string, error) {
		if moduleName != "virtual" {
			return nil, "\n\tno host searcher match", nil
		}

		return func(moduleName string) ([]Value, error) {
			return []Value{{Type: "string", Data: "host:" + moduleName}}, nil
		}, "", nil
	})
	if err != nil {
		t.Fatalf("register searcher function: %v", err)
	}

	if err := vm.ExecString(`return require("virtual")`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := vm.state.LastReturnValues()
	if len(returnValues) != 1 || returnValues[0].Type != "string" || returnValues[0].Data != "host:virtual" {
		t.Fatalf("unexpected return values: %#v", returnValues)
	}
}

func TestBuiltinPrintWritesToConfiguredOutput(t *testing.T) {
	vm := NewVM()
	var output bytes.Buffer
	vm.SetOutput(&output)

	if err := vm.ExecString("print(\"hello\", 42)"); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	if output.String() != "hello\t42\n" {
		t.Fatalf("unexpected print output %q", output.String())
	}
}

func TestSetStepLimitStopsInfiniteLoop(t *testing.T) {
	vm := NewVM()
	vm.SetStepLimit(20)

	err := vm.ExecString(`
while true do
end
`)
	if err == nil {
		t.Fatal("expected step limit error")
	}

	if err.Error() != `execute compiled Lua source "<memory>": execution step limit exceeded` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecStringWithContextStopsCanceledScript(t *testing.T) {
	vm := NewVM()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := vm.ExecStringWithContext(ctx, `
while true do
end
`)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}

	if err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
	}
}

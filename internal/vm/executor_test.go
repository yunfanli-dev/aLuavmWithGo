package vm

import "testing"

func TestExecStringEvaluatesArithmeticAndReturn(t *testing.T) {
	state := NewState()

	if err := state.ExecString("local value = 1 + 2 * 3\nreturn value\n"); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(7) {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesStringConcatAndLength(t *testing.T) {
	state := NewState()

	if err := state.ExecString("local text = \"ab\" .. \"cd\"\nreturn #text\n"); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(4) {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesBooleanLogic(t *testing.T) {
	state := NewState()

	if err := state.ExecString("local flag = false or true\nreturn flag\n"); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != true {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesIfElseAndAssignment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local n = 1
if n == 1 then
	n = n + 4
else
	n = n + 8
end
return n
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(5) {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesWhileLoop(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local n = 0
while n < 3 do
	n = n + 1
end
return n
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(3) {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringHonorsBlockLocalScope(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local n = 1
if true then
	local n = 5
end
return n
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(1) {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesFunctionCall(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function add(a, b)
	return a + b
end
return add(2, 5)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(7) {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringSupportsRecursiveFunction(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function countdown(n)
	if n == 0 then
		return 0
	end
	return countdown(n - 1)
end
return countdown(3)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(0) {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

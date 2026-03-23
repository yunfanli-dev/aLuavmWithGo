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

func TestExecStringEvaluatesRepeatUntil(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local n = 0
repeat
	n = n + 2
until n >= 6
return n
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(6) {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesNumericFor(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local total = 0
for i = 1, 4 do
	total = total + i
end
return total
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(10) {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesNumericForNegativeStep(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local total = 0
for i = 5, 1, -2 do
	total = total + i
end
return total
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(9) {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesGenericForWithPairs(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local result = ""
for key, value in pairs({ answer = 42 }) do
	result = key .. ":" .. tostring(value)
end
return result
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "answer:42" {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesGenericForWithIPairs(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local total = 0
for _, value in ipairs({ [1] = 3, [2] = 4, [3] = 5 }) do
	total = total + value
end
return total
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(12) {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesBuiltinNext(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local key, value = next({ answer = 42 })
return key, value
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "answer" {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(42) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
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

func TestExecStringEvaluatesTableConstructorAndFieldAccess(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local t = { answer = 42, ["name"] = "lua" }
return t.answer, t["name"]
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(42) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != "lua" {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

func TestExecStringEvaluatesTableAssignment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local t = {}
t["count"] = 1
t.count = t["count"] + 2
return t.count
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

func TestExecStringEvaluatesLocalFunction(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local function addOne(n)
	return n + 1
end
return addOne(2)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 || returnValues[0].Data != float64(3) {
		t.Fatalf("unexpected return values: %#v", returnValues)
	}
}

func TestExecStringEvaluatesAnonymousFunctionClosureRead(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local makeAdder = function(step)
	return function(value)
		return value + step
	end
end
return makeAdder(2)(3)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 || returnValues[0].Data != float64(5) {
		t.Fatalf("unexpected return values: %#v", returnValues)
	}
}

func TestExecStringEvaluatesClosureWriteBack(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local counter = 0
local inc = function()
	counter = counter + 1
	return counter
end
inc()
inc()
return counter
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 || returnValues[0].Data != float64(2) {
		t.Fatalf("unexpected return values: %#v", returnValues)
	}
}

func TestExecStringSupportsLocalRecursiveFunction(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local function countdown(n)
	if n == 0 then
		return 0
	end
	return countdown(n - 1)
end
return countdown(2)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 || returnValues[0].Data != float64(0) {
		t.Fatalf("unexpected return values: %#v", returnValues)
	}
}

func TestExecStringEvaluatesBuiltinTypeAndToString(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`return type(1), tostring(false), type({})`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != "number" || returnValues[1].Data != "false" || returnValues[2].Data != "table" {
		t.Fatalf("unexpected return values: %#v", returnValues)
	}
}

func TestExecStringEvaluatesBuiltinToNumber(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`return tonumber(" 42 "), tonumber("nope")`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(42) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNil {
		t.Fatalf("expected second return value to be nil, got %#v", returnValues[1])
	}
}

func TestExecStringEvaluatesBuiltinAssert(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`return assert(true, "bad"), assert(1)`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != true {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(1) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

func TestExecStringExpandsLastCallReturnValues(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pair()
	return 1, 2
end
return 0, pair()
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(0) || returnValues[1].Data != float64(1) || returnValues[2].Data != float64(2) {
		t.Fatalf("unexpected return values: %#v", returnValues)
	}
}

func TestExecStringBuiltinAssertReturnsError(t *testing.T) {
	state := NewState()

	err := state.ExecString(`assert(false, "boom")`)
	if err == nil {
		t.Fatal("expected assert failure")
	}

	if err.Error() != `execute compiled Lua source "<memory>": boom` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecStringBuiltinErrorReturnsError(t *testing.T) {
	state := NewState()

	err := state.ExecString(`error("fail")`)
	if err == nil {
		t.Fatal("expected error failure")
	}

	if err.Error() != `execute compiled Lua source "<memory>": fail` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecStringEvaluatesBuiltinPCallSuccess(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pair()
	return 7, 8
end
return pcall(pair)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != true {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Data != float64(7) || returnValues[2].Data != float64(8) {
		t.Fatalf("unexpected protected call values: %#v", returnValues)
	}
}

func TestExecStringEvaluatesBuiltinPCallFailure(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`return pcall(function() error("boom") end)`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != false {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != "boom" {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

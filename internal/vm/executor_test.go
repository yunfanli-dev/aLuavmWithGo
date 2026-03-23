package vm

import (
	"math"
	"testing"
)

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

func TestExecStringEvaluatesExponentNumberLiterals(t *testing.T) {
	state := NewState()

	if err := state.ExecString("return 1e3 + 2.5e-1\n"); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(1000.25) {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesHexNumberLiterals(t *testing.T) {
	state := NewState()

	if err := state.ExecString("return 0xff + 0X10\n"); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(271) {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesLongStringLiteral(t *testing.T) {
	state := NewState()

	if err := state.ExecString("return [[hello\nworld]]\n"); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "hello\nworld" {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesLeveledLongStringLiteral(t *testing.T) {
	state := NewState()

	if err := state.ExecString("return [==[hello [world] ]=] test]==]\n"); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "hello [world] ]=] test" {
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

func TestExecStringEvaluatesMathLibrary(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`return math.abs(-3), math.floor(2.9), math.ceil(2.1), math.max(1, 7, 3), math.min(1, 7, 3), math.sqrt(9), math.pow(2, 5), math.log(math.exp(1)), math.sin(0), math.cos(0)`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 10 {
		t.Fatalf("expected 10 return values, got %d", len(returnValues))
	}

	expected := []float64{3, 2, 3, 7, 1, 3, 32}
	for index, want := range expected[:7] {
		if returnValues[index].Type != ValueTypeNumber || returnValues[index].Data != want {
			t.Fatalf("unexpected math return value at %d: %#v", index, returnValues[index])
		}
	}

	approximate := []struct {
		index int
		want  float64
	}{
		{index: 7, want: 1},
		{index: 8, want: 0},
		{index: 9, want: 1},
	}
	for _, entry := range approximate {
		value := returnValues[entry.index]
		number, ok := value.Data.(float64)
		if value.Type != ValueTypeNumber || !ok || math.Abs(number-entry.want) > 1e-9 {
			t.Fatalf("unexpected approximate math return value at %d: %#v", entry.index, value)
		}
	}
}

func TestExecStringEvaluatesMathRandomAndSeed(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
math.randomseed(7)
local a = math.random()
local b = math.random(5)
local c = math.random(3, 7)
math.randomseed(7)
return a, b, c, math.random(), math.random(5), math.random(3, 7)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 6 {
		t.Fatalf("expected 6 return values, got %d", len(returnValues))
	}

	for _, index := range []int{0, 3} {
		value := returnValues[index]
		number, ok := value.Data.(float64)
		if value.Type != ValueTypeNumber || !ok || number < 0 || number >= 1 {
			t.Fatalf("unexpected math.random float value at %d: %#v", index, value)
		}
	}

	if returnValues[1] != returnValues[4] {
		t.Fatalf("expected reseeded bounded random values to match: %#v vs %#v", returnValues[1], returnValues[4])
	}

	if returnValues[2] != returnValues[5] {
		t.Fatalf("expected reseeded ranged random values to match: %#v vs %#v", returnValues[2], returnValues[5])
	}

	for _, index := range []int{1, 4} {
		if returnValues[index].Type != ValueTypeNumber {
			t.Fatalf("unexpected math.random bounded value at %d: %#v", index, returnValues[index])
		}
		number := returnValues[index].Data.(float64)
		if number < 1 || number > 5 {
			t.Fatalf("out-of-range math.random bounded value at %d: %#v", index, returnValues[index])
		}
	}

	for _, index := range []int{2, 5} {
		if returnValues[index].Type != ValueTypeNumber {
			t.Fatalf("unexpected math.random ranged value at %d: %#v", index, returnValues[index])
		}
		number := returnValues[index].Data.(float64)
		if number < 3 || number > 7 {
			t.Fatalf("out-of-range math.random ranged value at %d: %#v", index, returnValues[index])
		}
	}
}

func TestExecStringEvaluatesStringLibrary(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local b1, b2, b3 = string.byte("ABC", 1, 3)
return string.len("AbCd"), string.lower("AbCd"), string.upper("AbCd"), string.sub("abcdef", 2, 4), string.sub("abcdef", -3, -1), string.rep("ha", 3), string.reverse("stressed"), string.byte("ABC", 2), b1, b2, b3, string.char(65, 66, 67)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 12 {
		t.Fatalf("expected 12 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(4) {
		t.Fatalf("unexpected string.len return value: %#v", returnValues[0])
	}

	expected := []string{"abcd", "ABCD", "bcd", "def", "hahaha", "desserts"}
	for index, want := range expected {
		value := returnValues[index+1]
		if value.Type != ValueTypeString || value.Data != want {
			t.Fatalf("unexpected string return value at %d: %#v", index+1, value)
		}
	}

	expectedNumbers := []float64{66, 65, 66, 67}
	for index, want := range expectedNumbers {
		value := returnValues[index+7]
		if value.Type != ValueTypeNumber || value.Data != want {
			t.Fatalf("unexpected string.byte return value at %d: %#v", index+7, value)
		}
	}

	if returnValues[11].Type != ValueTypeString || returnValues[11].Data != "ABC" {
		t.Fatalf("unexpected string.char return value: %#v", returnValues[11])
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

func TestExecStringEvaluatesDoBlockScope(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local n = 1
do
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

func TestExecStringEvaluatesBreakInLoop(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local n = 0
while true do
	n = n + 1
	break
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

func TestExecStringRejectsBreakOutsideLoop(t *testing.T) {
	state := NewState()

	err := state.ExecString(`
do
	break
end
`)
	if err == nil {
		t.Fatal("expected break outside loop error")
	}

	if err.Error() != `execute compiled Lua source "<memory>": break outside loop` {
		t.Fatalf("unexpected error: %v", err)
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

func TestExecStringEvaluatesTableCallSugar(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function id(value)
	return value
end
local result = id{ answer = 42 }
return result.answer
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(42) {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesStringCallSugar(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function id(value)
	return value
end
return id"hello"
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "hello" {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesMethodCallSugar(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local obj = { value = 40 }
function obj:add(payload)
	return self.value + payload.delta
end
function obj:say(message)
	return self.value .. ":" .. message
end
return obj:add{ delta = 2 }, obj:say"ok"
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

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != "40:ok" {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
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

func TestExecStringEvaluatesMethodDefinitionAndCall(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local counter = { value = 5 }
function counter:inc(step)
	self.value = self.value + step
	return self.value
end
return counter:inc(2), counter.value
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(7) || returnValues[1].Data != float64(7) {
		t.Fatalf("unexpected return values: %#v", returnValues)
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

func TestExecStringExpandsLastCallInTableConstructorListField(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pair()
	return 2, 3
end
local t = { 1, pair() }
return t[1], t[2], t[3]
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(1) || returnValues[1].Data != float64(2) || returnValues[2].Data != float64(3) {
		t.Fatalf("unexpected return values: %#v", returnValues)
	}
}

func TestExecStringDoesNotExpandNonTrailingTableConstructorListField(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pair()
	return 2, 3
end
local t = { pair(), 4 }
return t[1], t[2], t[3]
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(2) || returnValues[1].Data != float64(4) || returnValues[2].Type != ValueTypeNil {
		t.Fatalf("unexpected return values: %#v", returnValues)
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

func TestExecStringEvaluatesMetatableIndexFallback(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local fallback = { answer = 42 }
local target = {}
setmetatable(target, { __index = fallback })
return target.answer, getmetatable(target).__index.answer
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

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(42) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

func TestExecStringEvaluatesMetatableNewIndexFallback(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local store = {}
local target = {}
setmetatable(target, { __newindex = store })
target.answer = 42
return target.answer, store.answer
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNil {
		t.Fatalf("expected first return value to be nil, got %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(42) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

func TestExecStringEvaluatesMetatableFunctionHandlers(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local writes = {}
local target = {}
setmetatable(target, {
	__index = function(_, key)
		return "missing:" .. key
	end,
	__newindex = function(_, key, value)
		writes[key] = value + 1
	end
})
target.answer = 41
return target.answer, writes.answer
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "missing:answer" {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(42) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

func TestExecStringEvaluatesMetatableToString(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local value = {}
setmetatable(value, {
	__tostring = function()
		return "meta-table"
	end
})
return tostring(value)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "meta-table" {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesMetatableCall(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local callable = {}
setmetatable(callable, {
	__call = function(self, value)
		return value + 2
	end
})
return callable(5)
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

func TestExecStringEvaluatesArithmeticMetamethods(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local lhs = {}
local rhs = {}
setmetatable(lhs, {
	__add = function(_, _)
		return 7
	end,
	__sub = function(_, _)
		return 3
	end,
	__mul = function(_, _)
		return 12
	end,
	__div = function(_, _)
		return 2
	end,
	__mod = function(_, _)
		return 1
	end,
	__pow = function(_, _)
		return 16
	end
})
return lhs + rhs, lhs - rhs, lhs * rhs, lhs / rhs, lhs % rhs, lhs ^ rhs
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 6 {
		t.Fatalf("expected 6 return values, got %d", len(returnValues))
	}

	expected := []float64{7, 3, 12, 2, 1, 16}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber || returnValues[index].Data != want {
			t.Fatalf("unexpected return value at %d: %#v", index, returnValues[index])
		}
	}
}

func TestExecStringEvaluatesUnaryAndConcatMetamethods(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local lhs = {}
local rhs = {}
setmetatable(lhs, {
	__unm = function(_)
		return 9
	end,
	__concat = function(_, _)
		return "joined"
	end
})
return -lhs, lhs .. rhs
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(9) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != "joined" {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

func TestExecStringEvaluatesComparisonMetamethods(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local lhs = {}
local rhs = {}
setmetatable(lhs, {
	__eq = function(_, _)
		return true
	end,
	__lt = function(_, _)
		return true
	end,
	__le = function(_, _)
		return false
	end
})
return lhs == rhs, lhs ~= rhs, lhs < rhs, lhs > rhs, lhs <= rhs, lhs >= rhs
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 6 {
		t.Fatalf("expected 6 return values, got %d", len(returnValues))
	}

	expected := []bool{true, false, true, true, false, false}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeBoolean || returnValues[index].Data != want {
			t.Fatalf("unexpected return value at %d: %#v", index, returnValues[index])
		}
	}
}

func TestExecStringEvaluatesProtectedMetatableView(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local target = {}
setmetatable(target, { __metatable = "locked" })
return getmetatable(target)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "locked" {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringRejectsProtectedMetatableChange(t *testing.T) {
	state := NewState()

	err := state.ExecString(`
local target = {}
setmetatable(target, { __metatable = "locked" })
setmetatable(target, {})
`)
	if err == nil {
		t.Fatal("expected protected metatable error")
	}

	if err.Error() != `execute compiled Lua source "<memory>": cannot change a protected metatable` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecStringEvaluatesRawGetAndRawSet(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local target = {}
setmetatable(target, {
	__index = function()
		return "fallback"
	end,
	__newindex = function()
	end
})
rawset(target, "answer", 42)
return rawget(target, "answer"), target.answer
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

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(42) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

func TestExecStringEvaluatesRawEqual(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local lhs = {}
local rhs = {}
setmetatable(lhs, {
	__eq = function(_, _)
		return true
	end
})
setmetatable(rhs, getmetatable(lhs))
local alias = lhs
return lhs == rhs, rawequal(lhs, rhs), rawequal(lhs, alias), rawequal(1, 1), rawequal(1, 2)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 5 {
		t.Fatalf("expected 5 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != true {
		t.Fatalf("unexpected metamethod equality result: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeBoolean || returnValues[1].Data != false {
		t.Fatalf("unexpected raw table equality result: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeBoolean || returnValues[2].Data != true {
		t.Fatalf("unexpected raw alias equality result: %#v", returnValues[2])
	}

	if returnValues[3].Type != ValueTypeBoolean || returnValues[3].Data != true {
		t.Fatalf("unexpected raw numeric equality result: %#v", returnValues[3])
	}

	if returnValues[4].Type != ValueTypeBoolean || returnValues[4].Data != false {
		t.Fatalf("unexpected raw numeric inequality result: %#v", returnValues[4])
	}
}

func TestExecStringEvaluatesTableInsertAndRemove(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local values = { 1, 3 }
table.insert(values, 2)
table.insert(values, 2, 9)
local removed_mid = table.remove(values, 2)
local removed_last = table.remove(values)
return values[1], values[2], values[3], values[4], removed_mid, removed_last
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 6 {
		t.Fatalf("expected 6 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(1) || returnValues[1].Data != float64(3) || returnValues[2].Type != ValueTypeNil || returnValues[3].Type != ValueTypeNil {
		t.Fatalf("unexpected table contents after insert/remove: %#v", returnValues[:4])
	}

	if returnValues[4].Data != float64(9) || returnValues[5].Data != float64(2) {
		t.Fatalf("unexpected removed values: %#v", returnValues[4:6])
	}
}

func TestExecStringEvaluatesTableRemoveOnEmptySequence(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local values = {}
return table.remove(values)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNil {
		t.Fatalf("expected nil from empty table.remove, got %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesTableConcat(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local values = { "a", "b", "c", 4 }
return table.concat(values), table.concat(values, "-"), table.concat(values, "-", 2, 3), table.concat(values, "-", 5, 4)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "abc4" {
		t.Fatalf("unexpected default concat value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != "a-b-c-4" {
		t.Fatalf("unexpected separated concat value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeString || returnValues[2].Data != "b-c" {
		t.Fatalf("unexpected sliced concat value: %#v", returnValues[2])
	}

	if returnValues[3].Type != ValueTypeString || returnValues[3].Data != "" {
		t.Fatalf("unexpected empty-range concat value: %#v", returnValues[3])
	}
}

func TestExecStringEvaluatesTableSort(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local ascending = { 3, 1, 2 }
table.sort(ascending)

local descending = { 3, 1, 2 }
table.sort(descending, function(a, b)
	return a > b
end)

return table.concat(ascending, ","), table.concat(descending, ",")
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "1,2,3" {
		t.Fatalf("unexpected ascending sort value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != "3,2,1" {
		t.Fatalf("unexpected descending sort value: %#v", returnValues[1])
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

func TestExecStringEvaluatesBuiltinSelect(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function capture(...)
	local a = select("#", ...)
	local b, c = select(2, ...)
	local d = select(-1, ...)
	return a, b, c, d
end
return capture(4, 5, 6)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(3) {
		t.Fatalf("unexpected count result: %#v", returnValues[0])
	}

	if returnValues[1].Data != float64(5) || returnValues[2].Data != float64(6) {
		t.Fatalf("unexpected select(2, ...) values: %#v", returnValues[1:3])
	}

	if returnValues[3].Data != float64(6) {
		t.Fatalf("unexpected select(-1, ...) value: %#v", returnValues[3])
	}
}

func TestExecStringEvaluatesBuiltinUnpack(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local values = { 10, 20, 30, 40 }
local a, b, c, d = unpack(values)
local e, f = unpack(values, 2, 3)
local g, h = unpack(values, 5, 4)
return a, b, c, d, e, f, g, h
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 8 {
		t.Fatalf("expected 8 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(10) || returnValues[1].Data != float64(20) || returnValues[2].Data != float64(30) || returnValues[3].Data != float64(40) {
		t.Fatalf("unexpected unpack(values) values: %#v", returnValues[:4])
	}

	if returnValues[4].Data != float64(20) || returnValues[5].Data != float64(30) {
		t.Fatalf("unexpected unpack(values, 2, 3) values: %#v", returnValues[4:6])
	}

	if returnValues[6].Type != ValueTypeNil || returnValues[7].Type != ValueTypeNil {
		t.Fatalf("unexpected unpack(values, 5, 4) values: %#v", returnValues[6:8])
	}
}

func TestExecStringBuiltinSelectRejectsZeroIndex(t *testing.T) {
	state := NewState()

	err := state.ExecString(`
function capture(...)
	return select(0, ...)
end
return capture(1, 2)
`)
	if err == nil {
		t.Fatal("expected select index error")
	}

	if err.Error() != `execute compiled Lua source "<memory>": select index out of range` {
		t.Fatalf("unexpected error: %v", err)
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

func TestExecStringSuppressesExpansionForParenthesizedCall(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pair()
	return 1, 2
end
return (pair())
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(1) {
		t.Fatalf("unexpected return values: %#v", returnValues)
	}
}

func TestExecStringSuppressesExpansionForParenthesizedVararg(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pack(...)
	return (...)
end
return pack(4, 5, 6)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(4) {
		t.Fatalf("unexpected return values: %#v", returnValues)
	}
}

func TestExecStringSuppressesExpansionForParenthesizedCallInTableConstructor(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pair()
	return 2, 3
end
local t = { 1, (pair()) }
return t[1], t[2], t[3]
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(1) || returnValues[1].Data != float64(2) || returnValues[2].Type != ValueTypeNil {
		t.Fatalf("unexpected return values: %#v", returnValues)
	}
}

func TestExecStringEvaluatesVarargReturnAndCall(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pack(first, ...)
	return first, ...
end
return pack(1, 2, 3)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(1) || returnValues[1].Data != float64(2) || returnValues[2].Data != float64(3) {
		t.Fatalf("unexpected return values: %#v", returnValues)
	}
}

func TestExecStringEvaluatesVarargAsSingleExpressionWhenNotExpanded(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function head(...)
	local first = ...
	return first
end
return head(4, 5, 6)
`); err != nil {
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

func TestExecStringEvaluatesAssignmentMultivalueAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pair()
	return 1, 2
end
local a, b, c = pair()
local d, e = pair(), 4
local f, g, h = 0, pair()
return a, b, c, d, e, f, g, h
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 8 {
		t.Fatalf("expected 8 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(1) || returnValues[1].Data != float64(2) || returnValues[2].Type != ValueTypeNil {
		t.Fatalf("unexpected first assignment values: %#v", returnValues[:3])
	}

	if returnValues[3].Data != float64(1) || returnValues[4].Data != float64(4) {
		t.Fatalf("unexpected second assignment values: %#v", returnValues[3:5])
	}

	if returnValues[5].Data != float64(0) || returnValues[6].Data != float64(1) || returnValues[7].Data != float64(2) {
		t.Fatalf("unexpected third assignment values: %#v", returnValues[5:8])
	}
}

func TestExecStringEvaluatesCallArgumentMultivalueAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pair()
	return 1, 2
end
function pack(a, b, c)
	return a, b, c
end
local a, b, c = pack(pair())
local d, e, f = pack(pair(), 4)
local g, h, i = pack(0, pair())
return a, b, c, d, e, f, g, h, i
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 9 {
		t.Fatalf("expected 9 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(1) || returnValues[1].Data != float64(2) || returnValues[2].Type != ValueTypeNil {
		t.Fatalf("unexpected first packed values: %#v", returnValues[:3])
	}

	if returnValues[3].Data != float64(1) || returnValues[4].Data != float64(4) || returnValues[5].Type != ValueTypeNil {
		t.Fatalf("unexpected second packed values: %#v", returnValues[3:6])
	}

	if returnValues[6].Data != float64(0) || returnValues[7].Data != float64(1) || returnValues[8].Data != float64(2) {
		t.Fatalf("unexpected third packed values: %#v", returnValues[6:9])
	}
}

func TestExecStringEvaluatesVarargAssignmentAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function capture(...)
	local a, b, c = ...
	local d, e = (...), 4
	return a, b, c, d, e
end
return capture(1, 2, 3)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 5 {
		t.Fatalf("expected 5 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(1) || returnValues[1].Data != float64(2) || returnValues[2].Data != float64(3) {
		t.Fatalf("unexpected expanded assignment values: %#v", returnValues[:3])
	}

	if returnValues[3].Data != float64(1) || returnValues[4].Data != float64(4) {
		t.Fatalf("unexpected parenthesized assignment values: %#v", returnValues[3:5])
	}
}

func TestExecStringEvaluatesVarargCallArgumentAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pack(a, b, c)
	return a, b, c
end
function capture(...)
	local a, b, c = pack(...)
	local d, e, f = pack(..., 4)
	local g, h, i = pack(0, ...)
	return a, b, c, d, e, f, g, h, i
end
return capture(1, 2, 3)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 9 {
		t.Fatalf("expected 9 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(1) || returnValues[1].Data != float64(2) || returnValues[2].Data != float64(3) {
		t.Fatalf("unexpected direct vararg call values: %#v", returnValues[:3])
	}

	if returnValues[3].Data != float64(1) || returnValues[4].Data != float64(4) || returnValues[5].Type != ValueTypeNil {
		t.Fatalf("unexpected non-final vararg call values: %#v", returnValues[3:6])
	}

	if returnValues[6].Data != float64(0) || returnValues[7].Data != float64(1) || returnValues[8].Data != float64(2) {
		t.Fatalf("unexpected final vararg call values: %#v", returnValues[6:9])
	}
}

func TestExecStringEvaluatesVarargTableConstructorAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function capture(...)
	local t = { 0, ... }
	local u = { (...) }
	return t[1], t[2], t[3], t[4], u[1], u[2]
end
return capture(1, 2, 3)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 6 {
		t.Fatalf("expected 6 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(0) || returnValues[1].Data != float64(1) || returnValues[2].Data != float64(2) || returnValues[3].Data != float64(3) {
		t.Fatalf("unexpected expanded table values: %#v", returnValues[:4])
	}

	if returnValues[4].Data != float64(1) || returnValues[5].Type != ValueTypeNil {
		t.Fatalf("unexpected parenthesized table values: %#v", returnValues[4:6])
	}
}

func TestExecStringEvaluatesMethodCallMultivalueAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local obj = {}
function obj:pair()
	return 1, 2
end
function pack(a, b, c)
	return a, b, c
end
local a, b, c = pack(obj:pair())
local d, e, f = pack(obj:pair(), 4)
local g, h, i = pack(0, obj:pair())
local j, k = (obj:pair()), 4
return a, b, c, d, e, f, g, h, i, j, k
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 11 {
		t.Fatalf("expected 11 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(1) || returnValues[1].Data != float64(2) || returnValues[2].Type != ValueTypeNil {
		t.Fatalf("unexpected direct method values: %#v", returnValues[:3])
	}

	if returnValues[3].Data != float64(1) || returnValues[4].Data != float64(4) || returnValues[5].Type != ValueTypeNil {
		t.Fatalf("unexpected non-final method values: %#v", returnValues[3:6])
	}

	if returnValues[6].Data != float64(0) || returnValues[7].Data != float64(1) || returnValues[8].Data != float64(2) {
		t.Fatalf("unexpected final method values: %#v", returnValues[6:9])
	}

	if returnValues[9].Data != float64(1) || returnValues[10].Data != float64(4) {
		t.Fatalf("unexpected parenthesized method values: %#v", returnValues[9:11])
	}
}

func TestExecStringEvaluatesPCallMultivalueAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pair()
	return 1, 2
end
function pack(a, b, c, d)
	return a, b, c, d
end
local a, b, c = pcall(pair)
local d, e, f = pcall(pair), 4
local g, h, i, j = 0, pcall(pair)
local k, l, m, n = pack(pcall(pair))
local o, p, q, r = pack(0, pcall(pair))
return a, b, c, d, e, f, g, h, i, j, k, l, m, n, o, p, q, r
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 18 {
		t.Fatalf("expected 18 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != true || returnValues[1].Data != float64(1) || returnValues[2].Data != float64(2) {
		t.Fatalf("unexpected direct pcall values: %#v", returnValues[:3])
	}

	if returnValues[3].Type != ValueTypeBoolean || returnValues[3].Data != true || returnValues[4].Data != float64(4) || returnValues[5].Type != ValueTypeNil {
		t.Fatalf("unexpected non-final pcall assignment values: %#v", returnValues[3:6])
	}

	if returnValues[6].Data != float64(0) || returnValues[7].Type != ValueTypeBoolean || returnValues[7].Data != true || returnValues[8].Data != float64(1) || returnValues[9].Data != float64(2) {
		t.Fatalf("unexpected final pcall assignment values: %#v", returnValues[6:10])
	}

	if returnValues[10].Type != ValueTypeBoolean || returnValues[10].Data != true || returnValues[11].Data != float64(1) || returnValues[12].Data != float64(2) || returnValues[13].Type != ValueTypeNil {
		t.Fatalf("unexpected direct pcall call-arg values: %#v", returnValues[10:14])
	}

	if returnValues[14].Data != float64(0) || returnValues[15].Type != ValueTypeBoolean || returnValues[15].Data != true || returnValues[16].Data != float64(1) || returnValues[17].Data != float64(2) {
		t.Fatalf("unexpected final pcall call-arg values: %#v", returnValues[14:18])
	}
}

func TestExecStringEvaluatesNestedExpressionListMultivalueAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pair()
	return 1, 2
end
function capture(...)
	local a, b, c = ..., pair()
	local d, e, f = ..., (pair())
	local t = { ..., pair(), (pair()) }
	return a, b, c, d, e, f, t[1], t[2], t[3], t[4], t[5]
end
return capture(7, 8)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 11 {
		t.Fatalf("expected 11 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(7) || returnValues[1].Data != float64(1) || returnValues[2].Data != float64(2) {
		t.Fatalf("unexpected first nested values: %#v", returnValues[:3])
	}

	if returnValues[3].Data != float64(7) || returnValues[4].Data != float64(1) || returnValues[5].Type != ValueTypeNil {
		t.Fatalf("unexpected second nested values: %#v", returnValues[3:6])
	}

	if returnValues[6].Data != float64(7) || returnValues[7].Data != float64(1) || returnValues[8].Data != float64(1) || returnValues[9].Type != ValueTypeNil || returnValues[10].Type != ValueTypeNil {
		t.Fatalf("unexpected nested table values: %#v", returnValues[6:11])
	}
}

func TestExecStringEvaluatesEmptyVarargAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function capture(...)
	local a, b = ...
	local c, d = 0, ...
	local t = { 0, ... }
	return a, b, c, d, t[1], t[2]
end
return capture()
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 6 {
		t.Fatalf("expected 6 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNil || returnValues[1].Type != ValueTypeNil {
		t.Fatalf("unexpected empty vararg assignment values: %#v", returnValues[:2])
	}

	if returnValues[2].Data != float64(0) || returnValues[3].Type != ValueTypeNil {
		t.Fatalf("unexpected empty vararg mixed assignment values: %#v", returnValues[2:4])
	}

	if returnValues[4].Data != float64(0) || returnValues[5].Type != ValueTypeNil {
		t.Fatalf("unexpected empty vararg table values: %#v", returnValues[4:6])
	}
}

func TestExecStringEvaluatesPCallFailureMultivalueAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function fail()
	error("boom")
end
function pack(a, b, c, d)
	return a, b, c, d
end
local a, b, c = pcall(fail)
local d, e, f, g = 0, pcall(fail)
local h, i, j, k = pack(pcall(fail))
return a, b, c, d, e, f, g, h, i, j, k
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 11 {
		t.Fatalf("expected 11 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != false || returnValues[1].Type != ValueTypeString || returnValues[1].Data != "boom" || returnValues[2].Type != ValueTypeNil {
		t.Fatalf("unexpected direct failing pcall values: %#v", returnValues[:3])
	}

	if returnValues[3].Data != float64(0) || returnValues[4].Type != ValueTypeBoolean || returnValues[4].Data != false || returnValues[5].Type != ValueTypeString || returnValues[5].Data != "boom" || returnValues[6].Type != ValueTypeNil {
		t.Fatalf("unexpected mixed failing pcall values: %#v", returnValues[3:7])
	}

	if returnValues[7].Type != ValueTypeBoolean || returnValues[7].Data != false || returnValues[8].Type != ValueTypeString || returnValues[8].Data != "boom" || returnValues[9].Type != ValueTypeNil || returnValues[10].Type != ValueTypeNil {
		t.Fatalf("unexpected failing pcall call-arg values: %#v", returnValues[7:11])
	}
}

func TestExecStringEvaluatesZeroResultCallAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function noop()
	return
end
function pack(a, b, c)
	return a, b, c
end
local a, b = noop()
local c, d = 0, noop()
local e, f, g = pack(noop())
local t = { 1, noop() }
return a, b, c, d, e, f, g, t[1], t[2]
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 9 {
		t.Fatalf("expected 9 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNil || returnValues[1].Type != ValueTypeNil {
		t.Fatalf("unexpected zero-result assignment values: %#v", returnValues[:2])
	}

	if returnValues[2].Data != float64(0) || returnValues[3].Type != ValueTypeNil {
		t.Fatalf("unexpected zero-result mixed assignment values: %#v", returnValues[2:4])
	}

	if returnValues[4].Type != ValueTypeNil || returnValues[5].Type != ValueTypeNil || returnValues[6].Type != ValueTypeNil {
		t.Fatalf("unexpected zero-result call-arg values: %#v", returnValues[4:7])
	}

	if returnValues[7].Data != float64(1) || returnValues[8].Type != ValueTypeNil {
		t.Fatalf("unexpected zero-result table values: %#v", returnValues[7:9])
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

func TestExecStringEvaluatesBuiltinXPCallSuccess(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pair()
	return 7, 8
end
function handler(err)
	return "wrapped:" .. err
end
return xpcall(pair, handler)
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
		t.Fatalf("unexpected xpcall success values: %#v", returnValues[1:])
	}
}

func TestExecStringEvaluatesBuiltinXPCallFailure(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function fail()
	error("boom")
end
function handler(err)
	return "wrapped:" .. err
end
return xpcall(fail, handler)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != false {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != `wrapped:boom` {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

func TestExecStringEvaluatesBuiltinXPCallHandlerFailure(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function fail()
	error("boom")
end
function bad_handler(err)
	error("handler:" .. err)
end
return xpcall(fail, bad_handler)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != false {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != `handler:boom` {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

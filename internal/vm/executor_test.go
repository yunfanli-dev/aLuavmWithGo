package vm

import (
	"context"
	"math"
	"os"
	"path/filepath"
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

func TestExecStringEvaluatesNumericStringConcat(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`return 1 .. "x", "x" .. 2`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "1x" {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != "x2" {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

func TestExecStringRejectsInvalidPrimitiveConcatOperand(t *testing.T) {
	state := NewState()

	err := state.ExecString(`return "x" .. true`)
	if err == nil {
		t.Fatal("expected invalid concat operand error")
	}

	if err.Error() != `execute compiled Lua source "<memory>": operator ".." expects string-like operands, got string and boolean` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecStringEvaluatesOrderedStringComparisons(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`return "ant" < "bat", "bat" <= "bat", "cat" > "bat", "cat" >= "cat"`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	expected := []bool{true, true, true, true}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeBoolean || returnValues[index].Data != want {
			t.Fatalf("unexpected ordered string comparison value at %d: %#v", index, returnValues[index])
		}
	}
}

func TestExecStringCoercesNumericStringsInArithmeticAndNumericFor(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local total = 0
for i = "1", "3" do
	total = total + i
end
return "2" + 3, -"4", math.abs("5"), total
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	expected := []float64{5, -4, 5, 6}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber || returnValues[index].Data != want {
			t.Fatalf("unexpected numeric-string coercion value at %d: %#v", index, returnValues[index])
		}
	}
}

func TestExecStringRejectsMixedStringNumberOrderedComparison(t *testing.T) {
	state := NewState()

	err := state.ExecString(`return "2" < 10`)
	if err == nil {
		t.Fatal("expected mixed string-number comparison error")
	}

	if err.Error() != `execute compiled Lua source "<memory>": operator "<" expects number operand, got string` {
		t.Fatalf("unexpected error: %v", err)
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

func TestExecStringAssignsUndeclaredNameIntoGlobalScope(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function set_answer()
	answer = 42
end

set_answer()
return answer
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

func TestExecStringExposesMinimalGlobalEnvThroughG(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
answer = 7
_G.extra = 9
rawset(_G, "third", 11)
return _G.answer, extra, third, type(_G.print), rawequal(_G, _G._G)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 5 {
		t.Fatalf("expected 5 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(7) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(9) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeNumber || returnValues[2].Data != float64(11) {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}

	if returnValues[3].Type != ValueTypeString || returnValues[3].Data != "function" {
		t.Fatalf("unexpected fourth return value: %#v", returnValues[3])
	}

	if returnValues[4].Type != ValueTypeBoolean || returnValues[4].Data != true {
		t.Fatalf("unexpected fifth return value: %#v", returnValues[4])
	}
}

func TestExecStringSupportsMinimalModuleAndPackageSeeAll(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local mod = module("alpha.beta", package.seeall)
mod.answer = 42
return rawequal(mod, package.loaded["alpha.beta"]),
       rawequal(mod, alpha.beta),
       mod._NAME,
       mod._PACKAGE,
       type(mod.print),
       alpha.beta.answer
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 6 {
		t.Fatalf("expected 6 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != true {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeBoolean || returnValues[1].Data != true {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeString || returnValues[2].Data != "alpha.beta" {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}

	if returnValues[3].Type != ValueTypeString || returnValues[3].Data != "alpha." {
		t.Fatalf("unexpected fourth return value: %#v", returnValues[3])
	}

	if returnValues[4].Type != ValueTypeString || returnValues[4].Data != "function" {
		t.Fatalf("unexpected fifth return value: %#v", returnValues[4])
	}

	if returnValues[5].Type != ValueTypeNumber || returnValues[5].Data != float64(42) {
		t.Fatalf("unexpected sixth return value: %#v", returnValues[5])
	}
}

func TestExecStringModuleSwitchesCurrentChunkEnvironment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
module("legacy.mod", package.seeall)
answer = 12

function greet()
	return answer, type(print)
end

return legacy.mod.answer, legacy.mod.greet()
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(12) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(12) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeString || returnValues[2].Data != "function" {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}
}

func TestExecStringModuleSyncsCurrentFrameEnvironment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local mod = module("legacy.env", package.seeall)
return rawequal(getfenv(1), mod), rawequal(getfenv(0), mod)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != true {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeBoolean || returnValues[1].Data != true {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

func TestExecStringModuleBindsPathIntoCurrentThreadEnvironment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local env = {}
setmetatable(env, { __index = _G })
setfenv(0, env)

local mod = module("sandbox.mod", package.seeall)
return rawequal(env.sandbox.mod, mod), rawget(_G, "sandbox"), rawequal(package.loaded["sandbox.mod"], mod)
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

	if returnValues[1].Type != ValueTypeNil {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeBoolean || returnValues[2].Data != true {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}
}

func TestExecStringRequireExecutesModuleInCurrentThreadEnvironment(t *testing.T) {
	state := NewState()
	tempDir := t.TempDir()
	modulePath := filepath.Join(tempDir, "child.lua")
	if err := os.WriteFile(modulePath, []byte(`
value = answer + 1
kind = type(print)
return value
`), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	if err := state.ExecSource(Source{
		Name: modulePath,
		Content: `
local env = { answer = 41 }
setmetatable(env, { __index = _G })
setfenv(0, env)
return require("child"), kind
`,
	}); err != nil {
		t.Fatalf("exec source: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(42) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != "function" {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

func TestExecSourceRequireExecutesModuleInCurrentFunctionEnvironment(t *testing.T) {
	state := NewState()
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.lua")
	modulePath := filepath.Join(tempDir, "child.lua")
	if err := os.WriteFile(modulePath, []byte(`
seen = answer + 1
return { answer = seen }
`), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	if err := state.ExecSource(Source{
		Name: mainPath,
		Content: `
local env = { answer = 41 }
setmetatable(env, { __index = _G })

local function load_child()
	setfenv(load_child, env)
	local mod = require("child")
	return mod.answer, env.seen, rawget(_G, "seen")
end

return load_child()
`,
	}); err != nil {
		t.Fatalf("exec source: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(42) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(42) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeNil {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}
}

func TestExecSourceRequireModuleRespectsCurrentFunctionEnvironment(t *testing.T) {
	state := NewState()
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.lua")
	modulePath := filepath.Join(tempDir, "child.lua")
	if err := os.WriteFile(modulePath, []byte(`
local mod = module("child.module", package.seeall)
answer = answer + 1
return mod, answer, rawget(_G, "answer")
`), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	if err := state.ExecSource(Source{
		Name: mainPath,
		Content: `
local env = { answer = 41 }
setmetatable(env, { __index = _G })

local function load_child()
	setfenv(load_child, env)
	local mod = require("child")
	return mod.answer, env.child.module.answer, rawget(_G, "child"), rawequal(env.child.module, mod), env.answer
end

return load_child()
`,
	}); err != nil {
		t.Fatalf("exec source: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 5 {
		t.Fatalf("expected 5 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(42) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(42) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeNil {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}

	if returnValues[3].Type != ValueTypeBoolean || returnValues[3].Data != true {
		t.Fatalf("unexpected fourth return value: %#v", returnValues[3])
	}

	if returnValues[4].Type != ValueTypeNumber || returnValues[4].Data != float64(41) {
		t.Fatalf("unexpected fifth return value: %#v", returnValues[4])
	}
}

func TestExecStringPackageSeeAllFallsBackToCurrentThreadEnvironment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local env = { answer = 41 }
setmetatable(env, { __index = _G })
setfenv(0, env)

local mod = module("thread.visible", package.seeall)
return mod.answer, rawget(_G, "answer"), type(mod.print)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(41) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNil {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeString || returnValues[2].Data != "function" {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}
}

func TestExecStringGetFEnvForNativeFunctionFollowsCurrentThreadEnvironment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local env = { answer = 41 }
setmetatable(env, { __index = _G })
setfenv(0, env)
return rawequal(getfenv(print), env), getfenv(print).answer
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != true {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(41) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

func TestExecStringPackageSeeAllAfterModuleSwitchUsesOriginalThreadEnvironment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local env = { answer = 41 }
setmetatable(env, { __index = _G })
setfenv(0, env)

module("legacy.late")
env.package.seeall(_M)
return answer, type(print)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(41) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != "function" {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

func TestExecStringPackageSeeAllDoesNotExposeInternalBaseField(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local mod = module("hidden.meta", package.seeall)
return getmetatable(mod).__seeall_base
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNil {
		t.Fatalf("unexpected return value: %#v", returnValues[0])
	}
}

func TestExecStringSupportsMinimalGetFEnvAndSetFEnv(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local env = { answer = 41 }
setmetatable(env, { __index = _G })

local function read_answer()
	return answer + 1
end

local before = getfenv(read_answer)
setfenv(read_answer, env)

return rawequal(before, _G), getfenv(read_answer).answer, read_answer(), type(getfenv(0).print)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != true {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(41) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeNumber || returnValues[2].Data != float64(42) {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}

	if returnValues[3].Type != ValueTypeString || returnValues[3].Data != "function" {
		t.Fatalf("unexpected fourth return value: %#v", returnValues[3])
	}
}

func TestExecStringSetFEnvOnCurrentFunctionAppliesImmediately(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local env = { answer = 40 }
setmetatable(env, { __index = _G })

local function read_answer()
	setfenv(read_answer, env)
	return answer + 2
end

return read_answer(), getfenv(read_answer).answer
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

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(40) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

func TestExecStringSetFEnvOnCallerFunctionValueAppliesImmediately(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local env = { answer = 41 }
setmetatable(env, { __index = _G })

local outer
outer = function()
	local function inner()
		setfenv(outer, env)
	end

	inner()
	return answer + 1
end

return outer(), getfenv(outer).answer
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

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(41) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

func TestExecStringModuleSeeAllFallsBackToCurrentFunctionEnvironment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local env = { answer = 41 }
setmetatable(env, { __index = _G })

local function build()
	setfenv(build, env)
	local mod = module("legacy.fn_env", package.seeall)
	return answer + 1, mod.answer, rawequal(env.legacy.fn_env, mod), rawget(_G, "legacy")
end

return build()
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(42) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(41) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeBoolean || returnValues[2].Data != true {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}

	if returnValues[3].Type != ValueTypeNil {
		t.Fatalf("unexpected fourth return value: %#v", returnValues[3])
	}
}

func TestExecStringLatePackageSeeAllFallsBackToCurrentFunctionEnvironment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local env = { answer = 41 }
setmetatable(env, { __index = _G })

local function build()
	setfenv(build, env)
	module("legacy.fn_env_late")
	env.package.seeall(_M)
	return answer + 1, _M.answer, rawequal(env.legacy.fn_env_late, _M), rawget(_G, "legacy")
end

return build()
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(42) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(41) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeBoolean || returnValues[2].Data != true {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}

	if returnValues[3].Type != ValueTypeNil {
		t.Fatalf("unexpected fourth return value: %#v", returnValues[3])
	}
}

func TestExecStringSupportsSetFEnvOnCallerFrame(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local env = { answer = 39 }
setmetatable(env, { __index = _G })

local function outer()
	local function inner()
		return type(getfenv(2).print), setfenv(2, env)
	end

	inner()
	return answer + 3
end

return outer()
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

func TestExecStringSetFEnvOnCallerFramePersistsForLaterCalls(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local env = { answer = 41 }
setmetatable(env, { __index = _G })

local function outer()
	local function inner()
		setfenv(2, env)
	end

	inner()
	return answer + 1
end

local first = outer()
local second = outer()
return first, second, getfenv(outer).answer
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(42) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(42) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeNumber || returnValues[2].Data != float64(41) {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}
}

func TestExecStringSupportsThreadLevelGetFEnvAndSetFEnv(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local env = { answer = 40 }
setmetatable(env, { __index = _G })

setfenv(0, env)

function read_answer()
	return answer + 2
end

return getfenv(0).answer, read_answer(), type(getfenv(0).print)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(40) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(42) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeString || returnValues[2].Data != "function" {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}
}

func TestExecStringRespectsStepLimit(t *testing.T) {
	state := NewState()
	state.SetStepLimit(100)

	if err := state.ExecString(`
local sum = 0
for i = 1, 5 do
	sum = sum + i
end
return sum
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 || returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(15) {
		t.Fatalf("unexpected return values: %#v", returnValues)
	}
}

func TestExecStringStepLimitStopsInfiniteLoop(t *testing.T) {
	state := NewState()
	state.SetStepLimit(20)

	err := state.ExecString(`
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
	state := NewState()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := state.ExecStringWithContext(ctx, `
while true do
end
`)
	if err != context.Canceled {
		t.Fatalf("unexpected error: %v", err)
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

func TestExecStringEvaluatesMathModF(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local i1, f1 = math.modf(3.75)
local i2, f2 = math.modf(-2.25)
return i1, f1, i2, f2
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	expected := []float64{3, 0.75, -2, -0.25}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber || returnValues[index].Data != want {
			t.Fatalf("unexpected math.modf return value at %d: %#v", index, returnValues[index])
		}
	}
}

func TestExecStringEvaluatesMathDegAndRad(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
return math.deg(math.pi), math.rad(180), math.deg(math.pi / 2), math.rad(90), math.huge > 1e308
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 5 {
		t.Fatalf("expected 5 return values, got %d", len(returnValues))
	}

	expected := []float64{180, math.Pi, 90, math.Pi / 2}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber {
			t.Fatalf("unexpected math.deg/math.rad return type at %d: %#v", index, returnValues[index])
		}

		got := returnValues[index].Data.(float64)
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("unexpected math.deg/math.rad return value at %d: got %v want %v", index, got, want)
		}
	}

	if returnValues[4].Type != ValueTypeBoolean || returnValues[4].Data != true {
		t.Fatalf("unexpected math.huge comparison result: %#v", returnValues[4])
	}
}

func TestExecStringEvaluatesMathFrexpAndLdexp(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local fraction, exponent = math.frexp(10)
return fraction, exponent, math.ldexp(fraction, exponent)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	expected := []float64{0.625, 4, 10}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber {
			t.Fatalf("unexpected math.frexp/math.ldexp return type at %d: %#v", index, returnValues[index])
		}

		got := returnValues[index].Data.(float64)
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("unexpected math.frexp/math.ldexp return value at %d: got %v want %v", index, got, want)
		}
	}
}

func TestExecStringEvaluatesMathFMod(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
return math.fmod(7.5, 2), math.fmod(-7.5, 2)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	expected := []float64{1.5, -1.5}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber {
			t.Fatalf("unexpected math.fmod return type at %d: %#v", index, returnValues[index])
		}

		got := returnValues[index].Data.(float64)
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("unexpected math.fmod return value at %d: got %v want %v", index, got, want)
		}
	}
}

func TestExecStringEvaluatesMathMod(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
return math.mod(7.5, 2), math.mod(-7.5, 2)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	expected := []float64{1.5, -1.5}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber {
			t.Fatalf("unexpected math.mod return type at %d: %#v", index, returnValues[index])
		}

		got := returnValues[index].Data.(float64)
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("unexpected math.mod return value at %d: got %v want %v", index, got, want)
		}
	}
}

func TestExecStringEvaluatesMathTan(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
return math.tan(0), math.tan(0.7853981633974483)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	expected := []float64{0, 1}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber {
			t.Fatalf("unexpected math.tan return type at %d: %#v", index, returnValues[index])
		}

		got := returnValues[index].Data.(float64)
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("unexpected math.tan return value at %d: got %v want %v", index, got, want)
		}
	}
}

func TestExecStringEvaluatesMathAtan(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
return math.atan(0), math.atan(1)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	expected := []float64{0, math.Pi / 4}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber {
			t.Fatalf("unexpected math.atan return type at %d: %#v", index, returnValues[index])
		}

		got := returnValues[index].Data.(float64)
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("unexpected math.atan return value at %d: got %v want %v", index, got, want)
		}
	}
}

func TestExecStringEvaluatesMathAtan2(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
return math.atan2(1, 1), math.atan2(1, -1)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	expected := []float64{math.Pi / 4, 3 * math.Pi / 4}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber {
			t.Fatalf("unexpected math.atan2 return type at %d: %#v", index, returnValues[index])
		}

		got := returnValues[index].Data.(float64)
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("unexpected math.atan2 return value at %d: got %v want %v", index, got, want)
		}
	}
}

func TestExecStringEvaluatesMathLog10(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
return math.log10(1000), math.log10(0.01)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	expected := []float64{3, -2}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber {
			t.Fatalf("unexpected math.log10 return type at %d: %#v", index, returnValues[index])
		}

		got := returnValues[index].Data.(float64)
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("unexpected math.log10 return value at %d: got %v want %v", index, got, want)
		}
	}
}

func TestExecStringEvaluatesMathLog10EdgeCases(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
return math.log10(1), math.log10(0), math.log10(-1)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || math.Abs(returnValues[0].Data.(float64)-0) > 1e-9 {
		t.Fatalf("unexpected math.log10(1) return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNumber || !math.IsInf(returnValues[1].Data.(float64), -1) {
		t.Fatalf("unexpected math.log10(0) return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeNumber || !math.IsNaN(returnValues[2].Data.(float64)) {
		t.Fatalf("unexpected math.log10(-1) return value: %#v", returnValues[2])
	}
}

func TestExecStringRejectsInvalidMathLog10Argument(t *testing.T) {
	state := NewState()

	err := state.ExecString(`
return math.log10("x")
`)
	if err == nil {
		t.Fatal("expected error from math.log10 with string argument")
	}
}

func TestExecStringEvaluatesMathSinh(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
return math.sinh(0), math.sinh(1)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	expected := []float64{0, math.Sinh(1)}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber {
			t.Fatalf("unexpected math.sinh return type at %d: %#v", index, returnValues[index])
		}

		got := returnValues[index].Data.(float64)
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("unexpected math.sinh return value at %d: got %v want %v", index, got, want)
		}
	}
}

func TestExecStringEvaluatesMathCosh(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
return math.cosh(0), math.cosh(1)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	expected := []float64{1, math.Cosh(1)}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber {
			t.Fatalf("unexpected math.cosh return type at %d: %#v", index, returnValues[index])
		}

		got := returnValues[index].Data.(float64)
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("unexpected math.cosh return value at %d: got %v want %v", index, got, want)
		}
	}
}

func TestExecStringEvaluatesMathTanh(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
return math.tanh(0), math.tanh(1)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	expected := []float64{0, math.Tanh(1)}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber {
			t.Fatalf("unexpected math.tanh return type at %d: %#v", index, returnValues[index])
		}

		got := returnValues[index].Data.(float64)
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("unexpected math.tanh return value at %d: got %v want %v", index, got, want)
		}
	}
}

func TestExecStringEvaluatesMathAsinAndAcos(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
return math.asin(0), math.asin(1), math.acos(1), math.acos(0)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	expected := []float64{0, math.Pi / 2, 0, math.Pi / 2}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber {
			t.Fatalf("unexpected math.asin/math.acos return type at %d: %#v", index, returnValues[index])
		}

		got := returnValues[index].Data.(float64)
		if math.Abs(got-want) > 1e-9 {
			t.Fatalf("unexpected math.asin/math.acos return value at %d: got %v want %v", index, got, want)
		}
	}
}

func TestExecStringEvaluatesClockMillis(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local start = clock_ms()
local sum = 0
for i = 1, 10 do
	sum = sum + i
end
local finish = clock_ms()
return start, finish, finish >= start, sum
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	for _, index := range []int{0, 1} {
		value := returnValues[index]
		if value.Type != ValueTypeNumber {
			t.Fatalf("unexpected clock_ms return value at %d: %#v", index, value)
		}
	}

	if returnValues[2].Type != ValueTypeBoolean || returnValues[2].Data != true {
		t.Fatalf("unexpected clock comparison value: %#v", returnValues[2])
	}

	if returnValues[3].Type != ValueTypeNumber || returnValues[3].Data != float64(55) {
		t.Fatalf("unexpected loop sum value: %#v", returnValues[3])
	}
}

func TestExecStringEvaluatesStringLibrary(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local b1, b2, b3 = string.byte("ABC", 1, 3)
local f1s, f1e = string.find("hello world", "lo")
local f2s, f2e = string.find("a.b.c", ".", 3)
local f3s, f3e = string.find("banana", "na", -4, true)
local f4s = string.find("banana", "zz")
local f5s, f5e = string.find("abc", "", 2)
local m1 = string.match("hello world", "lo")
local m2 = string.match("banana", "na", -4)
local m3 = string.match("banana", "zz")
local m4 = string.match("abc", "", 3)
local gm = {}
for part in string.gmatch("banana", "na") do
	table.insert(gm, part)
end
local gm2 = string.gmatch("banana", "na", 4)
local gm2a = gm2()
local gm2b = gm2()
local gm3 = {}
for part in string.gmatch("abc", "", 2) do
	table.insert(gm3, part)
end
local gf = {}
for part in string.gfind("banana", "na") do
	table.insert(gf, part)
end
local g1, g1n = string.gsub("banana", "na", "NA")
local g2, g2n = string.gsub("banana", "na", "NA", 1)
local g3, g3n = string.gsub("abc", "", ".")
local g4, g4n = string.gsub("banana", "zz", "NA")
local g5, g5n = string.gsub("banana", "na", function(m) return "<" .. m .. ">" end, 1)
local g6, g6n = string.gsub("banana", "na", { na = 7 })
local sf1 = string.format("hello %s %d", "lua", 51)
local sf2 = string.format("%q %f %%", "hi", 1.5)
local sf3 = string.format("%i", -7)
local sf4 = string.format("%c%c", 65, 66)
local sf5 = string.format("%x %X", 255, 255)
local sf6 = string.format("%o", 64)
local sf7 = string.format("%u", -1)
local sf8 = string.format("%e %E", 12.5, 12.5)
local sf9 = string.format("%g %G", 12345.5, 0.00125)
return string.len("AbCd"), string.lower("AbCd"), string.upper("AbCd"), string.sub("abcdef", 2, 4), string.sub("abcdef", -3, -1), string.rep("ha", 3), string.reverse("stressed"), string.byte("ABC", 2), b1, b2, b3, string.char(65, 66, 67), f1s, f1e, f2s, f2e, f3s, f3e, f4s, f5s, f5e, m1, m2, m3, m4, gm[1], gm[2], gm2a, gm2b, table.getn(gm3), gf[1], gf[2], sf1, sf2, sf3, sf4, sf5, sf6, sf7, sf8, sf9, g1, g1n, g2, g2n, g3, g3n, g4, g4n, g5, g5n, g6, g6n
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 53 {
		t.Fatalf("expected 53 return values, got %d", len(returnValues))
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

	expectedFindNumbers := map[int]float64{
		12: 4,
		13: 5,
		14: 4,
		15: 4,
		16: 3,
		17: 4,
		19: 2,
		20: 1,
	}
	for index, want := range expectedFindNumbers {
		value := returnValues[index]
		if value.Type != ValueTypeNumber || value.Data != want {
			t.Fatalf("unexpected string.find return value at %d: %#v", index, value)
		}
	}

	if returnValues[18].Type != ValueTypeNil {
		t.Fatalf("expected nil from missing string.find match, got %#v", returnValues[18])
	}

	if returnValues[21].Type != ValueTypeString || returnValues[21].Data != "lo" {
		t.Fatalf("unexpected string.match return value at 21: %#v", returnValues[21])
	}

	if returnValues[22].Type != ValueTypeString || returnValues[22].Data != "na" {
		t.Fatalf("unexpected string.match return value at 22: %#v", returnValues[22])
	}

	if returnValues[23].Type != ValueTypeNil {
		t.Fatalf("expected nil from missing string.match result, got %#v", returnValues[23])
	}

	if returnValues[24].Type != ValueTypeString || returnValues[24].Data != "" {
		t.Fatalf("unexpected empty string.match result: %#v", returnValues[24])
	}

	if returnValues[25].Type != ValueTypeString || returnValues[25].Data != "na" {
		t.Fatalf("unexpected first string.gmatch iteration result: %#v", returnValues[25])
	}

	if returnValues[26].Type != ValueTypeString || returnValues[26].Data != "na" {
		t.Fatalf("unexpected second string.gmatch iteration result: %#v", returnValues[26])
	}

	if returnValues[27].Type != ValueTypeString || returnValues[27].Data != "na" {
		t.Fatalf("unexpected direct string.gmatch iterator first result: %#v", returnValues[27])
	}

	if returnValues[28].Type != ValueTypeNil {
		t.Fatalf("expected nil from exhausted string.gmatch iterator, got %#v", returnValues[28])
	}

	if returnValues[29].Type != ValueTypeNumber || returnValues[29].Data != float64(1) {
		t.Fatalf("unexpected empty-pattern string.gmatch count: %#v", returnValues[29])
	}

	if returnValues[30].Type != ValueTypeString || returnValues[30].Data != "na" {
		t.Fatalf("unexpected first string.gfind iteration result: %#v", returnValues[30])
	}

	if returnValues[31].Type != ValueTypeString || returnValues[31].Data != "na" {
		t.Fatalf("unexpected second string.gfind iteration result: %#v", returnValues[31])
	}

	expectedFormats := map[int]string{
		32: "hello lua 51",
		33: `"hi" 1.500000 %`,
		34: "-7",
		35: "AB",
		36: "ff FF",
		37: "100",
		38: "4294967295",
		39: "1.250000e+01 1.250000E+01",
		40: "12345.5 0.00125",
	}
	for index, want := range expectedFormats {
		if returnValues[index].Type != ValueTypeString || returnValues[index].Data != want {
			t.Fatalf("unexpected string.format return value at %d: %#v", index, returnValues[index])
		}
	}

	if returnValues[41].Type != ValueTypeString || returnValues[41].Data != "baNANA" {
		t.Fatalf("unexpected string.gsub return value at 41: %#v", returnValues[41])
	}

	if returnValues[42].Type != ValueTypeNumber || returnValues[42].Data != float64(2) {
		t.Fatalf("unexpected string.gsub replacement count at 42: %#v", returnValues[42])
	}

	if returnValues[43].Type != ValueTypeString || returnValues[43].Data != "baNAna" {
		t.Fatalf("unexpected string.gsub return value at 43: %#v", returnValues[43])
	}

	if returnValues[44].Type != ValueTypeNumber || returnValues[44].Data != float64(1) {
		t.Fatalf("unexpected string.gsub replacement count at 44: %#v", returnValues[44])
	}

	if returnValues[45].Type != ValueTypeString || returnValues[45].Data != ".a.b.c." {
		t.Fatalf("unexpected empty-pattern string.gsub result: %#v", returnValues[45])
	}

	if returnValues[46].Type != ValueTypeNumber || returnValues[46].Data != float64(4) {
		t.Fatalf("unexpected empty-pattern string.gsub replacement count: %#v", returnValues[46])
	}

	if returnValues[47].Type != ValueTypeString || returnValues[47].Data != "banana" {
		t.Fatalf("unexpected missing-match string.gsub result: %#v", returnValues[47])
	}

	if returnValues[48].Type != ValueTypeNumber || returnValues[48].Data != float64(0) {
		t.Fatalf("unexpected missing-match string.gsub replacement count: %#v", returnValues[48])
	}

	if returnValues[49].Type != ValueTypeString || returnValues[49].Data != "ba<na>na" {
		t.Fatalf("unexpected function-replacer string.gsub result: %#v", returnValues[49])
	}

	if returnValues[50].Type != ValueTypeNumber || returnValues[50].Data != float64(1) {
		t.Fatalf("unexpected function-replacer string.gsub replacement count: %#v", returnValues[50])
	}

	if returnValues[51].Type != ValueTypeString || returnValues[51].Data != "ba77" {
		t.Fatalf("unexpected table-replacer string.gsub result: %#v", returnValues[51])
	}

	if returnValues[52].Type != ValueTypeNumber || returnValues[52].Data != float64(2) {
		t.Fatalf("unexpected table-replacer string.gsub replacement count: %#v", returnValues[52])
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

func TestExecStringEvaluatesGenericForWithCustomIteratorTriple(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local function step(state, control)
	local next_index = control + 1
	if next_index > state.limit then
		return nil
	end

	return next_index, state.values[next_index]
end

local total = 0
for index, value in step, { limit = 3, values = { 10, 20, 30 } }, 0 do
	total = total + index + value
end

return total
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(66) {
		t.Fatalf("unexpected custom iterator total: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesGenericForWithMetatableCallIterator(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local iterator = {}
setmetatable(iterator, {
	__call = function(_, state, control)
		local next_index = control + 1
		if next_index > state.limit then
			return nil
		end

		return next_index, state.values[next_index]
	end
})

local total = 0
for index, value in iterator, { limit = 3, values = { 5, 6, 7 } }, 0 do
	total = total + index * value
end

return total
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(38) {
		t.Fatalf("unexpected metatable iterator total: %#v", returnValues[0])
	}
}

func TestExecStringEvaluatesGenericForWithStringGMatch(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local parts = {}
for part in string.gmatch("aba", "a") do
	table.insert(parts, part)
end
return table.concat(parts, "|")
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "a|a" {
		t.Fatalf("unexpected gmatch generic-for result: %#v", returnValues[0])
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

func TestExecStringRejectsInvalidMetatableIndexTarget(t *testing.T) {
	state := NewState()

	err := state.ExecString(`
local target = {}
setmetatable(target, { __index = 1 })
return target.answer
`)
	if err == nil {
		t.Fatal("expected invalid __index target error")
	}

	if err.Error() != `execute compiled Lua source "<memory>": attempt to index non-table __index value of type number` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecStringRejectsInvalidMetatableNewIndexTarget(t *testing.T) {
	state := NewState()

	err := state.ExecString(`
local target = {}
setmetatable(target, { __newindex = 1 })
target.answer = 42
`)
	if err == nil {
		t.Fatal("expected invalid __newindex target error")
	}

	if err.Error() != `execute compiled Lua source "<memory>": attempt to index-assign non-table __newindex value of type number` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecStringRejectsMetatableIndexLoop(t *testing.T) {
	state := NewState()

	err := state.ExecString(`
local target = {}
local meta = {}
meta.__index = target
setmetatable(target, meta)
return target.answer
`)
	if err == nil {
		t.Fatal("expected __index loop error")
	}

	if err.Error() != `execute compiled Lua source "<memory>": loop in table __index chain` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecStringRejectsMetatableNewIndexLoop(t *testing.T) {
	state := NewState()

	err := state.ExecString(`
local target = {}
local meta = {}
meta.__newindex = target
setmetatable(target, meta)
target.answer = 42
`)
	if err == nil {
		t.Fatal("expected __newindex loop error")
	}

	if err.Error() != `execute compiled Lua source "<memory>": loop in table __newindex chain` {
		t.Fatalf("unexpected error: %v", err)
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

func TestExecStringRejectsMetatableCallLoop(t *testing.T) {
	state := NewState()

	err := state.ExecString(`
local target = {}
setmetatable(target, { __call = target })
return target()
`)
	if err == nil {
		t.Fatal("expected __call loop error")
	}

	if err.Error() != `execute compiled Lua source "<memory>": loop in table __call chain` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecStringRejectsMetatableCallCycle(t *testing.T) {
	state := NewState()

	err := state.ExecString(`
local a = {}
local b = {}
setmetatable(a, { __call = b })
setmetatable(b, { __call = a })
return a()
`)
	if err == nil {
		t.Fatal("expected __call cycle error")
	}

	if err.Error() != `execute compiled Lua source "<memory>": loop in table __call chain` {
		t.Fatalf("unexpected error: %v", err)
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
local meta = {
	__eq = function(_, _)
		return true
	end,
	__lt = function(_, _)
		return true
	end,
	__le = function(_, _)
		return false
	end
}
setmetatable(lhs, meta)
setmetatable(rhs, meta)
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

func TestExecStringRequiresSharedEqMetamethod(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local lhs = {}
local rhs = {}
setmetatable(lhs, {
	__eq = function(_, _)
		return true
	end
})
setmetatable(rhs, {})
return lhs == rhs, lhs ~= rhs
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != false {
		t.Fatalf("unexpected equality result without shared __eq: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeBoolean || returnValues[1].Data != true {
		t.Fatalf("unexpected inequality result without shared __eq: %#v", returnValues[1])
	}
}

func TestExecStringRequiresSharedOrderedMetamethod(t *testing.T) {
	state := NewState()

	err := state.ExecString(`
local lhs = {}
local rhs = {}
setmetatable(lhs, {
	__lt = function(_, _)
		return true
	end
})
setmetatable(rhs, {})
return lhs < rhs
`)
	if err == nil {
		t.Fatal("expected ordered comparison without shared __lt to fail")
	}

	if err.Error() != `execute compiled Lua source "<memory>": operator "<" expects number operand, got table` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecStringRequiresSharedOrderedMetamethodForLessEqual(t *testing.T) {
	state := NewState()

	err := state.ExecString(`
local lhs = {}
local rhs = {}
setmetatable(lhs, {
	__le = function(_, _)
		return true
	end
})
setmetatable(rhs, {})
return lhs <= rhs
`)
	if err == nil {
		t.Fatal("expected less-equal comparison without shared __le to fail")
	}

	if err.Error() != `execute compiled Lua source "<memory>": operator "<=" expects number operand, got table` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecStringEvaluatesLessEqualFallbackViaLessThanMetamethod(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local low = { score = 1 }
local high = { score = 3 }
local meta = {
	__lt = function(lhs, rhs)
		return lhs.score < rhs.score
	end
}
setmetatable(low, meta)
setmetatable(high, meta)
return low <= high, high <= low, high >= low, low >= high
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	expected := []bool{true, false, true, false}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeBoolean || returnValues[index].Data != want {
			t.Fatalf("unexpected fallback comparison result at %d: %#v", index, returnValues[index])
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

func TestExecStringEvaluatesTableAndFunctionIdentityKeys(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local store = {}
local table_key_a = {}
local table_key_b = {}

local function make_key()
	return function()
		return "same-body"
	end
end

local fn_key_a = make_key()
local fn_key_b = make_key()

store[table_key_a] = "table-a"
store[table_key_b] = "table-b"
rawset(store, fn_key_a, "fn-a")
rawset(store, fn_key_b, "fn-b")

return store[table_key_a], store[table_key_b], rawget(store, fn_key_a), rawget(store, fn_key_b)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "table-a" {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != "table-b" {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeString || returnValues[2].Data != "fn-a" {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}

	if returnValues[3].Type != ValueTypeString || returnValues[3].Data != "fn-b" {
		t.Fatalf("unexpected fourth return value: %#v", returnValues[3])
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
local sparse = { [1] = "x", [4] = "y" }
table.insert(sparse, "z")
local sparse_removed = table.remove(sparse)
return values[1], values[2], values[3], values[4], removed_mid, removed_last, sparse[1], sparse[4], sparse[5], sparse_removed
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 10 {
		t.Fatalf("expected 10 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(1) || returnValues[1].Data != float64(3) || returnValues[2].Type != ValueTypeNil || returnValues[3].Type != ValueTypeNil {
		t.Fatalf("unexpected table contents after insert/remove: %#v", returnValues[:4])
	}

	if returnValues[4].Data != float64(9) || returnValues[5].Data != float64(2) {
		t.Fatalf("unexpected removed values: %#v", returnValues[4:6])
	}

	if returnValues[6].Type != ValueTypeString || returnValues[6].Data != "x" {
		t.Fatalf("unexpected sparse first value after insert/remove: %#v", returnValues[6])
	}

	if returnValues[7].Type != ValueTypeString || returnValues[7].Data != "y" {
		t.Fatalf("unexpected sparse fourth value after insert/remove: %#v", returnValues[7])
	}

	if returnValues[8].Type != ValueTypeNil {
		t.Fatalf("expected sparse fifth slot to be cleared, got %#v", returnValues[8])
	}

	if returnValues[9].Type != ValueTypeString || returnValues[9].Data != "z" {
		t.Fatalf("unexpected sparse removed value: %#v", returnValues[9])
	}
}

func TestExecStringEvaluatesTableGetN(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local dense = { 10, 20, 30 }
local sparse = { [1] = "x", [2] = "y", [4] = "z" }
local nofirst = { [2] = "only" }
local empty = {}
return table.getn(dense), table.getn(sparse), table.getn(nofirst), table.getn(empty)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	expected := []float64{3, 4, 0, 0}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber || returnValues[index].Data != want {
			t.Fatalf("unexpected table.getn return value at %d: %#v", index, returnValues[index])
		}
	}
}

func TestExecStringEvaluatesTableForeachI(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local values = { 10, 20, 30 }
local sum = 0
local stop = table.foreachi(values, function(i, v)
	sum = sum + i + v
	if i == 2 then
		return "stop@" .. i
	end
end)
local empty = table.foreachi({}, function()
	return "never"
end)
local sparse_seen = ""
local sparse_stop = table.foreachi({ [1] = "x", [4] = "y" }, function(i, v)
	sparse_seen = sparse_seen .. i .. ":" .. v .. ";"
	if i == 4 then
		return "stop@" .. v
	end
end)
return sum, stop, empty, sparse_seen, sparse_stop
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 5 {
		t.Fatalf("expected 5 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(33) {
		t.Fatalf("unexpected table.foreachi accumulated sum: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != "stop@2" {
		t.Fatalf("unexpected table.foreachi early-stop value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeNil {
		t.Fatalf("expected nil from empty table.foreachi, got %#v", returnValues[2])
	}

	if returnValues[3].Type != ValueTypeString || returnValues[3].Data != "1:x;4:y;" {
		t.Fatalf("unexpected sparse table.foreachi traversal output: %#v", returnValues[3])
	}

	if returnValues[4].Type != ValueTypeString || returnValues[4].Data != "stop@y" {
		t.Fatalf("unexpected sparse table.foreachi early-stop value: %#v", returnValues[4])
	}
}

func TestExecStringEvaluatesTableForeachIWithMetatableCallCallback(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local callback = {}
local seen = ""
setmetatable(callback, {
	__call = function(_, i, v)
		seen = seen .. i .. "=" .. v .. ";"
		if i == 2 then
			return "stop@" .. v
		end
	end
})
return table.foreachi({ 10, 20, 30 }, callback), seen
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "stop@20" {
		t.Fatalf("unexpected metatable foreachi stop value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != "1=10;2=20;" {
		t.Fatalf("unexpected metatable foreachi traversal output: %#v", returnValues[1])
	}
}

func TestExecStringEvaluatesTableForeach(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local values = { answer = 42, [1] = 10, label = "lua" }
local seen = ""
local stop = table.foreach(values, function(k, v)
	seen = seen .. tostring(k) .. "=" .. tostring(v) .. ";"
	if k == "label" then
		return "stop@" .. v
	end
end)
local empty = table.foreach({}, function()
	return "never"
end)
return seen, stop, empty
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "answer=42;1=10;label=lua;" {
		t.Fatalf("unexpected table.foreach traversal output: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != "stop@lua" {
		t.Fatalf("unexpected table.foreach early-stop value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeNil {
		t.Fatalf("expected nil from empty table.foreach, got %#v", returnValues[2])
	}
}

func TestExecStringEvaluatesTableForeachWithMetatableCallCallback(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local callback = {}
local seen = ""
setmetatable(callback, {
	__call = function(_, k, v)
		seen = seen .. tostring(k) .. "=" .. tostring(v) .. ";"
		if k == "label" then
			return "stop@" .. v
		end
	end
})

local values = { answer = 42, label = "lua" }
return table.foreach(values, callback), seen
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "stop@lua" {
		t.Fatalf("unexpected metatable foreach stop value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != "answer=42;label=lua;" {
		t.Fatalf("unexpected metatable foreach traversal output: %#v", returnValues[1])
	}
}

func TestExecStringEvaluatesTableMaxN(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local mixed = { [1] = "x", [2] = "y", [4] = "z", answer = 42, [-3] = "neg" }
local empty = {}
return table.maxn(mixed), table.maxn(empty)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(4) {
		t.Fatalf("unexpected table.maxn return value for mixed table: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(0) {
		t.Fatalf("unexpected table.maxn return value for empty table: %#v", returnValues[1])
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
local tagged = {}
setmetatable(tagged, {
	__tostring = function()
		return "tagged"
	end
})
local mixed = { "a", tagged, true }
local sparse = { [1] = "a", [4] = "d" }
local sparse_ok, sparse_err = pcall(table.concat, sparse, "|")
return table.concat(values), table.concat(values, "-"), table.concat(values, "-", 2, 3), table.concat(values, "-", 5, 4), table.concat(mixed, "|"), sparse_ok, sparse_err
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 7 {
		t.Fatalf("expected 7 return values, got %d", len(returnValues))
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

	if returnValues[4].Type != ValueTypeString || returnValues[4].Data != "a|tagged|true" {
		t.Fatalf("unexpected stringable concat value: %#v", returnValues[4])
	}

	if returnValues[5].Type != ValueTypeBoolean || returnValues[5].Data != false {
		t.Fatalf("unexpected sparse concat status: %#v", returnValues[5])
	}

	if returnValues[6].Type != ValueTypeString || returnValues[6].Data != "table.concat encountered nil value" {
		t.Fatalf("unexpected sparse concat error: %#v", returnValues[6])
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

local callable_desc = { 3, 1, 2 }
local comparator = {}
setmetatable(comparator, {
	__call = function(_, a, b)
		return a > b
	end
})
table.sort(callable_desc, comparator)

local low = { score = 1 }
local high = { score = 3 }
local mid = { score = 2 }
local ranked = { high, low, mid }

local meta = {
	__lt = function(lhs, rhs)
		return lhs.score < rhs.score
	end
}

setmetatable(low, meta)
setmetatable(mid, meta)
setmetatable(high, meta)

table.sort(ranked)

local sparse = { [1] = 3, [4] = 1 }
local sparse_ok, sparse_err = pcall(table.sort, sparse)

return table.concat(ascending, ","), table.concat(descending, ","), table.concat(callable_desc, ","), ranked[1].score, ranked[2].score, ranked[3].score, sparse_ok, sparse_err
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 8 {
		t.Fatalf("expected 8 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "1,2,3" {
		t.Fatalf("unexpected ascending sort value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != "3,2,1" {
		t.Fatalf("unexpected descending sort value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeString || returnValues[2].Data != "3,2,1" {
		t.Fatalf("unexpected callable comparator sort value: %#v", returnValues[2])
	}

	if returnValues[3].Type != ValueTypeNumber || returnValues[3].Data != float64(1) {
		t.Fatalf("unexpected first ranked score: %#v", returnValues[3])
	}

	if returnValues[4].Type != ValueTypeNumber || returnValues[4].Data != float64(2) {
		t.Fatalf("unexpected second ranked score: %#v", returnValues[4])
	}

	if returnValues[5].Type != ValueTypeNumber || returnValues[5].Data != float64(3) {
		t.Fatalf("unexpected third ranked score: %#v", returnValues[5])
	}

	if returnValues[6].Type != ValueTypeBoolean || returnValues[6].Data != false {
		t.Fatalf("unexpected sparse sort status: %#v", returnValues[6])
	}

	if returnValues[7].Type != ValueTypeString || returnValues[7].Data != "table.sort encountered nil value" {
		t.Fatalf("unexpected sparse sort error: %#v", returnValues[7])
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

func TestExecSourceEvaluatesBuiltinRequireRelativeModule(t *testing.T) {
	state := NewState()
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.lua")
	modulePath := filepath.Join(tempDir, "helper.lua")

	if err := os.WriteFile(modulePath, []byte(`
load_count = (load_count or 0) + 1
return { answer = 42 }
`), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	if err := state.ExecSource(Source{
		Name: mainPath,
		Content: `
local first = require("helper")
local second = require("helper")
return first.answer, rawequal(first, second), load_count
`,
	}); err != nil {
		t.Fatalf("exec source: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(42) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeBoolean || returnValues[1].Data != true {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeNumber || returnValues[2].Data != float64(1) {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}
}

func TestExecSourceRequireReturnsTrueWhenModuleReturnsNothing(t *testing.T) {
	state := NewState()
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.lua")
	modulePath := filepath.Join(tempDir, "flag.lua")

	if err := os.WriteFile(modulePath, []byte(`
side_effect = "loaded"
`), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	if err := state.ExecSource(Source{
		Name: mainPath,
		Content: `
local first = require("flag")
local second = require("flag")
return first, second, side_effect
`,
	}); err != nil {
		t.Fatalf("exec source: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != true {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeBoolean || returnValues[1].Data != true {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeString || returnValues[2].Data != "loaded" {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}
}

func TestExecSourceRejectsRequireLoop(t *testing.T) {
	state := NewState()
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.lua")
	firstPath := filepath.Join(tempDir, "first.lua")
	secondPath := filepath.Join(tempDir, "second.lua")

	if err := os.WriteFile(firstPath, []byte(`
return require("second")
`), 0o644); err != nil {
		t.Fatalf("write first module: %v", err)
	}

	if err := os.WriteFile(secondPath, []byte(`
return require("first")
`), 0o644); err != nil {
		t.Fatalf("write second module: %v", err)
	}

	err := state.ExecSource(Source{
		Name: mainPath,
		Content: `
return require("first")
`,
	})
	if err == nil {
		t.Fatal("expected require loop error")
	}

	if err.Error() != `execute compiled Lua source "`+mainPath+`": execute compiled Lua source "`+firstPath+`": execute compiled Lua source "`+secondPath+`": loop in require chain for module "first"` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecSourceRequireRespectsPackageLoaded(t *testing.T) {
	state := NewState()
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.lua")
	modulePath := filepath.Join(tempDir, "helper.lua")

	if err := os.WriteFile(modulePath, []byte(`
load_count = (load_count or 0) + 1
return { answer = 99 }
`), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	if err := state.ExecSource(Source{
		Name: mainPath,
		Content: `
package.loaded.helper = { answer = 7 }
local mod = require("helper")
return mod.answer, load_count, package.loaded.helper.answer
`,
	}); err != nil {
		t.Fatalf("exec source: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(7) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNil {
		t.Fatalf("expected second return value to be nil, got %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeNumber || returnValues[2].Data != float64(7) {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}
}

func TestExecSourceRequireRespectsPackagePath(t *testing.T) {
	state := NewState()
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.lua")
	moduleDir := filepath.Join(tempDir, "lib")
	modulePath := filepath.Join(moduleDir, "nested.lua")

	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir module dir: %v", err)
	}

	if err := os.WriteFile(modulePath, []byte(`
return { answer = 55 }
`), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	if err := state.ExecSource(Source{
		Name: mainPath,
		Content: `
package.path = "lib/?.lua"
local mod = require("nested")
return mod.answer, package.path
`,
	}); err != nil {
		t.Fatalf("exec source: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(55) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != "lib/?.lua" {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

func TestExecStringRequireRespectsPackagePreload(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
package.preload.helper = function(name)
	package.loaded._helper_load_count = (package.loaded._helper_load_count or 0) + 1
	return { module_name = name, answer = 88 }
end
local first = require("helper")
local second = require("helper")
return first.module_name, first.answer, rawequal(first, second), package.loaded._helper_load_count
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "helper" {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNumber || returnValues[1].Data != float64(88) {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeBoolean || returnValues[2].Data != true {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}

	if returnValues[3].Type != ValueTypeNumber || returnValues[3].Data != float64(1) {
		t.Fatalf("unexpected fourth return value: %#v", returnValues[3])
	}
}

func TestExecStringRequirePreloadRespectsCurrentFunctionEnvironment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local env = { answer = 41 }
setmetatable(env, { __index = _G })

local function install_helper()
	setfenv(install_helper, env)
	package.preload.helper = function(name)
		local mod = module("preloaded.module", package.seeall)
		answer = answer + 1
		return mod
	end
end

install_helper()
local mod = require("helper")
return mod.answer, rawequal(env.preloaded.module, mod), rawget(_G, "preloaded"), env.answer
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(42) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeBoolean || returnValues[1].Data != true {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeNil {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}

	if returnValues[3].Type != ValueTypeNumber || returnValues[3].Data != float64(41) {
		t.Fatalf("unexpected fourth return value: %#v", returnValues[3])
	}
}

func TestExecStringRejectsRequirePreloadLoop(t *testing.T) {
	state := NewState()

	err := state.ExecString(`
package.preload.helper = function()
	return require("helper")
end
return require("helper")
`)
	if err == nil {
		t.Fatal("expected preload require loop error")
	}

	if err.Error() != `execute compiled Lua source "<memory>": loop in require chain for module "helper"` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecStringRequireUsesCustomPackageLoader(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
table.insert(package.loaders, 1, function(name)
	return function(name)
		return { answer = 73, name = name }
	end
end)
local mod = require("virtual")
return mod.answer, mod.name
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(73) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeString || returnValues[1].Data != "virtual" {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}
}

func TestExecStringCustomPackageLoaderRespectsCurrentFunctionEnvironment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local env = { answer = 41 }
setmetatable(env, { __index = _G })

local function install_loader()
	setfenv(install_loader, env)
	table.insert(package.loaders, 1, function(name)
		if name ~= "virtual_env" then
			return "\n\tno virtual env loader"
		end

		return function(name)
			local mod = module("virtual.loader", package.seeall)
			answer = answer + 1
			return mod
		end
	end)
end

install_loader()
local mod = require("virtual_env")
return mod.answer, rawequal(env.virtual.loader, mod), rawget(_G, "virtual"), env.answer
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeNumber || returnValues[0].Data != float64(42) {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeBoolean || returnValues[1].Data != true {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeNil {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}

	if returnValues[3].Type != ValueTypeNumber || returnValues[3].Data != float64(41) {
		t.Fatalf("unexpected fourth return value: %#v", returnValues[3])
	}
}

func TestExecSourceRequireReloadsAfterClearingPackageLoaded(t *testing.T) {
	state := NewState()
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.lua")
	modulePath := filepath.Join(tempDir, "reload.lua")

	if err := os.WriteFile(modulePath, []byte(`
package.loaded._reload_count = (package.loaded._reload_count or 0) + 1
return package.loaded._reload_count
`), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	if err := state.ExecSource(Source{
		Name: mainPath,
		Content: `
local first = require("reload")
package.loaded.reload = nil
local second = require("reload")
return first, second, package.loaded.reload, package.loaded._reload_count
`,
	}); err != nil {
		t.Fatalf("exec source: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	expected := []float64{1, 2, 2, 2}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber || returnValues[index].Data != want {
			t.Fatalf("unexpected reload return value at %d: %#v", index, returnValues[index])
		}
	}
}

func TestExecSourceRequireReloadsAfterPackageLoadedFalse(t *testing.T) {
	state := NewState()
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.lua")
	modulePath := filepath.Join(tempDir, "false_reload.lua")

	if err := os.WriteFile(modulePath, []byte(`
package.loaded._false_reload_count = (package.loaded._false_reload_count or 0) + 1
return package.loaded._false_reload_count
`), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	if err := state.ExecSource(Source{
		Name: mainPath,
		Content: `
local first = require("false_reload")
package.loaded.false_reload = false
local second = require("false_reload")
return first, second, package.loaded.false_reload, package.loaded._false_reload_count
`,
	}); err != nil {
		t.Fatalf("exec source: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	expected := []float64{1, 2, 2, 2}
	for index, want := range expected {
		if returnValues[index].Type != ValueTypeNumber || returnValues[index].Data != want {
			t.Fatalf("unexpected false-reload return value at %d: %#v", index, returnValues[index])
		}
	}
}

func TestExecSourceEvaluatesPackageSearchPath(t *testing.T) {
	state := NewState()
	tempDir := t.TempDir()
	mainPath := filepath.Join(tempDir, "main.lua")
	moduleDir := filepath.Join(tempDir, "pkg")
	modulePath := filepath.Join(moduleDir, "tool.lua")

	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir module dir: %v", err)
	}

	if err := os.WriteFile(modulePath, []byte(`return 1`), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	if err := state.ExecSource(Source{
		Name: mainPath,
		Content: `
local found = package.searchpath("tool", "pkg/?.lua")
local missing, err = package.searchpath("missing", "pkg/?.lua")
return found, missing, err
`,
	}); err != nil {
		t.Fatalf("exec source: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != modulePath {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeNil {
		t.Fatalf("expected second return value to be nil, got %#v", returnValues[1])
	}

	expectedError := "\n\tno file '" + filepath.Join(tempDir, "pkg", "missing.lua") + "'\n\tno file 'pkg/missing.lua'"
	if returnValues[2].Type != ValueTypeString || returnValues[2].Data != expectedError {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}
}

func TestExecStringRequirePromotesFalseLoaderResultToTrue(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
package.preload.falsemod = function()
	return false
end
local first = require("falsemod")
local second = package.loaded.falsemod
return first, second
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	for index := 0; index < 2; index++ {
		if returnValues[index].Type != ValueTypeBoolean || returnValues[index].Data != true {
			t.Fatalf("unexpected promoted false-loader value at %d: %#v", index, returnValues[index])
		}
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
local sparse = { [1] = "x", [4] = "y" }
local a, b, c, d = unpack(values)
local e, f = unpack(values, 2, 3)
local g, h = unpack(values, 5, 4)
local i, j, k, l = unpack(sparse)
return a, b, c, d, e, f, g, h, i, j, k, l
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 12 {
		t.Fatalf("expected 12 return values, got %d", len(returnValues))
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

	if returnValues[8].Type != ValueTypeString || returnValues[8].Data != "x" ||
		returnValues[9].Type != ValueTypeNil ||
		returnValues[10].Type != ValueTypeNil ||
		returnValues[11].Type != ValueTypeString || returnValues[11].Data != "y" {
		t.Fatalf("unexpected unpack(sparse) values: %#v", returnValues[8:12])
	}
}

func TestExecStringEvaluatesBuiltinSelectMultivalueAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pack(a, b, c, d)
	return a, b, c, d
end
function capture(...)
	local a, b, c = pack(select(2, ...))
	local d, e, f, g = pack(0, select(2, ...))
	return a, b, c, d, e, f, g
end
return capture(4, 5, 6)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 7 {
		t.Fatalf("expected 7 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(5) || returnValues[1].Data != float64(6) || returnValues[2].Type != ValueTypeNil {
		t.Fatalf("unexpected select call-arg values: %#v", returnValues[:3])
	}

	if returnValues[3].Data != float64(0) || returnValues[4].Data != float64(5) || returnValues[5].Data != float64(6) || returnValues[6].Type != ValueTypeNil {
		t.Fatalf("unexpected mixed select call-arg values: %#v", returnValues[3:7])
	}
}

func TestExecStringEvaluatesSelectTableConstructorAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function capture(...)
	local a = { 0, select(2, ...) }
	local b = { 0, (select(2, ...)) }
	return a[1], a[2], a[3], a[4], b[1], b[2], b[3]
end

return capture(4, 5, 6)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 7 {
		t.Fatalf("expected 7 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(0) || returnValues[1].Data != float64(5) || returnValues[2].Data != float64(6) || returnValues[3].Type != ValueTypeNil {
		t.Fatalf("unexpected select table values: %#v", returnValues[:4])
	}

	if returnValues[4].Data != float64(0) || returnValues[5].Data != float64(5) || returnValues[6].Type != ValueTypeNil {
		t.Fatalf("unexpected parenthesized select table values: %#v", returnValues[4:7])
	}
}

func TestExecStringEvaluatesBuiltinUnpackMultivalueAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local values = { 10, 20, 30 }
function pack(a, b, c, d)
	return a, b, c, d
end
local a, b, c = unpack(values)
local d, e, f, g = 0, unpack(values)
local t = { 0, unpack(values) }
local u = { (unpack(values)) }
local v = { 0, (unpack(values)) }
local p, q, r, s = pack(unpack(values))
return a, b, c, d, e, f, g, t[1], t[2], t[3], t[4], u[1], u[2], v[1], v[2], v[3], p, q, r, s
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 20 {
		t.Fatalf("expected 20 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(10) || returnValues[1].Data != float64(20) || returnValues[2].Data != float64(30) {
		t.Fatalf("unexpected direct unpack assignment values: %#v", returnValues[:3])
	}

	if returnValues[3].Data != float64(0) || returnValues[4].Data != float64(10) || returnValues[5].Data != float64(20) || returnValues[6].Data != float64(30) {
		t.Fatalf("unexpected mixed unpack assignment values: %#v", returnValues[3:7])
	}

	if returnValues[7].Data != float64(0) || returnValues[8].Data != float64(10) || returnValues[9].Data != float64(20) || returnValues[10].Data != float64(30) {
		t.Fatalf("unexpected unpack table values: %#v", returnValues[7:11])
	}

	if returnValues[11].Data != float64(10) || returnValues[12].Type != ValueTypeNil {
		t.Fatalf("unexpected parenthesized unpack table values: %#v", returnValues[11:13])
	}

	if returnValues[13].Data != float64(0) || returnValues[14].Data != float64(10) || returnValues[15].Type != ValueTypeNil {
		t.Fatalf("unexpected prefixed parenthesized unpack table values: %#v", returnValues[13:16])
	}

	if returnValues[16].Data != float64(10) || returnValues[17].Data != float64(20) || returnValues[18].Data != float64(30) || returnValues[19].Type != ValueTypeNil {
		t.Fatalf("unexpected unpack call-arg values: %#v", returnValues[16:20])
	}
}

func TestExecStringEvaluatesNativeReturnListMultivalueAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local values = { 10, 20, 30 }
function pair()
	return 1, 2
end
function handler(err)
	return "handled:" .. err, "extra"
end
function fail()
	error("boom")
end
return 0, unpack(values), select(2, pair()), xpcall(fail, handler)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 6 {
		t.Fatalf("expected 6 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(0) || returnValues[1].Data != float64(10) || returnValues[2].Data != float64(2) {
		t.Fatalf("unexpected native return-list prefix values: %#v", returnValues[:3])
	}

	if returnValues[3].Type != ValueTypeBoolean || returnValues[3].Data != false || returnValues[4].Type != ValueTypeString || returnValues[4].Data != "handled:boom" || returnValues[5].Type != ValueTypeString || returnValues[5].Data != "extra" {
		t.Fatalf("unexpected native return-list suffix values: %#v", returnValues[3:6])
	}
}

func TestExecStringEvaluatesNativeReturnListMixedMultivalueAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local values = { 10, 20, 30 }
function pair()
	return 1, 2
end
function handler(err)
	return "handled:" .. err, "extra"
end
function fail()
	error("boom")
end
return unpack(values), select(2, pair()), xpcall(fail, handler)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 5 {
		t.Fatalf("expected 5 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(10) {
		t.Fatalf("unexpected first native mixed return value: %#v", returnValues[0])
	}

	if returnValues[1].Data != float64(2) {
		t.Fatalf("unexpected second native mixed return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeBoolean || returnValues[2].Data != false {
		t.Fatalf("unexpected third native mixed return value: %#v", returnValues[2])
	}

	if returnValues[3].Type != ValueTypeString || returnValues[3].Data != "handled:boom" {
		t.Fatalf("unexpected fourth native mixed return value: %#v", returnValues[3])
	}

	if returnValues[4].Type != ValueTypeString || returnValues[4].Data != "extra" {
		t.Fatalf("unexpected fifth native mixed return value: %#v", returnValues[4])
	}
}

func TestExecStringSuppressesExpansionForParenthesizedNativeMultivalue(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local values = { 10, 20, 30 }
function pair()
	return 1, 2
end
function handler(err)
	return "handled:" .. err, "extra"
end
function fail()
	error("boom")
end
return (unpack(values)), (select(2, pair())), (xpcall(fail, handler))
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(10) || returnValues[1].Data != float64(2) {
		t.Fatalf("unexpected parenthesized native values: %#v", returnValues[:2])
	}

	if returnValues[2].Type != ValueTypeBoolean || returnValues[2].Data != false {
		t.Fatalf("unexpected parenthesized xpcall value: %#v", returnValues[2])
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

func TestExecStringEvaluatesBuiltinAssertMultivalueAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pack(a, b, c)
	return a, b, c
end
local a, b = assert(true, "ok")
local c, d = assert(true, "ok"), 4
local e, f, g = pack(assert(true, "ok"))
local h, i, j = pack(0, assert(true, "ok"))
return a, b, c, d, e, f, g, h, i, j
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 10 {
		t.Fatalf("expected 10 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != true || returnValues[1].Type != ValueTypeString || returnValues[1].Data != "ok" {
		t.Fatalf("unexpected direct assert values: %#v", returnValues[:2])
	}

	if returnValues[2].Type != ValueTypeBoolean || returnValues[2].Data != true || returnValues[3].Data != float64(4) {
		t.Fatalf("unexpected non-final assert assignment values: %#v", returnValues[2:4])
	}

	if returnValues[4].Type != ValueTypeBoolean || returnValues[4].Data != true || returnValues[5].Type != ValueTypeString || returnValues[5].Data != "ok" || returnValues[6].Type != ValueTypeNil {
		t.Fatalf("unexpected direct assert call-arg values: %#v", returnValues[4:7])
	}

	if returnValues[7].Data != float64(0) || returnValues[8].Type != ValueTypeBoolean || returnValues[8].Data != true || returnValues[9].Type != ValueTypeString || returnValues[9].Data != "ok" {
		t.Fatalf("unexpected final assert call-arg values: %#v", returnValues[7:10])
	}
}

func TestExecStringEvaluatesBuiltinNextAndPairsMultivalueAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local tbl = { answer = 42 }
function pack(a, b, c, d)
	return a, b, c, d
end
local a, b = next(tbl)
local c, d = next(tbl), 4
local e, f, g, h = pack(next(tbl))
local i, j, k, l = pack(0, next(tbl))
local iterator, state_value, control = pairs(tbl)
return a, b, c, d, e, f, g, h, i, j, k, l, type(iterator), state_value.answer, control
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 15 {
		t.Fatalf("expected 15 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "answer" || returnValues[1].Data != float64(42) {
		t.Fatalf("unexpected direct next values: %#v", returnValues[:2])
	}

	if returnValues[2].Type != ValueTypeString || returnValues[2].Data != "answer" || returnValues[3].Data != float64(4) {
		t.Fatalf("unexpected non-final next assignment values: %#v", returnValues[2:4])
	}

	if returnValues[4].Type != ValueTypeString || returnValues[4].Data != "answer" || returnValues[5].Data != float64(42) || returnValues[6].Type != ValueTypeNil || returnValues[7].Type != ValueTypeNil {
		t.Fatalf("unexpected direct next call-arg values: %#v", returnValues[4:8])
	}

	if returnValues[8].Data != float64(0) || returnValues[9].Type != ValueTypeString || returnValues[9].Data != "answer" || returnValues[10].Data != float64(42) || returnValues[11].Type != ValueTypeNil {
		t.Fatalf("unexpected final next call-arg values: %#v", returnValues[8:12])
	}

	if returnValues[12].Type != ValueTypeString || returnValues[12].Data != "function" || returnValues[13].Data != float64(42) || returnValues[14].Type != ValueTypeNil {
		t.Fatalf("unexpected pairs return values: %#v", returnValues[12:15])
	}
}

func TestExecStringEvaluatesAssertAndNextTableConstructorAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local tbl = { answer = 42 }
local a = { 0, assert(true, "ok") }
local b = { 0, next(tbl) }
local c = { 0, (assert(true, "ok")) }
local d = { 0, (next(tbl)) }
return a[1], a[2], a[3], b[1], b[2], b[3], c[1], c[2], c[3], d[1], d[2], d[3]
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 12 {
		t.Fatalf("expected 12 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(0) || returnValues[1].Type != ValueTypeBoolean || returnValues[1].Data != true || returnValues[2].Type != ValueTypeString || returnValues[2].Data != "ok" {
		t.Fatalf("unexpected assert table values: %#v", returnValues[:3])
	}

	if returnValues[3].Data != float64(0) || returnValues[4].Type != ValueTypeString || returnValues[4].Data != "answer" || returnValues[5].Data != float64(42) {
		t.Fatalf("unexpected next table values: %#v", returnValues[3:6])
	}

	if returnValues[6].Data != float64(0) || returnValues[7].Type != ValueTypeBoolean || returnValues[7].Data != true || returnValues[8].Type != ValueTypeNil {
		t.Fatalf("unexpected parenthesized assert table values: %#v", returnValues[6:9])
	}

	if returnValues[9].Data != float64(0) || returnValues[10].Type != ValueTypeString || returnValues[10].Data != "answer" || returnValues[11].Type != ValueTypeNil {
		t.Fatalf("unexpected parenthesized next table values: %#v", returnValues[9:12])
	}
}

func TestExecStringEvaluatesBuiltinIPairsReturnAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local iterator, state_value, control = ipairs({ 10, 20 })
return type(iterator), state_value[1], state_value[2], control
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeString || returnValues[0].Data != "function" {
		t.Fatalf("unexpected ipairs iterator type: %#v", returnValues[0])
	}

	if returnValues[1].Data != float64(10) || returnValues[2].Data != float64(20) || returnValues[3].Data != float64(0) {
		t.Fatalf("unexpected ipairs return values: %#v", returnValues[1:])
	}
}

func TestExecStringEvaluatesPairsAndIPairsTableConstructorAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local a = { 0, pairs({ answer = 42 }) }
local b = { 0, ipairs({ 10, 20 }) }
local c = { 0, (pairs({ answer = 42 })) }
local d = { 0, (ipairs({ 10, 20 })) }
return a[1], type(a[2]), a[3].answer, a[4], b[1], type(b[2]), b[3][1], b[4], c[1], type(c[2]), c[3], d[1], type(d[2]), d[3]
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 14 {
		t.Fatalf("expected 14 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(0) || returnValues[1].Type != ValueTypeString || returnValues[1].Data != "function" || returnValues[2].Data != float64(42) || returnValues[3].Type != ValueTypeNil {
		t.Fatalf("unexpected pairs table values: %#v", returnValues[:4])
	}

	if returnValues[4].Data != float64(0) || returnValues[5].Type != ValueTypeString || returnValues[5].Data != "function" || returnValues[6].Data != float64(10) || returnValues[7].Data != float64(0) {
		t.Fatalf("unexpected ipairs table values: %#v", returnValues[4:8])
	}

	if returnValues[8].Data != float64(0) || returnValues[9].Type != ValueTypeString || returnValues[9].Data != "function" || returnValues[10].Type != ValueTypeNil {
		t.Fatalf("unexpected parenthesized pairs table values: %#v", returnValues[8:11])
	}

	if returnValues[11].Data != float64(0) || returnValues[12].Type != ValueTypeString || returnValues[12].Data != "function" || returnValues[13].Type != ValueTypeNil {
		t.Fatalf("unexpected parenthesized ipairs table values: %#v", returnValues[11:14])
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

func TestExecStringEvaluatesTableLengthOperatorForBoundary(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local a = #{ 10, 20, 30 }
local t = { [1] = "x", [2] = "y", [4] = "z" }
local b = #t
return a, b
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(3) || returnValues[1].Data != float64(4) {
		t.Fatalf("unexpected table length values: %#v", returnValues)
	}
}

func TestExecStringEvaluatesTableLengthOperatorForEmptyAndSparsePrefix(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local empty = {}
local sparse = { [2] = "x" }
return #empty, #sparse
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(0) || returnValues[1].Data != float64(0) {
		t.Fatalf("unexpected empty/sparse table lengths: %#v", returnValues)
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

func TestExecStringEvaluatesProtectedCallTableConstructorAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pair()
	return 1, 2
end
function fail()
	error("boom")
end
function handler(err)
	return "handled:" .. err
end

local a = { 0, pcall(pair) }
local b = { 0, xpcall(fail, handler) }
local c = { 0, (pcall(pair)) }
local d = { 0, (xpcall(fail, handler)) }
return a[1], a[2], a[3], a[4], b[1], b[2], b[3], c[1], c[2], c[3], d[1], d[2], d[3]
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 13 {
		t.Fatalf("expected 13 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(0) || returnValues[1].Type != ValueTypeBoolean || returnValues[1].Data != true || returnValues[2].Data != float64(1) || returnValues[3].Data != float64(2) {
		t.Fatalf("unexpected pcall table values: %#v", returnValues[:4])
	}

	if returnValues[4].Data != float64(0) || returnValues[5].Type != ValueTypeBoolean || returnValues[5].Data != false || returnValues[6].Type != ValueTypeString || returnValues[6].Data != "handled:boom" {
		t.Fatalf("unexpected xpcall table values: %#v", returnValues[4:7])
	}

	if returnValues[7].Data != float64(0) || returnValues[8].Type != ValueTypeBoolean || returnValues[8].Data != true || returnValues[9].Type != ValueTypeNil {
		t.Fatalf("unexpected parenthesized pcall table values: %#v", returnValues[7:10])
	}

	if returnValues[10].Data != float64(0) || returnValues[11].Type != ValueTypeBoolean || returnValues[11].Data != false || returnValues[12].Type != ValueTypeNil {
		t.Fatalf("unexpected parenthesized xpcall table values: %#v", returnValues[10:13])
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

func TestExecStringEvaluatesReturnListMixedMultivalueAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pair()
	return 1, 2
end
function capture(...)
	return pair(), ..., pair()
end
return capture(7, 8)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(1) || returnValues[1].Data != float64(7) || returnValues[2].Data != float64(1) || returnValues[3].Data != float64(2) {
		t.Fatalf("unexpected mixed return-list values: %#v", returnValues)
	}
}

func TestExecStringEvaluatesNestedCallArgumentMixedMultivalueAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pair()
	return 1, 2
end
function pack(a, b, c, d)
	return a, b, c, d
end
function capture(...)
	return pack(pair(), ..., pair())
end
return capture(7, 8)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(1) || returnValues[1].Data != float64(7) || returnValues[2].Data != float64(1) || returnValues[3].Data != float64(2) {
		t.Fatalf("unexpected nested call-argument values: %#v", returnValues)
	}
}

func TestExecStringEvaluatesGenericForIteratorMultivalueAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function iter()
	return next, { 10, 20 }, nil, "ignored"
end
local total = 0
for _, value in iter() do
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

	if returnValues[0].Data != float64(30) {
		t.Fatalf("unexpected generic for total: %#v", returnValues[0])
	}
}

// TestExecStringEvaluatesVarargGenericForIteratorExpressionAdjustment 验证 `...` 作为 generic for 最后迭代表达式时会按 Lua 规则展开成迭代器三元组。
func TestExecStringEvaluatesVarargGenericForIteratorExpressionAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function capture(...)
	local total = 0
	for _, value in ... do
		total = total + value
	end
	return total
end

return capture(next, { 9, 10 }, nil)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 1 {
		t.Fatalf("expected 1 return value, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(19) {
		t.Fatalf("unexpected vararg generic for total: %#v", returnValues[0])
	}
}

// TestExecStringEvaluatesBuiltinGenericForIteratorExpressionAdjustment 验证 builtin 多返回值在 generic for 迭代表达式列表末尾会按 Lua 规则展开。
func TestExecStringEvaluatesBuiltinGenericForIteratorExpressionAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local total_assert = 0
for _, value in assert(next, { 10, 20 }, nil) do
	total_assert = total_assert + value
end

local total_unpack = 0
for _, value in unpack({ next, { 3, 4 }, nil }) do
	total_unpack = total_unpack + value
end

local total_select = 0
for _, value in select(2, false, next, { 5, 6 }, nil) do
	total_select = total_select + value
end

return total_assert, total_unpack, total_select
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(30) {
		t.Fatalf("unexpected assert generic for total: %#v", returnValues[0])
	}

	if returnValues[1].Data != float64(7) {
		t.Fatalf("unexpected unpack generic for total: %#v", returnValues[1])
	}

	if returnValues[2].Data != float64(11) {
		t.Fatalf("unexpected select generic for total: %#v", returnValues[2])
	}
}

// TestExecStringEvaluatesProtectedCallGenericForIteratorExpressionAdjustment 验证 protected call 的结果经 `select(2, ...)` 调整后可作为 generic for 迭代器三元组使用。
func TestExecStringEvaluatesProtectedCallGenericForIteratorExpressionAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function iter()
	return next, { 7, 8 }, nil
end

function handler(err)
	return "handled:" .. err
end

local total_pcall = 0
for _, value in select(2, pcall(iter)) do
	total_pcall = total_pcall + value
end

local total_xpcall = 0
for _, value in select(2, xpcall(iter, handler)) do
	total_xpcall = total_xpcall + value
end

return total_pcall, total_xpcall
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 2 {
		t.Fatalf("expected 2 return values, got %d", len(returnValues))
	}

	if returnValues[0].Data != float64(15) {
		t.Fatalf("unexpected pcall generic for total: %#v", returnValues[0])
	}

	if returnValues[1].Data != float64(15) {
		t.Fatalf("unexpected xpcall generic for total: %#v", returnValues[1])
	}
}

// TestExecStringEvaluatesParenthesizedBuiltinInGenericForIteratorList 验证圆括号会抑制 builtin / protected call 在 generic for 迭代表达式列表里的多返回值展开。
func TestExecStringEvaluatesParenthesizedBuiltinInGenericForIteratorList(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
for _, value in (assert(next, { 10, 20 }, nil)) do
	return value
end
return "done"
`); err == nil {
		t.Fatal("expected parenthesized assert iterator list to fail")
	} else if err.Error() != `execute compiled Lua source "<memory>": next expects table argument` {
		t.Fatalf("unexpected assert iterator error: %v", err)
	}

	state = NewState()
	if err := state.ExecString(`
function iter()
	return next, { 10, 20 }, nil
end

for _, value in (select(2, pcall(iter))) do
	return value
end
return "done"
`); err == nil {
		t.Fatal("expected parenthesized protected-call iterator list to fail")
	} else if err.Error() != `execute compiled Lua source "<memory>": next expects table argument` {
		t.Fatalf("unexpected protected-call iterator error: %v", err)
	}

	state = NewState()
	if err := state.ExecString(`
function capture(...)
	for _, value in (...) do
		return value
	end
	return "done"
end

return capture(next, { 10, 20 }, nil)
`); err == nil {
		t.Fatal("expected parenthesized vararg iterator list to fail")
	} else if err.Error() != `execute compiled Lua source "<memory>": next expects table argument` {
		t.Fatalf("unexpected vararg iterator error: %v", err)
	}
}

func TestExecStringEvaluatesParenthesizedCallInGenericForIteratorList(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function iter()
	return next, { 10, 20 }, nil
end
for _, value in (iter()) do
	return value
end
return "done"
`); err == nil {
		t.Fatal("expected parenthesized iterator list to fail")
	} else if err.Error() != `execute compiled Lua source "<memory>": next expects table argument` {
		t.Fatalf("unexpected error: %v", err)
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

func TestExecStringEvaluatesBuiltinPCallWithMetatableCall(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local callable = {}
setmetatable(callable, {
	__call = function(self, left, right)
		return self == callable, left + right, left * right
	end
})
return pcall(callable, 3, 4)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 4 {
		t.Fatalf("expected 4 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != true {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeBoolean || returnValues[1].Data != true {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeNumber || returnValues[2].Data != float64(7) {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}

	if returnValues[3].Type != ValueTypeNumber || returnValues[3].Data != float64(12) {
		t.Fatalf("unexpected fourth return value: %#v", returnValues[3])
	}
}

func TestExecStringEvaluatesXPCAllMultivalueAdjustment(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
function pair()
	return 1, 2
end
function fail()
	error("boom")
end
function handler(err)
	return "handled:" .. err, "extra"
end
function pack(a, b, c, d)
	return a, b, c, d
end
local a, b, c = xpcall(pair, handler)
local d, e, f = xpcall(pair, handler), 4
local g, h, i, j = 0, xpcall(pair, handler)
local k, l, m, n = pack(xpcall(pair, handler))
local o, p, q, r = pack(0, xpcall(pair, handler))
local s, t, u = xpcall(fail, handler)
local v, w, x, y = 0, xpcall(fail, handler)
return a, b, c, d, e, f, g, h, i, j, k, l, m, n, o, p, q, r, s, t, u, v, w, x, y
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 25 {
		t.Fatalf("expected 25 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != true || returnValues[1].Data != float64(1) || returnValues[2].Data != float64(2) {
		t.Fatalf("unexpected direct xpcall success values: %#v", returnValues[:3])
	}

	if returnValues[3].Type != ValueTypeBoolean || returnValues[3].Data != true || returnValues[4].Data != float64(4) || returnValues[5].Type != ValueTypeNil {
		t.Fatalf("unexpected non-final xpcall success assignment values: %#v", returnValues[3:6])
	}

	if returnValues[6].Data != float64(0) || returnValues[7].Type != ValueTypeBoolean || returnValues[7].Data != true || returnValues[8].Data != float64(1) || returnValues[9].Data != float64(2) {
		t.Fatalf("unexpected final xpcall success assignment values: %#v", returnValues[6:10])
	}

	if returnValues[10].Type != ValueTypeBoolean || returnValues[10].Data != true || returnValues[11].Data != float64(1) || returnValues[12].Data != float64(2) || returnValues[13].Type != ValueTypeNil {
		t.Fatalf("unexpected direct xpcall success call-arg values: %#v", returnValues[10:14])
	}

	if returnValues[14].Data != float64(0) || returnValues[15].Type != ValueTypeBoolean || returnValues[15].Data != true || returnValues[16].Data != float64(1) || returnValues[17].Data != float64(2) {
		t.Fatalf("unexpected final xpcall success call-arg values: %#v", returnValues[14:18])
	}

	if returnValues[18].Type != ValueTypeBoolean || returnValues[18].Data != false || returnValues[19].Type != ValueTypeString || returnValues[19].Data != "handled:boom" || returnValues[20].Type != ValueTypeString || returnValues[20].Data != "extra" {
		t.Fatalf("unexpected direct xpcall failure values: %#v", returnValues[18:21])
	}

	if returnValues[21].Data != float64(0) || returnValues[22].Type != ValueTypeBoolean || returnValues[22].Data != false || returnValues[23].Type != ValueTypeString || returnValues[23].Data != "handled:boom" || returnValues[24].Type != ValueTypeString || returnValues[24].Data != "extra" {
		t.Fatalf("unexpected mixed xpcall failure values: %#v", returnValues[21:25])
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

func TestExecStringEvaluatesBuiltinXPCallWithMetatableCallAndHandler(t *testing.T) {
	state := NewState()

	if err := state.ExecString(`
local callable = {}
setmetatable(callable, {
	__call = function(_, message)
		error("boom:" .. tostring(message))
	end
})

local handler = {}
setmetatable(handler, {
	__call = function(self, err)
		return self == handler, "handled:" .. err
	end
})

return xpcall(callable, handler)
`); err != nil {
		t.Fatalf("exec string: %v", err)
	}

	returnValues := state.LastReturnValues()
	if len(returnValues) != 3 {
		t.Fatalf("expected 3 return values, got %d", len(returnValues))
	}

	if returnValues[0].Type != ValueTypeBoolean || returnValues[0].Data != false {
		t.Fatalf("unexpected first return value: %#v", returnValues[0])
	}

	if returnValues[1].Type != ValueTypeBoolean || returnValues[1].Data != true {
		t.Fatalf("unexpected second return value: %#v", returnValues[1])
	}

	if returnValues[2].Type != ValueTypeString || returnValues[2].Data != "handled:boom:nil" {
		t.Fatalf("unexpected third return value: %#v", returnValues[2])
	}
}

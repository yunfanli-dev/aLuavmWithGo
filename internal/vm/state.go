package vm

import (
	"fmt"
	"io"
	"strings"
)

// State owns the bootstrap VM runtime objects for a single Lua execution context.
type State struct {
	stack        *Stack
	globals      map[string]*valueCell
	output       io.Writer
	lastProgram  *FrontendResult
	lastReturned []Value
}

// NewState creates the minimum VM state required for the M1 bootstrap stage.
func NewState() *State {
	state := &State{
		stack:   NewStack(),
		globals: make(map[string]*valueCell),
		output:  io.Discard,
	}

	state.registerBuiltins()
	return state
}

// ExecString is the temporary execution entry for Lua source strings.
func (s *State) ExecString(source string) error {
	return s.ExecSource(Source{
		Name:    "<memory>",
		Content: source,
	})
}

// ExecSource is the temporary execution entry for loaded Lua source payloads.
func (s *State) ExecSource(source Source) error {
	trimmed := strings.TrimSpace(source.Content)
	if trimmed == "" {
		return nil
	}

	sourceName := source.Name
	if sourceName == "" {
		sourceName = "<unknown>"
	}

	frontendResult, err := compileSource(Source{
		Name:    sourceName,
		Content: source.Content,
	})
	if err != nil {
		return err
	}

	s.lastProgram = frontendResult
	s.lastReturned = nil

	result, err := executeProgram(s, frontendResult.Program)
	if err != nil {
		return fmt.Errorf("execute compiled Lua source %q: %w", sourceName, err)
	}

	s.lastReturned = append([]Value(nil), result.returnValues...)
	return nil
}

// StackSize reports the current operand stack size for verification and debugging.
func (s *State) StackSize() int {
	return s.stack.Len()
}

// LastProgram returns the most recent compiled frontend result for testing and debugging.
func (s *State) LastProgram() *FrontendResult {
	return s.lastProgram
}

// LastReturnValues returns the most recent explicit return values produced by execution.
func (s *State) LastReturnValues() []Value {
	return append([]Value(nil), s.lastReturned...)
}

// SetOutput changes the writer used by builtin output functions like print.
func (s *State) SetOutput(writer io.Writer) {
	if writer == nil {
		s.output = io.Discard
		return
	}

	s.output = writer
}

// RegisterFunction exposes a Go host function to the Lua global environment.
func (s *State) RegisterFunction(name string, fn NativeFunction) error {
	return s.registerNativeFunction(name, &nativeFunction{
		name: name,
		fn:   fn,
	})
}

func (s *State) registerContextualFunction(name string, fn contextualNativeFunction) error {
	return s.registerNativeFunction(name, &nativeFunction{
		name:           name,
		contextualImpl: fn,
	})
}

func (s *State) registerNativeFunction(name string, function *nativeFunction) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("register function with empty name")
	}

	if function == nil || (function.fn == nil && function.contextualImpl == nil) {
		return fmt.Errorf("register function %q with nil handler", name)
	}

	s.globals[name] = &valueCell{
		value: Value{
			Type: ValueTypeFunction,
			Data: function,
		},
	}

	return nil
}

func (s *State) registerBuiltinPrint() {
	_ = s.registerContextualFunction("print", func(exec *executor, args []Value) ([]Value, error) {
		parts := make([]string, 0, len(args))
		for _, arg := range args {
			text, err := exec.valueToString(arg)
			if err != nil {
				return nil, err
			}

			parts = append(parts, text)
		}

		if _, err := fmt.Fprintln(s.output, strings.Join(parts, "\t")); err != nil {
			return nil, err
		}

		return nil, nil
	})
}

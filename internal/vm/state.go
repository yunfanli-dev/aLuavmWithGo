package vm

import (
	"fmt"
	"strings"
)

// State owns the bootstrap VM runtime objects for a single Lua execution context.
type State struct {
	stack        *Stack
	lastProgram  *FrontendResult
	lastReturned []Value
}

// NewState creates the minimum VM state required for the M1 bootstrap stage.
func NewState() *State {
	return &State{
		stack: NewStack(),
	}
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

	result, err := executeProgram(frontendResult.Program)
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

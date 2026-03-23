package api

import (
	"context"
	"io"

	"github.com/yunfanli-dev/aLuavmWithGo/internal/vm"
)

// VM is the high-level entry point used by Go hosts to interact with the Lua runtime.
type VM struct {
	state *vm.State
}

// NewVM creates a VM with the minimum runtime state required by the current bootstrap stage.
func NewVM() *VM {
	return &VM{
		state: vm.NewState(),
	}
}

// ExecString executes a Lua source string through the current bootstrap pipeline.
func (v *VM) ExecString(source string) error {
	return v.ExecStringWithContext(context.Background(), source)
}

// ExecStringWithContext executes a Lua source string with host cancellation support.
func (v *VM) ExecStringWithContext(ctx context.Context, source string) error {
	return v.ExecSourceWithContext(ctx, NewStringSource("<memory>", source))
}

// ExecSource executes a loaded Lua source payload through the current bootstrap pipeline.
func (v *VM) ExecSource(source Source) error {
	return v.ExecSourceWithContext(context.Background(), source)
}

// ExecSourceWithContext executes a loaded Lua source payload with host cancellation support.
func (v *VM) ExecSourceWithContext(ctx context.Context, source Source) error {
	return v.state.ExecSourceWithContext(ctx, vm.Source{
		Name:    source.Name,
		Content: source.Content,
	})
}

// ExecFile loads a Lua file from disk and sends it through the current bootstrap pipeline.
func (v *VM) ExecFile(path string) error {
	return v.ExecFileWithContext(context.Background(), path)
}

// ExecFileWithContext loads a Lua file from disk and executes it with host cancellation support.
func (v *VM) ExecFileWithContext(ctx context.Context, path string) error {
	source, err := NewFileSource(path)
	if err != nil {
		return err
	}

	return v.ExecSourceWithContext(ctx, source)
}

// RegisterFunction exposes a Go host function to the Lua global environment.
func (v *VM) RegisterFunction(name string, fn func(args []Value) ([]Value, error)) error {
	return v.state.RegisterFunction(name, func(args []vm.Value) ([]vm.Value, error) {
		return fn(args)
	})
}

// SetOutput changes the writer used by builtin output functions like print.
func (v *VM) SetOutput(writer io.Writer) {
	v.state.SetOutput(writer)
}

// SetStepLimit configures the maximum number of execution steps for one script run.
func (v *VM) SetStepLimit(limit int) {
	v.state.SetStepLimit(limit)
}

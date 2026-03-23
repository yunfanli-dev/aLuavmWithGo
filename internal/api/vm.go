package api

import "github.com/yunfanli-dev/aLuavmWithGo/internal/vm"

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
	return v.ExecSource(NewStringSource("<memory>", source))
}

// ExecSource executes a loaded Lua source payload through the current bootstrap pipeline.
func (v *VM) ExecSource(source Source) error {
	return v.state.ExecSource(vm.Source{
		Name:    source.Name,
		Content: source.Content,
	})
}

// ExecFile loads a Lua file from disk and sends it through the current bootstrap pipeline.
func (v *VM) ExecFile(path string) error {
	source, err := NewFileSource(path)
	if err != nil {
		return err
	}

	return v.ExecSource(source)
}

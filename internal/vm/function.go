package vm

import "github.com/yunfanli-dev/aLuavmWithGo/internal/ir"

// NativeFunction is a Go host function exposed to the Lua runtime.
type NativeFunction func(args []Value) ([]Value, error)
type contextualNativeFunction func(exec *executor, args []Value) ([]Value, error)

type valueCell struct {
	value Value
}

type userFunction struct {
	name       string
	parameters []string
	body       []ir.Statement
	captured   map[string]*valueCell
}

type nativeFunction struct {
	name           string
	fn             NativeFunction
	contextualImpl contextualNativeFunction
}

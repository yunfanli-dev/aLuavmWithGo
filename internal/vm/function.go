package vm

import "github.com/yunfanli-dev/aLuavmWithGo/internal/ir"

// NativeFunction is a Go host function exposed to the Lua runtime.
type NativeFunction func(args []Value) ([]Value, error)

type userFunction struct {
	name       string
	parameters []string
	body       []ir.Statement
}

type nativeFunction struct {
	name string
	fn   NativeFunction
}

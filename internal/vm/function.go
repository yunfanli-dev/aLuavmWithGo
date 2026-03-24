package vm

import "github.com/yunfanli-dev/aLuavmWithGo/internal/ir"

// NativeFunction 表示暴露给 Lua 运行时调用的 Go 宿主函数签名。
// 参数和返回值都使用统一的 Value 切片承载，以便和 VM 内部直接对接。
type NativeFunction func(args []Value) ([]Value, error)
type contextualNativeFunction func(exec *executor, args []Value) ([]Value, error)

type valueCell struct {
	value Value
}

type userFunction struct {
	name       string
	parameters []string
	isVararg   bool
	body       []ir.Statement
	captured   map[string]*valueCell
}

type nativeFunction struct {
	name           string
	fn             NativeFunction
	contextualImpl contextualNativeFunction
}

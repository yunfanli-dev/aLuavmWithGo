package vm

import "github.com/yunfanli-dev/aLuavmWithGo/internal/ir"

// NativeFunction 表示暴露给 Lua 运行时调用的 Go 宿主函数签名。
// 参数和返回值都使用统一的 Value 切片承载，以便和 VM 内部直接对接。
type NativeFunction func(args []Value) ([]Value, error)
type contextualNativeFunction func(exec *executor, args []Value) ([]Value, error)

type valueCell struct {
	// value 保存当前变量槽位中的实际运行时值。
	// 多个作用域或闭包共享同一个 cell 时，会通过更新这个字段实现联动可见。
	value Value
}

type userFunction struct {
	// name 记录函数声明时绑定的名字。
	// 对匿名函数这里通常为空，仅用于调试和少量错误上下文展示。
	name string
	// parameters 按声明顺序保存形参名列表。
	// 调用时实参会依次写入这些名字对应的局部槽位。
	parameters []string
	// isVararg 标记该函数是否声明了 `...`。
	// 为 true 时，多余实参会额外保存在执行器的 varargs 区域中。
	isVararg bool
	// body 保存函数体对应的 IR 语句序列。
	// 每次调用用户函数时，执行器都会从这里开始解释执行。
	body []ir.Statement
	// captured 保存闭包捕获到的外层变量槽位。
	// 这里存的是 valueCell 指针，以便函数内外能共享同一份可变状态。
	captured map[string]*valueCell
	// env 保存该函数当前绑定的最小环境表。
	// 未命中的全局名读写会回落到这里，而不是直接固定写死到 `_G`。
	env *table
}

type nativeFunction struct {
	// name 记录宿主函数或库函数在运行时中的可读名称。
	// 它主要用于调试、错误信息和辅助定位当前调用来源。
	name string
	// fn 是不依赖执行器上下文的宿主实现入口。
	// 这类函数只基于传入参数工作，不直接访问当前执行器状态。
	fn NativeFunction
	// contextualImpl 是需要访问执行器上下文的宿主实现入口。
	// 这类函数可以调用 Lua 函数、读取元方法或复用当前执行器的辅助能力。
	contextualImpl contextualNativeFunction
	// env 保存该宿主函数当前绑定的最小环境表。
	// 当前主要用于 `getfenv` / `setfenv` 维持最小可观察兼容性。
	env *table
}

package vm

import "fmt"

// ValueType 描述运行时 Value 当前承载的 Lua 值类型。
// 执行器、内建函数和宿主函数都会通过它判断值的实际语义。
type ValueType string

const (
	// ValueTypeNil 表示 Lua 里的 `nil`。
	ValueTypeNil ValueType = "nil"
	// ValueTypeBoolean 表示 Lua 布尔值。
	ValueTypeBoolean ValueType = "boolean"
	// ValueTypeNumber 表示 Lua 数值。
	ValueTypeNumber ValueType = "number"
	// ValueTypeString 表示 Lua 字符串。
	ValueTypeString ValueType = "string"
	// ValueTypeFunction 表示 Lua 函数值，包括用户函数和宿主函数包装。
	ValueTypeFunction ValueType = "function"
	// ValueTypeTable 表示 Lua table。
	ValueTypeTable ValueType = "table"
)

// Value 是 VM 内部统一使用的运行时值容器。
// Type 描述值的逻辑类型，Data 保存对应的实际负载。
type Value struct {
	Type ValueType
	Data any
}

// NilValue 构造运行时统一使用的 `nil` 值。
// 这样可以避免在代码各处手写零散的 nil Value 初始化逻辑。
func NilValue() Value {
	return Value{Type: ValueTypeNil}
}

// String 返回便于调试输出的值表示形式。
// 它主要面向日志和测试断言，不保证与 Lua `tostring` 完全一致。
func (v Value) String() string {
	if v.Type == ValueTypeNil {
		return "nil"
	}

	return fmt.Sprintf("%s(%v)", v.Type, v.Data)
}

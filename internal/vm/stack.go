package vm

import "fmt"

// Stack 表示当前 VM 执行上下文使用的操作数栈。
// 目前主要服务于执行器和测试，后续也可以承载更完整的调用栈交互。
type Stack struct {
	values []Value
}

// NewStack 创建一个空的操作数栈。
// 新栈默认不包含任何运行时值，调用方需要按执行过程逐步压栈。
func NewStack() *Stack {
	return &Stack{
		values: make([]Value, 0),
	}
}

// Push 把一个运行时值压入栈顶。
// 该操作不会复制值本身，只会把 Value 追加到内部切片末尾。
func (s *Stack) Push(value Value) {
	s.values = append(s.values, value)
}

// Pop 从栈顶弹出一个运行时值并返回。
// 如果当前栈为空，会返回明确的下溢错误，便于执行器和测试快速定位问题。
func (s *Stack) Pop() (Value, error) {
	if len(s.values) == 0 {
		return NilValue(), fmt.Errorf("vm stack underflow")
	}

	lastIndex := len(s.values) - 1
	value := s.values[lastIndex]
	s.values = s.values[:lastIndex]

	return value, nil
}

// Len 返回当前栈里保存的值数量。
// 该方法主要用于调试、测试和状态检查。
func (s *Stack) Len() int {
	return len(s.values)
}

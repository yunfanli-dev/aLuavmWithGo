package vm

import "fmt"

// Stack stores operand values for the current VM execution context.
type Stack struct {
	values []Value
}

// NewStack creates an empty operand stack for a VM state.
func NewStack() *Stack {
	return &Stack{
		values: make([]Value, 0),
	}
}

// Push appends a runtime value to the top of the operand stack.
func (s *Stack) Push(value Value) {
	s.values = append(s.values, value)
}

// Pop removes and returns the top runtime value from the operand stack.
func (s *Stack) Pop() (Value, error) {
	if len(s.values) == 0 {
		return NilValue(), fmt.Errorf("vm stack underflow")
	}

	lastIndex := len(s.values) - 1
	value := s.values[lastIndex]
	s.values = s.values[:lastIndex]

	return value, nil
}

// Len returns the number of values currently stored in the operand stack.
func (s *Stack) Len() int {
	return len(s.values)
}

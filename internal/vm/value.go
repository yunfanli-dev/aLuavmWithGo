package vm

import "fmt"

// ValueType describes the runtime type carried by a Lua value.
type ValueType string

const (
	// ValueTypeNil represents the Lua nil value.
	ValueTypeNil ValueType = "nil"
	// ValueTypeBoolean represents a Lua boolean.
	ValueTypeBoolean ValueType = "boolean"
	// ValueTypeNumber represents a Lua number.
	ValueTypeNumber ValueType = "number"
	// ValueTypeString represents a Lua string.
	ValueTypeString ValueType = "string"
)

// Value is the unified runtime container used by the VM stack and future instructions.
type Value struct {
	Type ValueType
	Data any
}

// NilValue creates the canonical nil value used by the bootstrap runtime.
func NilValue() Value {
	return Value{Type: ValueTypeNil}
}

// String returns a debug-friendly representation of the current runtime value.
func (v Value) String() string {
	if v.Type == ValueTypeNil {
		return "nil"
	}

	return fmt.Sprintf("%s(%v)", v.Type, v.Data)
}

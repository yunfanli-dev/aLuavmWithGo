package vm

import (
	"fmt"
	"strconv"
	"strings"
)

func (s *State) registerBuiltins() {
	s.registerBuiltinPrint()
	s.registerBuiltinType()
	s.registerBuiltinToString()
	s.registerBuiltinToNumber()
	s.registerBuiltinAssert()
	s.registerBuiltinError()
	s.registerBuiltinNext()
	s.registerBuiltinPairs()
	s.registerBuiltinIPairs()
	s.registerBuiltinGetMetatable()
	s.registerBuiltinSetMetatable()
	s.registerBuiltinRawGet()
	s.registerBuiltinRawSet()
	s.registerBuiltinPCall()
}

// registerBuiltinType installs the minimal `type` builtin.
func (s *State) registerBuiltinType() {
	_ = s.RegisterFunction("type", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("type expects 1 argument")
		}

		return []Value{{Type: ValueTypeString, Data: string(args[0].Type)}}, nil
	})
}

// registerBuiltinToString installs the minimal `tostring` builtin.
func (s *State) registerBuiltinToString() {
	_ = s.registerContextualFunction("tostring", func(exec *executor, args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("tostring expects 1 argument")
		}

		text, err := exec.valueToString(args[0])
		if err != nil {
			return nil, err
		}

		return []Value{{Type: ValueTypeString, Data: text}}, nil
	})
}

// registerBuiltinToNumber installs the minimal `tonumber` builtin.
func (s *State) registerBuiltinToNumber() {
	_ = s.RegisterFunction("tonumber", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("tonumber expects 1 argument")
		}

		switch args[0].Type {
		case ValueTypeNumber:
			return []Value{args[0]}, nil
		case ValueTypeString:
			number, err := strconv.ParseFloat(strings.TrimSpace(args[0].Data.(string)), 64)
			if err != nil {
				return []Value{NilValue()}, nil
			}

			return []Value{{Type: ValueTypeNumber, Data: number}}, nil
		default:
			return []Value{NilValue()}, nil
		}
	})
}

// registerBuiltinAssert installs the minimal `assert` builtin.
func (s *State) registerBuiltinAssert() {
	_ = s.RegisterFunction("assert", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("assert expects at least 1 argument")
		}

		if isTruthy(args[0]) {
			return append([]Value(nil), args...), nil
		}

		if len(args) > 1 {
			return nil, fmt.Errorf("%s", valueToString(args[1]))
		}

		return nil, fmt.Errorf("assertion failed!")
	})
}

// registerBuiltinError installs the minimal `error` builtin.
func (s *State) registerBuiltinError() {
	_ = s.RegisterFunction("error", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("error")
		}

		return nil, fmt.Errorf("%s", valueToString(args[0]))
	})
}

// registerBuiltinNext installs the table iteration primitive used by generic for loops.
func (s *State) registerBuiltinNext() {
	_ = s.RegisterFunction("next", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("next expects 1 argument")
		}

		if args[0].Type != ValueTypeTable {
			return nil, fmt.Errorf("next expects table argument")
		}

		key := NilValue()
		if len(args) > 1 {
			key = args[1]
		}

		tableValue, ok := args[0].Data.(*table)
		if !ok {
			return nil, fmt.Errorf("invalid table payload %T", args[0].Data)
		}

		nextKey, nextValue, exists, err := tableValue.next(key)
		if err != nil {
			return nil, err
		}

		if !exists {
			return []Value{NilValue()}, nil
		}

		return []Value{nextKey, nextValue}, nil
	})
}

// registerBuiltinPairs installs the minimal `pairs` builtin backed by `next`.
func (s *State) registerBuiltinPairs() {
	_ = s.RegisterFunction("pairs", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("pairs expects 1 argument")
		}

		if args[0].Type != ValueTypeTable {
			return nil, fmt.Errorf("pairs expects table argument")
		}

		nextValue, ok := s.globals["next"]
		if !ok {
			return nil, fmt.Errorf("pairs requires builtin 'next'")
		}

		return []Value{nextValue.value, args[0], NilValue()}, nil
	})
}

// registerBuiltinIPairs installs the minimal sequential array iterator.
func (s *State) registerBuiltinIPairs() {
	iterator := &nativeFunction{
		name: "ipairs_iterator",
		fn: func(args []Value) ([]Value, error) {
			if len(args) < 1 {
				return nil, fmt.Errorf("ipairs iterator expects table state")
			}

			if args[0].Type != ValueTypeTable {
				return nil, fmt.Errorf("ipairs iterator expects table state")
			}

			currentIndex := float64(0)
			if len(args) > 1 {
				if args[1].Type != ValueTypeNil {
					if args[1].Type != ValueTypeNumber {
						return nil, fmt.Errorf("ipairs iterator expects numeric index")
					}

					currentIndex = args[1].Data.(float64)
				}
			}

			tableValue, ok := args[0].Data.(*table)
			if !ok {
				return nil, fmt.Errorf("invalid table payload %T", args[0].Data)
			}

			nextIndex := currentIndex + 1
			nextKey := Value{Type: ValueTypeNumber, Data: nextIndex}
			value, exists, err := tableValue.get(nextKey)
			if err != nil {
				return nil, err
			}

			if !exists || value.Type == ValueTypeNil {
				return []Value{NilValue()}, nil
			}

			return []Value{nextKey, value}, nil
		},
	}

	_ = s.RegisterFunction("ipairs", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("ipairs expects 1 argument")
		}

		if args[0].Type != ValueTypeTable {
			return nil, fmt.Errorf("ipairs expects table argument")
		}

		return []Value{
			{Type: ValueTypeFunction, Data: iterator},
			args[0],
			{Type: ValueTypeNumber, Data: float64(0)},
		}, nil
	})
}

// registerBuiltinPCall installs the minimal protected-call builtin.
func (s *State) registerBuiltinPCall() {
	_ = s.registerContextualFunction("pcall", func(exec *executor, args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("pcall expects at least 1 argument")
		}

		callable := args[0]
		if callable.Type != ValueTypeFunction {
			return []Value{
				{Type: ValueTypeBoolean, Data: false},
				{Type: ValueTypeString, Data: "attempt to call non-function value"},
			}, nil
		}

		returnValues, err := exec.callFunctionValue(callable, args[1:])
		if err != nil {
			return []Value{
				{Type: ValueTypeBoolean, Data: false},
				{Type: ValueTypeString, Data: err.Error()},
			}, nil
		}

		result := []Value{{Type: ValueTypeBoolean, Data: true}}
		result = append(result, returnValues...)
		return result, nil
	})
}

// registerBuiltinGetMetatable installs the minimal table metatable getter.
func (s *State) registerBuiltinGetMetatable() {
	_ = s.RegisterFunction("getmetatable", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("getmetatable expects 1 argument")
		}

		if args[0].Type != ValueTypeTable {
			return []Value{NilValue()}, nil
		}

		tableValue, ok := args[0].Data.(*table)
		if !ok {
			return nil, fmt.Errorf("invalid table payload %T", args[0].Data)
		}

		metatable := tableValue.getMetatable()
		if metatable == nil {
			return []Value{NilValue()}, nil
		}

		protectedValue, protected, err := tableValue.getProtectedMetatable()
		if err != nil {
			return nil, err
		}

		if protected {
			return []Value{protectedValue}, nil
		}

		return []Value{{Type: ValueTypeTable, Data: metatable}}, nil
	})
}

// registerBuiltinSetMetatable installs the minimal table metatable setter.
func (s *State) registerBuiltinSetMetatable() {
	_ = s.RegisterFunction("setmetatable", func(args []Value) ([]Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("setmetatable expects 2 arguments")
		}

		if args[0].Type != ValueTypeTable {
			return nil, fmt.Errorf("setmetatable expects table as first argument")
		}

		tableValue, ok := args[0].Data.(*table)
		if !ok {
			return nil, fmt.Errorf("invalid table payload %T", args[0].Data)
		}

		if _, protected, err := tableValue.getProtectedMetatable(); err != nil {
			return nil, err
		} else if protected {
			return nil, fmt.Errorf("cannot change a protected metatable")
		}

		if args[1].Type == ValueTypeNil {
			tableValue.setMetatable(nil)
			return []Value{args[0]}, nil
		}

		if args[1].Type != ValueTypeTable {
			return nil, fmt.Errorf("setmetatable expects table or nil as second argument")
		}

		metatable, ok := args[1].Data.(*table)
		if !ok {
			return nil, fmt.Errorf("invalid metatable payload %T", args[1].Data)
		}

		tableValue.setMetatable(metatable)
		return []Value{args[0]}, nil
	})
}

// registerBuiltinRawGet installs direct table reads that bypass metatable __index logic.
func (s *State) registerBuiltinRawGet() {
	_ = s.RegisterFunction("rawget", func(args []Value) ([]Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("rawget expects 2 arguments")
		}

		if args[0].Type != ValueTypeTable {
			return nil, fmt.Errorf("rawget expects table as first argument")
		}

		tableValue, ok := args[0].Data.(*table)
		if !ok {
			return nil, fmt.Errorf("invalid table payload %T", args[0].Data)
		}

		value, exists, err := tableValue.get(args[1])
		if err != nil {
			return nil, err
		}

		if !exists {
			return []Value{NilValue()}, nil
		}

		return []Value{value}, nil
	})
}

// registerBuiltinRawSet installs direct table writes that bypass metatable __newindex logic.
func (s *State) registerBuiltinRawSet() {
	_ = s.RegisterFunction("rawset", func(args []Value) ([]Value, error) {
		if len(args) < 3 {
			return nil, fmt.Errorf("rawset expects 3 arguments")
		}

		if args[0].Type != ValueTypeTable {
			return nil, fmt.Errorf("rawset expects table as first argument")
		}

		tableValue, ok := args[0].Data.(*table)
		if !ok {
			return nil, fmt.Errorf("invalid table payload %T", args[0].Data)
		}

		if err := tableValue.set(args[1], args[2]); err != nil {
			return nil, err
		}

		return []Value{args[0]}, nil
	})
}

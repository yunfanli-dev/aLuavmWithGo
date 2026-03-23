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
	_ = s.RegisterFunction("tostring", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("tostring expects 1 argument")
		}

		return []Value{{Type: ValueTypeString, Data: valueToString(args[0])}}, nil
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

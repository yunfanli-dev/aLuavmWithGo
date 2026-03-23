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
	s.registerBuiltinPCall()
}

func (s *State) registerBuiltinType() {
	_ = s.RegisterFunction("type", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("type expects 1 argument")
		}

		return []Value{{Type: ValueTypeString, Data: string(args[0].Type)}}, nil
	})
}

func (s *State) registerBuiltinToString() {
	_ = s.RegisterFunction("tostring", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("tostring expects 1 argument")
		}

		return []Value{{Type: ValueTypeString, Data: valueToString(args[0])}}, nil
	})
}

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

func (s *State) registerBuiltinError() {
	_ = s.RegisterFunction("error", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("error")
		}

		return nil, fmt.Errorf("%s", valueToString(args[0]))
	})
}

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

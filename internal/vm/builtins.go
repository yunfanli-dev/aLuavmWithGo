package vm

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func (s *State) registerBuiltins() {
	s.registerBuiltinPrint()
	s.registerBuiltinType()
	s.registerBuiltinToString()
	s.registerBuiltinToNumber()
	s.registerBuiltinSelect()
	s.registerBuiltinUnpack()
	s.registerBuiltinAssert()
	s.registerBuiltinError()
	s.registerBuiltinNext()
	s.registerBuiltinPairs()
	s.registerBuiltinIPairs()
	s.registerBuiltinGetMetatable()
	s.registerBuiltinSetMetatable()
	s.registerBuiltinRawGet()
	s.registerBuiltinRawSet()
	s.registerBuiltinRawEqual()
	s.registerBuiltinPCall()
	s.registerBuiltinXPCall()
	s.registerBuiltinTableLibrary()
	s.registerBuiltinMathLibrary()
	s.registerBuiltinStringLibrary()
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

// registerBuiltinSelect installs the minimal Lua 5.1 `select` builtin for vararg slicing.
func (s *State) registerBuiltinSelect() {
	_ = s.RegisterFunction("select", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("select expects at least 1 argument")
		}

		if args[0].Type == ValueTypeString && args[0].Data == "#" {
			return []Value{{Type: ValueTypeNumber, Data: float64(len(args) - 1)}}, nil
		}

		index, err := builtinInteger(args[0], "select")
		if err != nil {
			return nil, err
		}

		count := len(args) - 1
		if index == 0 {
			return nil, fmt.Errorf("select index out of range")
		}

		if index < 0 {
			index = count + index + 1
		}

		if index < 1 {
			return nil, fmt.Errorf("select index out of range")
		}

		if index > count {
			return nil, nil
		}

		return append([]Value(nil), args[index:]...), nil
	})
}

// registerBuiltinUnpack installs the minimal Lua 5.1 `unpack` builtin for array-style table slices.
func (s *State) registerBuiltinUnpack() {
	_ = s.RegisterFunction("unpack", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("unpack expects at least 1 argument")
		}

		if args[0].Type != ValueTypeTable {
			return nil, fmt.Errorf("unpack expects table argument")
		}

		tableValue, ok := args[0].Data.(*table)
		if !ok {
			return nil, fmt.Errorf("invalid table payload %T", args[0].Data)
		}

		startIndex := 1
		if len(args) > 1 {
			start, err := builtinInteger(args[1], "unpack")
			if err != nil {
				return nil, err
			}

			startIndex = start
		}

		endIndex := 0
		if len(args) > 2 {
			end, err := builtinInteger(args[2], "unpack")
			if err != nil {
				return nil, err
			}

			endIndex = end
		} else {
			end, err := tableSequenceEnd(tableValue, startIndex)
			if err != nil {
				return nil, err
			}

			endIndex = end
		}

		if endIndex < startIndex {
			return nil, nil
		}

		values := make([]Value, 0, endIndex-startIndex+1)
		for index := startIndex; index <= endIndex; index++ {
			key := Value{Type: ValueTypeNumber, Data: float64(index)}
			value, exists, err := tableValue.get(key)
			if err != nil {
				return nil, err
			}

			if !exists {
				value = NilValue()
			}

			values = append(values, value)
		}

		return values, nil
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

func builtinInteger(value Value, name string) (int, error) {
	if value.Type != ValueTypeNumber {
		return 0, fmt.Errorf("%s expects numeric index", name)
	}

	number, ok := value.Data.(float64)
	if !ok {
		return 0, fmt.Errorf("%s received invalid numeric payload %T", name, value.Data)
	}

	index := int(number)
	if float64(index) != number {
		return 0, fmt.Errorf("%s expects integer index", name)
	}

	return index, nil
}

func tableSequenceEnd(tableValue *table, startIndex int) (int, error) {
	index := startIndex
	for {
		key := Value{Type: ValueTypeNumber, Data: float64(index)}
		value, exists, err := tableValue.get(key)
		if err != nil {
			return 0, err
		}

		if !exists || value.Type == ValueTypeNil {
			return index - 1, nil
		}

		index++
	}
}

// registerBuiltinTableLibrary installs the minimal global `table` library surface.
func (s *State) registerBuiltinTableLibrary() {
	s.registerTableInsert()
	s.registerTableRemove()
	s.registerTableConcat()
	s.registerTableSort()
}

// registerBuiltinMathLibrary installs the minimal global `math` library surface.
func (s *State) registerBuiltinMathLibrary() {
	s.registerMathAbs()
	s.registerMathFloor()
	s.registerMathCeil()
	s.registerMathMax()
	s.registerMathMin()
	s.registerMathSqrt()
	s.registerMathPow()
	s.registerMathRandom()
	s.registerMathRandomSeed()
	s.registerMathLog()
	s.registerMathExp()
	s.registerMathSin()
	s.registerMathCos()
}

// registerBuiltinStringLibrary installs the minimal global `string` library surface.
func (s *State) registerBuiltinStringLibrary() {
	s.registerStringLen()
	s.registerStringSub()
	s.registerStringLower()
	s.registerStringUpper()
	s.registerStringRep()
	s.registerStringReverse()
	s.registerStringByte()
	s.registerStringChar()
}

// registerTableInsert installs `table.insert` for the current sequence-style table subset.
func (s *State) registerTableInsert() {
	_ = s.registerTableLibraryFunction("insert", func(args []Value) ([]Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("table.insert expects 2 or 3 arguments")
		}

		tableValue, err := requireBuiltinTable(args[0], "table.insert")
		if err != nil {
			return nil, err
		}

		endIndex, err := tableSequenceEnd(tableValue, 1)
		if err != nil {
			return nil, err
		}

		insertIndex := endIndex + 1
		valueIndex := 1
		if len(args) > 2 {
			position, err := builtinInteger(args[1], "table.insert")
			if err != nil {
				return nil, err
			}

			if position < 1 || position > endIndex+1 {
				return nil, fmt.Errorf("table.insert position out of range")
			}

			insertIndex = position
			valueIndex = 2
		}

		for index := endIndex; index >= insertIndex; index-- {
			key := Value{Type: ValueTypeNumber, Data: float64(index)}
			value, exists, err := tableValue.get(key)
			if err != nil {
				return nil, err
			}

			if !exists {
				value = NilValue()
			}

			nextKey := Value{Type: ValueTypeNumber, Data: float64(index + 1)}
			if err := tableValue.set(nextKey, value); err != nil {
				return nil, err
			}
		}

		insertKey := Value{Type: ValueTypeNumber, Data: float64(insertIndex)}
		if err := tableValue.set(insertKey, args[valueIndex]); err != nil {
			return nil, err
		}

		return nil, nil
	})
}

// registerTableRemove installs `table.remove` for the current sequence-style table subset.
func (s *State) registerTableRemove() {
	_ = s.registerTableLibraryFunction("remove", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("table.remove expects 1 or 2 arguments")
		}

		tableValue, err := requireBuiltinTable(args[0], "table.remove")
		if err != nil {
			return nil, err
		}

		endIndex, err := tableSequenceEnd(tableValue, 1)
		if err != nil {
			return nil, err
		}

		if endIndex < 1 {
			return []Value{NilValue()}, nil
		}

		removeIndex := endIndex
		if len(args) > 1 {
			position, err := builtinInteger(args[1], "table.remove")
			if err != nil {
				return nil, err
			}

			if position < 1 || position > endIndex {
				return nil, fmt.Errorf("table.remove position out of range")
			}

			removeIndex = position
		}

		removeKey := Value{Type: ValueTypeNumber, Data: float64(removeIndex)}
		removed, exists, err := tableValue.get(removeKey)
		if err != nil {
			return nil, err
		}

		if !exists {
			removed = NilValue()
		}

		for index := removeIndex; index < endIndex; index++ {
			nextKey := Value{Type: ValueTypeNumber, Data: float64(index + 1)}
			value, exists, err := tableValue.get(nextKey)
			if err != nil {
				return nil, err
			}

			currentKey := Value{Type: ValueTypeNumber, Data: float64(index)}
			if !exists {
				value = NilValue()
			}

			if err := tableValue.set(currentKey, value); err != nil {
				return nil, err
			}
		}

		lastKey := Value{Type: ValueTypeNumber, Data: float64(endIndex)}
		if err := tableValue.set(lastKey, NilValue()); err != nil {
			return nil, err
		}

		return []Value{removed}, nil
	})
}

// registerTableConcat installs `table.concat` for the current sequence-style table subset.
func (s *State) registerTableConcat() {
	_ = s.registerTableLibraryFunction("concat", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("table.concat expects at least 1 argument")
		}

		tableValue, err := requireBuiltinTable(args[0], "table.concat")
		if err != nil {
			return nil, err
		}

		separator := ""
		if len(args) > 1 {
			if args[1].Type != ValueTypeString {
				return nil, fmt.Errorf("table.concat expects string separator")
			}

			separator = args[1].Data.(string)
		}

		startIndex := 1
		if len(args) > 2 {
			start, err := builtinInteger(args[2], "table.concat")
			if err != nil {
				return nil, err
			}

			startIndex = start
		}

		endIndex := 0
		if len(args) > 3 {
			end, err := builtinInteger(args[3], "table.concat")
			if err != nil {
				return nil, err
			}

			endIndex = end
		} else {
			end, err := tableSequenceEnd(tableValue, startIndex)
			if err != nil {
				return nil, err
			}

			endIndex = end
		}

		if endIndex < startIndex {
			return []Value{{Type: ValueTypeString, Data: ""}}, nil
		}

		parts := make([]string, 0, endIndex-startIndex+1)
		for index := startIndex; index <= endIndex; index++ {
			key := Value{Type: ValueTypeNumber, Data: float64(index)}
			value, exists, err := tableValue.get(key)
			if err != nil {
				return nil, err
			}

			if !exists || value.Type == ValueTypeNil {
				return nil, fmt.Errorf("table.concat encountered nil value")
			}

			if value.Type != ValueTypeString && value.Type != ValueTypeNumber {
				return nil, fmt.Errorf("table.concat expects stringable sequence values")
			}

			parts = append(parts, valueToString(value))
		}

		return []Value{{Type: ValueTypeString, Data: strings.Join(parts, separator)}}, nil
	})
}

// registerTableSort installs `table.sort` for the current sequence-style table subset.
func (s *State) registerTableSort() {
	_ = s.registerContextualLibraryFunction("table", "sort", func(exec *executor, args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("table.sort expects 1 or 2 arguments")
		}

		tableValue, err := requireBuiltinTable(args[0], "table.sort")
		if err != nil {
			return nil, err
		}

		var comparator Value
		hasComparator := false
		if len(args) > 1 {
			if args[1].Type != ValueTypeFunction {
				return nil, fmt.Errorf("table.sort expects function comparator")
			}

			comparator = args[1]
			hasComparator = true
		}

		endIndex, err := tableSequenceEnd(tableValue, 1)
		if err != nil {
			return nil, err
		}

		if endIndex <= 1 {
			return nil, nil
		}

		values := make([]Value, 0, endIndex)
		for index := 1; index <= endIndex; index++ {
			key := Value{Type: ValueTypeNumber, Data: float64(index)}
			value, exists, err := tableValue.get(key)
			if err != nil {
				return nil, err
			}

			if !exists || value.Type == ValueTypeNil {
				return nil, fmt.Errorf("table.sort encountered nil value")
			}

			values = append(values, value)
		}

		for index := 1; index < len(values); index++ {
			current := values[index]
			position := index
			for position > 0 {
				less, err := exec.tableSortLess(current, values[position-1], comparator, hasComparator)
				if err != nil {
					return nil, err
				}

				if !less {
					break
				}

				values[position] = values[position-1]
				position--
			}

			values[position] = current
		}

		for index, value := range values {
			key := Value{Type: ValueTypeNumber, Data: float64(index + 1)}
			if err := tableValue.set(key, value); err != nil {
				return nil, err
			}
		}

		return nil, nil
	})
}

// registerMathAbs installs `math.abs` for numeric absolute values.
func (s *State) registerMathAbs() {
	_ = s.registerLibraryFunction("math", "abs", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("math.abs expects 1 argument")
		}

		number, err := requireNumber(args[0], "math.abs")
		if err != nil {
			return nil, err
		}

		return []Value{{Type: ValueTypeNumber, Data: math.Abs(number)}}, nil
	})
}

// registerMathFloor installs `math.floor` for numeric floor rounding.
func (s *State) registerMathFloor() {
	_ = s.registerLibraryFunction("math", "floor", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("math.floor expects 1 argument")
		}

		number, err := requireNumber(args[0], "math.floor")
		if err != nil {
			return nil, err
		}

		return []Value{{Type: ValueTypeNumber, Data: math.Floor(number)}}, nil
	})
}

// registerMathCeil installs `math.ceil` for numeric ceil rounding.
func (s *State) registerMathCeil() {
	_ = s.registerLibraryFunction("math", "ceil", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("math.ceil expects 1 argument")
		}

		number, err := requireNumber(args[0], "math.ceil")
		if err != nil {
			return nil, err
		}

		return []Value{{Type: ValueTypeNumber, Data: math.Ceil(number)}}, nil
	})
}

// registerMathMax installs `math.max` for numeric maximum selection.
func (s *State) registerMathMax() {
	_ = s.registerLibraryFunction("math", "max", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("math.max expects at least 1 argument")
		}

		maxValue, err := requireNumber(args[0], "math.max")
		if err != nil {
			return nil, err
		}

		for _, arg := range args[1:] {
			number, err := requireNumber(arg, "math.max")
			if err != nil {
				return nil, err
			}

			if number > maxValue {
				maxValue = number
			}
		}

		return []Value{{Type: ValueTypeNumber, Data: maxValue}}, nil
	})
}

// registerMathMin installs `math.min` for numeric minimum selection.
func (s *State) registerMathMin() {
	_ = s.registerLibraryFunction("math", "min", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("math.min expects at least 1 argument")
		}

		minValue, err := requireNumber(args[0], "math.min")
		if err != nil {
			return nil, err
		}

		for _, arg := range args[1:] {
			number, err := requireNumber(arg, "math.min")
			if err != nil {
				return nil, err
			}

			if number < minValue {
				minValue = number
			}
		}

		return []Value{{Type: ValueTypeNumber, Data: minValue}}, nil
	})
}

// registerMathSqrt installs `math.sqrt` for square-root extraction.
func (s *State) registerMathSqrt() {
	_ = s.registerLibraryFunction("math", "sqrt", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("math.sqrt expects 1 argument")
		}

		number, err := requireNumber(args[0], "math.sqrt")
		if err != nil {
			return nil, err
		}

		return []Value{{Type: ValueTypeNumber, Data: math.Sqrt(number)}}, nil
	})
}

// registerMathPow installs `math.pow` for explicit exponentiation.
func (s *State) registerMathPow() {
	_ = s.registerLibraryFunction("math", "pow", func(args []Value) ([]Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("math.pow expects 2 arguments")
		}

		left, err := requireNumber(args[0], "math.pow")
		if err != nil {
			return nil, err
		}

		right, err := requireNumber(args[1], "math.pow")
		if err != nil {
			return nil, err
		}

		return []Value{{Type: ValueTypeNumber, Data: math.Pow(left, right)}}, nil
	})
}

// registerMathRandom installs `math.random` using the state's deterministic RNG.
func (s *State) registerMathRandom() {
	_ = s.registerLibraryFunction("math", "random", func(args []Value) ([]Value, error) {
		switch len(args) {
		case 0:
			return []Value{{Type: ValueTypeNumber, Data: s.random.Float64()}}, nil
		case 1:
			upper, err := builtinInteger(args[0], "math.random")
			if err != nil {
				return nil, err
			}

			if upper < 1 {
				return nil, fmt.Errorf("math.random interval is empty")
			}

			return []Value{{Type: ValueTypeNumber, Data: float64(s.random.Intn(upper) + 1)}}, nil
		case 2:
			lower, err := builtinInteger(args[0], "math.random")
			if err != nil {
				return nil, err
			}

			upper, err := builtinInteger(args[1], "math.random")
			if err != nil {
				return nil, err
			}

			if lower > upper {
				return nil, fmt.Errorf("math.random interval is empty")
			}

			return []Value{{Type: ValueTypeNumber, Data: float64(lower + s.random.Intn(upper-lower+1))}}, nil
		default:
			return nil, fmt.Errorf("math.random expects at most 2 arguments")
		}
	})
}

// registerMathRandomSeed installs `math.randomseed` to reseed the state's RNG.
func (s *State) registerMathRandomSeed() {
	_ = s.registerLibraryFunction("math", "randomseed", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("math.randomseed expects 1 argument")
		}

		seed, err := requireNumber(args[0], "math.randomseed")
		if err != nil {
			return nil, err
		}

		s.random.Seed(int64(seed))
		return nil, nil
	})
}

// registerMathLog installs `math.log` for natural logarithms.
func (s *State) registerMathLog() {
	_ = s.registerLibraryFunction("math", "log", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("math.log expects 1 argument")
		}

		number, err := requireNumber(args[0], "math.log")
		if err != nil {
			return nil, err
		}

		return []Value{{Type: ValueTypeNumber, Data: math.Log(number)}}, nil
	})
}

// registerMathExp installs `math.exp` for the natural exponential function.
func (s *State) registerMathExp() {
	_ = s.registerLibraryFunction("math", "exp", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("math.exp expects 1 argument")
		}

		number, err := requireNumber(args[0], "math.exp")
		if err != nil {
			return nil, err
		}

		return []Value{{Type: ValueTypeNumber, Data: math.Exp(number)}}, nil
	})
}

// registerMathSin installs `math.sin` for sine calculations in radians.
func (s *State) registerMathSin() {
	_ = s.registerLibraryFunction("math", "sin", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("math.sin expects 1 argument")
		}

		number, err := requireNumber(args[0], "math.sin")
		if err != nil {
			return nil, err
		}

		return []Value{{Type: ValueTypeNumber, Data: math.Sin(number)}}, nil
	})
}

// registerMathCos installs `math.cos` for cosine calculations in radians.
func (s *State) registerMathCos() {
	_ = s.registerLibraryFunction("math", "cos", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("math.cos expects 1 argument")
		}

		number, err := requireNumber(args[0], "math.cos")
		if err != nil {
			return nil, err
		}

		return []Value{{Type: ValueTypeNumber, Data: math.Cos(number)}}, nil
	})
}

// registerStringLen installs `string.len` for string length queries.
func (s *State) registerStringLen() {
	_ = s.registerLibraryFunction("string", "len", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("string.len expects 1 argument")
		}

		text, err := requireStringArg(args[0], "string.len")
		if err != nil {
			return nil, err
		}

		return []Value{{Type: ValueTypeNumber, Data: float64(len(text))}}, nil
	})
}

// registerStringSub installs `string.sub` for substring extraction using Lua-style indices.
func (s *State) registerStringSub() {
	_ = s.registerLibraryFunction("string", "sub", func(args []Value) ([]Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("string.sub expects at least 2 arguments")
		}

		text, err := requireStringArg(args[0], "string.sub")
		if err != nil {
			return nil, err
		}

		startIndex, err := builtinInteger(args[1], "string.sub")
		if err != nil {
			return nil, err
		}

		endIndex := len(text)
		if len(args) > 2 {
			end, err := builtinInteger(args[2], "string.sub")
			if err != nil {
				return nil, err
			}

			endIndex = end
		}

		start, end := normalizeStringRange(len(text), startIndex, endIndex)
		if end < start {
			return []Value{{Type: ValueTypeString, Data: ""}}, nil
		}

		return []Value{{Type: ValueTypeString, Data: text[start-1 : end]}}, nil
	})
}

// registerStringLower installs `string.lower` for ASCII case folding through Go's strings package.
func (s *State) registerStringLower() {
	_ = s.registerLibraryFunction("string", "lower", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("string.lower expects 1 argument")
		}

		text, err := requireStringArg(args[0], "string.lower")
		if err != nil {
			return nil, err
		}

		return []Value{{Type: ValueTypeString, Data: strings.ToLower(text)}}, nil
	})
}

// registerStringUpper installs `string.upper` for ASCII case folding through Go's strings package.
func (s *State) registerStringUpper() {
	_ = s.registerLibraryFunction("string", "upper", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("string.upper expects 1 argument")
		}

		text, err := requireStringArg(args[0], "string.upper")
		if err != nil {
			return nil, err
		}

		return []Value{{Type: ValueTypeString, Data: strings.ToUpper(text)}}, nil
	})
}

// registerStringRep installs `string.rep` for repeated string construction.
func (s *State) registerStringRep() {
	_ = s.registerLibraryFunction("string", "rep", func(args []Value) ([]Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("string.rep expects 2 arguments")
		}

		text, err := requireStringArg(args[0], "string.rep")
		if err != nil {
			return nil, err
		}

		count, err := builtinInteger(args[1], "string.rep")
		if err != nil {
			return nil, err
		}

		if count < 0 {
			return nil, fmt.Errorf("string.rep expects non-negative count")
		}

		return []Value{{Type: ValueTypeString, Data: strings.Repeat(text, count)}}, nil
	})
}

// registerStringReverse installs `string.reverse` for string reversal.
func (s *State) registerStringReverse() {
	_ = s.registerLibraryFunction("string", "reverse", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("string.reverse expects 1 argument")
		}

		text, err := requireStringArg(args[0], "string.reverse")
		if err != nil {
			return nil, err
		}

		runes := []rune(text)
		for left, right := 0, len(runes)-1; left < right; left, right = left+1, right-1 {
			runes[left], runes[right] = runes[right], runes[left]
		}

		return []Value{{Type: ValueTypeString, Data: string(runes)}}, nil
	})
}

// registerStringByte installs `string.byte` for byte extraction over the current ASCII-oriented string subset.
func (s *State) registerStringByte() {
	_ = s.registerLibraryFunction("string", "byte", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("string.byte expects 1 to 3 arguments")
		}

		text, err := requireStringArg(args[0], "string.byte")
		if err != nil {
			return nil, err
		}

		startIndex := 1
		if len(args) > 1 {
			start, err := builtinInteger(args[1], "string.byte")
			if err != nil {
				return nil, err
			}

			startIndex = start
		}

		endIndex := startIndex
		if len(args) > 2 {
			end, err := builtinInteger(args[2], "string.byte")
			if err != nil {
				return nil, err
			}

			endIndex = end
		}

		start, end := normalizeStringRange(len(text), startIndex, endIndex)
		if end < start {
			return nil, nil
		}

		values := make([]Value, 0, end-start+1)
		for index := start - 1; index < end; index++ {
			values = append(values, Value{Type: ValueTypeNumber, Data: float64(text[index])})
		}

		return values, nil
	})
}

// registerStringChar installs `string.char` for ASCII byte assembly.
func (s *State) registerStringChar() {
	_ = s.registerLibraryFunction("string", "char", func(args []Value) ([]Value, error) {
		var builder strings.Builder
		for _, arg := range args {
			code, err := builtinInteger(arg, "string.char")
			if err != nil {
				return nil, err
			}

			if code < 0 || code > 255 {
				return nil, fmt.Errorf("string.char byte value out of range")
			}

			builder.WriteByte(byte(code))
		}

		return []Value{{Type: ValueTypeString, Data: builder.String()}}, nil
	})
}

func (s *State) registerTableLibraryFunction(name string, fn NativeFunction) error {
	return s.registerLibraryFunction("table", name, fn)
}

// registerContextualLibraryFunction installs a builtin that needs executor access under a library table.
func (s *State) registerContextualLibraryFunction(libraryName string, name string, fn contextualNativeFunction) error {
	library, err := s.ensureLibraryTable(libraryName)
	if err != nil {
		return err
	}

	return library.set(Value{Type: ValueTypeString, Data: name}, Value{
		Type: ValueTypeFunction,
		Data: &nativeFunction{name: libraryName + "." + name, contextualImpl: fn},
	})
}

func (s *State) registerLibraryFunction(libraryName string, name string, fn NativeFunction) error {
	library, err := s.ensureLibraryTable(libraryName)
	if err != nil {
		return err
	}

	return library.set(Value{Type: ValueTypeString, Data: name}, Value{
		Type: ValueTypeFunction,
		Data: &nativeFunction{name: libraryName + "." + name, fn: fn},
	})
}

// tableSortLess evaluates one `table.sort` comparison using either a custom comparator or the runtime's `<` semantics.
func (e *executor) tableSortLess(left, right Value, comparator Value, hasComparator bool) (bool, error) {
	if hasComparator {
		returnValues, err := e.callFunctionValue(comparator, []Value{left, right})
		if err != nil {
			return false, err
		}

		if len(returnValues) == 0 {
			return false, nil
		}

		return isTruthy(returnValues[0]), nil
	}

	result, err := e.evaluateOrderedComparison(left, right, "<", "__lt", func(a, b float64) bool { return a < b })
	if err != nil {
		return false, err
	}

	if result.Type != ValueTypeBoolean {
		return false, nil
	}

	flag, ok := result.Data.(bool)
	if !ok {
		return false, fmt.Errorf("invalid comparison payload %T", result.Data)
	}

	return flag, nil
}

// requireStringArg unwraps a builtin string argument and validates its runtime type.
func requireStringArg(value Value, name string) (string, error) {
	if value.Type != ValueTypeString {
		return "", fmt.Errorf("%s expects string argument", name)
	}

	text, ok := value.Data.(string)
	if !ok {
		return "", fmt.Errorf("invalid string payload %T", value.Data)
	}

	return text, nil
}

// normalizeStringRange converts Lua-style substring bounds into a clamped 1-based closed interval.
func normalizeStringRange(length int, start int, end int) (int, int) {
	if start < 0 {
		start = length + start + 1
	}
	if end < 0 {
		end = length + end + 1
	}
	if start < 1 {
		start = 1
	}
	if end > length {
		end = length
	}
	return start, end
}

func (s *State) ensureLibraryTable(name string) (*table, error) {
	if existing, ok := s.globals[name]; ok {
		if existing.value.Type != ValueTypeTable {
			return nil, fmt.Errorf("%s library is not a table", name)
		}

		library, ok := existing.value.Data.(*table)
		if !ok {
			return nil, fmt.Errorf("invalid %s library payload %T", name, existing.value.Data)
		}

		return library, nil
	}

	library := newTable()
	s.globals[name] = &valueCell{value: Value{Type: ValueTypeTable, Data: library}}
	return library, nil
}

func requireBuiltinTable(value Value, name string) (*table, error) {
	if value.Type != ValueTypeTable {
		return nil, fmt.Errorf("%s expects table argument", name)
	}

	tableValue, ok := value.Data.(*table)
	if !ok {
		return nil, fmt.Errorf("invalid table payload %T", value.Data)
	}

	return tableValue, nil
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

// registerBuiltinRawEqual installs the minimal `rawequal` builtin that bypasses __eq.
func (s *State) registerBuiltinRawEqual() {
	_ = s.RegisterFunction("rawequal", func(args []Value) ([]Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("rawequal expects 2 arguments")
		}

		return []Value{{Type: ValueTypeBoolean, Data: valuesEqual(args[0], args[1])}}, nil
	})
}

// registerBuiltinXPCall installs the minimal Lua 5.1 `xpcall` builtin with an error handler callback.
func (s *State) registerBuiltinXPCall() {
	_ = s.registerContextualFunction("xpcall", func(exec *executor, args []Value) ([]Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("xpcall expects 2 arguments")
		}

		callable := args[0]
		if callable.Type != ValueTypeFunction {
			return []Value{
				{Type: ValueTypeBoolean, Data: false},
				{Type: ValueTypeString, Data: "attempt to call non-function value"},
			}, nil
		}

		handler := args[1]
		if handler.Type != ValueTypeFunction {
			return []Value{
				{Type: ValueTypeBoolean, Data: false},
				{Type: ValueTypeString, Data: "error handler must be a function"},
			}, nil
		}

		returnValues, err := exec.callFunctionValue(callable, nil)
		if err != nil {
			handlerValues, handlerErr := exec.callFunctionValue(handler, []Value{{Type: ValueTypeString, Data: err.Error()}})
			if handlerErr != nil {
				return []Value{
					{Type: ValueTypeBoolean, Data: false},
					{Type: ValueTypeString, Data: handlerErr.Error()},
				}, nil
			}

			result := []Value{{Type: ValueTypeBoolean, Data: false}}
			result = append(result, handlerValues...)
			return result, nil
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

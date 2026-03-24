package vm

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func (s *State) registerBuiltins() {
	s.registerBuiltinPrint()
	s.registerBuiltinClockMillis()
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

// registerBuiltinType 注册最小 `type` 内建函数。
// 它返回当前值对应的运行时类型名。
func (s *State) registerBuiltinType() {
	_ = s.RegisterFunction("type", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("type expects 1 argument")
		}

		return []Value{{Type: ValueTypeString, Data: string(args[0].Type)}}, nil
	})
}

// registerBuiltinToString 注册最小 `tostring` 内建函数。
// 该实现会复用执行器的字符串化逻辑，因此也能走最小 `__tostring` 钩子。
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

// registerBuiltinToNumber 注册最小 `tonumber` 内建函数。
// 当前主要支持数值直返和字符串到浮点数的基础转换。
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

// registerBuiltinSelect 注册最小 Lua 5.1 `select` 内建函数。
// 当前主要用于 vararg 计数和按位置切片。
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

// registerBuiltinUnpack 注册最小 Lua 5.1 `unpack` 内建函数。
// 它按当前 sequence 语义把 table 指定区间展开成多返回值。
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

// registerBuiltinAssert 注册最小 `assert` 内建函数。
// 条件为真时原样返回参数列表；条件为假时返回错误。
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

// registerBuiltinTableLibrary 注册当前最小可用的全局 `table` 库入口。
// 后续新增 table 子能力时，会继续从这里挂到全局环境。
func (s *State) registerBuiltinTableLibrary() {
	s.registerTableGetN()
	s.registerTableMaxN()
	s.registerTableInsert()
	s.registerTableRemove()
	s.registerTableConcat()
	s.registerTableSort()
}

// registerBuiltinMathLibrary 注册当前最小可用的全局 `math` 库入口。
// 当前只覆盖项目已落地的一小部分数值函数。
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

// registerBuiltinStringLibrary 注册当前最小可用的全局 `string` 库入口。
// 当前字符串库仍是子集实现，但已经覆盖若干高频文本处理能力。
func (s *State) registerBuiltinStringLibrary() {
	s.registerStringFind()
	s.registerStringGSub()
	s.registerStringMatch()
	s.registerStringLen()
	s.registerStringSub()
	s.registerStringLower()
	s.registerStringUpper()
	s.registerStringRep()
	s.registerStringReverse()
	s.registerStringByte()
	s.registerStringChar()
}

// registerTableGetN 注册最小 `table.getn`。
// 它返回从索引 1 开始的连续数组段长度，与当前 `#table` 语义保持一致。
func (s *State) registerTableGetN() {
	_ = s.registerTableLibraryFunction("getn", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("table.getn expects 1 argument")
		}

		tableValue, err := requireBuiltinTable(args[0], "table.getn")
		if err != nil {
			return nil, err
		}

		endIndex, err := tableSequenceEnd(tableValue, 1)
		if err != nil {
			return nil, err
		}

		return []Value{{Type: ValueTypeNumber, Data: float64(endIndex)}}, nil
	})
}

// registerTableMaxN 注册最小 `table.maxn`。
// 它返回当前 table 中存在的最大数值 key，而不是连续数组段长度。
func (s *State) registerTableMaxN() {
	_ = s.registerTableLibraryFunction("maxn", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("table.maxn expects 1 argument")
		}

		tableValue, err := requireBuiltinTable(args[0], "table.maxn")
		if err != nil {
			return nil, err
		}

		maximum, err := tableValue.maxNumericKey()
		if err != nil {
			return nil, err
		}

		return []Value{{Type: ValueTypeNumber, Data: maximum}}, nil
	})
}

// registerTableInsert 注册 `table.insert`。
// 当前实现基于最小 sequence 语义，在连续数组段内完成插入和后移。
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

// registerTableRemove 注册 `table.remove`。
// 当前实现按最小 sequence 语义移除元素，并把后续元素左移。
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

// registerTableConcat 注册 `table.concat`。
// 当前实现只面向最小 sequence 语义，并要求被拼接值可转成字符串。
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

// registerTableSort 注册 `table.sort`。
// 当前实现只排序最小 sequence 段，并支持可选比较函数。
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

// registerMathAbs 注册 `math.abs`，用于求绝对值。
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

// registerMathFloor 注册 `math.floor`，用于向下取整。
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

// registerMathCeil 注册 `math.ceil`，用于向上取整。
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

// registerMathMax 注册 `math.max`，用于从多个数值里选出最大值。
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

// registerMathMin 注册 `math.min`，用于从多个数值里选出最小值。
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

// registerMathSqrt 注册 `math.sqrt`，用于求平方根。
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

// registerMathPow 注册 `math.pow`，用于显式幂运算。
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

// registerMathRandom 注册 `math.random`。
// 当前使用 state 内部的确定性随机源，便于测试结果可重复。
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

// registerMathRandomSeed 注册 `math.randomseed`，用于重置 state 内部随机源。
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

// registerMathLog 注册 `math.log`，用于自然对数计算。
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

// registerMathExp 注册 `math.exp`，用于自然指数函数计算。
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

// registerMathSin 注册 `math.sin`，按弧度计算正弦值。
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

// registerMathCos 注册 `math.cos`，按弧度计算余弦值。
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

// registerStringGSub 注册最小 `string.gsub`。
// 当前只支持纯文本全局替换、字符串替换值和可选替换次数。
func (s *State) registerStringGSub() {
	_ = s.registerLibraryFunction("string", "gsub", func(args []Value) ([]Value, error) {
		if len(args) < 3 {
			return nil, fmt.Errorf("string.gsub expects at least 3 arguments")
		}

		text, err := requireStringArg(args[0], "string.gsub")
		if err != nil {
			return nil, err
		}

		pattern, err := requireStringArg(args[1], "string.gsub")
		if err != nil {
			return nil, err
		}

		replacement, err := requireStringArg(args[2], "string.gsub")
		if err != nil {
			return nil, err
		}

		replacementLimit := -1
		if len(args) > 3 {
			limit, err := builtinInteger(args[3], "string.gsub")
			if err != nil {
				return nil, err
			}

			if limit < 0 {
				return nil, fmt.Errorf("string.gsub expects non-negative replacement limit")
			}

			replacementLimit = limit
		}

		// TODO: 后续补齐 Lua 5.1 更完整的 `string.gsub` 语义，
		// 包括 pattern 匹配以及 table / function 替换器等形式。
		if replacementLimit == 0 {
			return []Value{
				{Type: ValueTypeString, Data: text},
				{Type: ValueTypeNumber, Data: float64(0)},
			}, nil
		}

		if pattern == "" {
			if replacementLimit < 0 {
				replacementLimit = len(text) + 1
			}

			result, replacements := replaceEmptyStringMatches(text, replacement, replacementLimit)
			return []Value{
				{Type: ValueTypeString, Data: result},
				{Type: ValueTypeNumber, Data: float64(replacements)},
			}, nil
		}

		result, replacements := replacePlainSubstrings(text, pattern, replacement, replacementLimit)
		return []Value{
			{Type: ValueTypeString, Data: result},
			{Type: ValueTypeNumber, Data: float64(replacements)},
		}, nil
	})
}

// registerStringFind 注册最小 `string.find`。
// 当前只支持纯文本查找和 Lua 风格起始下标。
func (s *State) registerStringFind() {
	_ = s.registerLibraryFunction("string", "find", func(args []Value) ([]Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("string.find expects at least 2 arguments")
		}

		text, err := requireStringArg(args[0], "string.find")
		if err != nil {
			return nil, err
		}

		pattern, err := requireStringArg(args[1], "string.find")
		if err != nil {
			return nil, err
		}

		startIndex := 1
		if len(args) > 2 {
			start, err := builtinInteger(args[2], "string.find")
			if err != nil {
				return nil, err
			}

			startIndex = start
		}

		// TODO: 后续补齐 Lua 5.1 的 pattern 匹配能力，
		// 当前实现始终按纯文本子串查找处理。
		searchStart := normalizeStringStart(len(text), startIndex)
		if pattern == "" {
			return []Value{
				{Type: ValueTypeNumber, Data: float64(searchStart)},
				{Type: ValueTypeNumber, Data: float64(searchStart - 1)},
			}, nil
		}

		if searchStart > len(text) {
			return []Value{NilValue()}, nil
		}

		matchOffset := strings.Index(text[searchStart-1:], pattern)
		if matchOffset < 0 {
			return []Value{NilValue()}, nil
		}

		matchStart := searchStart + matchOffset
		matchEnd := matchStart + len(pattern) - 1
		return []Value{
			{Type: ValueTypeNumber, Data: float64(matchStart)},
			{Type: ValueTypeNumber, Data: float64(matchEnd)},
		}, nil
	})
}

// registerStringMatch 注册最小 `string.match`。
// 当前只支持纯文本匹配提取和 Lua 风格起始下标。
func (s *State) registerStringMatch() {
	_ = s.registerLibraryFunction("string", "match", func(args []Value) ([]Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("string.match expects at least 2 arguments")
		}

		text, err := requireStringArg(args[0], "string.match")
		if err != nil {
			return nil, err
		}

		pattern, err := requireStringArg(args[1], "string.match")
		if err != nil {
			return nil, err
		}

		startIndex := 1
		if len(args) > 2 {
			start, err := builtinInteger(args[2], "string.match")
			if err != nil {
				return nil, err
			}

			startIndex = start
		}

		// TODO: 后续补齐 Lua 5.1 的 pattern 和 capture 语义，
		// 当前实现只提取纯文本子串匹配结果。
		searchStart := normalizeStringStart(len(text), startIndex)
		if pattern == "" {
			return []Value{{Type: ValueTypeString, Data: ""}}, nil
		}

		if searchStart > len(text) {
			return []Value{NilValue()}, nil
		}

		matchOffset := strings.Index(text[searchStart-1:], pattern)
		if matchOffset < 0 {
			return []Value{NilValue()}, nil
		}

		return []Value{{Type: ValueTypeString, Data: pattern}}, nil
	})
}

// registerStringLen 注册 `string.len`，用于获取字符串长度。
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

// registerStringSub 注册 `string.sub`。
// 当前按 Lua 风格索引规则提取子串，支持负索引。
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

// registerStringLower 注册 `string.lower`。
// 当前通过 Go 的字符串工具做基础大小写折叠。
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

// registerStringUpper 注册 `string.upper`。
// 当前通过 Go 的字符串工具做基础大小写折叠。
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

// registerStringRep 注册 `string.rep`，用于重复构造字符串。
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

// registerStringReverse 注册 `string.reverse`，用于反转字符串。
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

// registerStringByte 注册 `string.byte`。
// 当前按面向字节的最小字符串子集返回指定范围内的字节值。
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

// registerStringChar 注册 `string.char`，用于把字节值组装成字符串。
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

// registerContextualLibraryFunction 在某个库表下注册需要访问 executor 上下文的内建函数。
// 这类函数除了参数外，还需要读取或操作当前执行器状态。
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

// tableSortLess 执行一次 `table.sort` 比较。
// 它会优先使用自定义比较器；如果没有提供，则回退到运行时自身的 `<` 语义。
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

// requireStringArg 取出一个内建函数需要的字符串参数，并验证运行时类型是否正确。
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

// normalizeStringStart 把 Lua 风格字符串起始索引转换为夹紧后的 1 基位置。
func normalizeStringStart(length int, start int) int {
	if start < 0 {
		start = length + start + 1
	}
	if start < 1 {
		start = 1
	}
	if start > length+1 {
		start = length + 1
	}
	return start
}

// normalizeStringRange 把 Lua 风格的字符串区间边界转换成夹紧后的 1 基闭区间。
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

// replacePlainSubstrings 按从左到右的顺序执行纯文本子串替换，并支持可选替换次数限制。
func replacePlainSubstrings(text string, pattern string, replacement string, limit int) (string, int) {
	if pattern == "" {
		return text, 0
	}

	if limit < 0 {
		return strings.ReplaceAll(text, pattern, replacement), strings.Count(text, pattern)
	}

	var builder strings.Builder
	searchStart := 0
	replacements := 0
	for replacements < limit {
		matchOffset := strings.Index(text[searchStart:], pattern)
		if matchOffset < 0 {
			break
		}

		matchStart := searchStart + matchOffset
		builder.WriteString(text[searchStart:matchStart])
		builder.WriteString(replacement)
		searchStart = matchStart + len(pattern)
		replacements++
	}

	builder.WriteString(text[searchStart:])
	return builder.String(), replacements
}

// replaceEmptyStringMatches 实现空模式下最小 `gsub` 行为。
// 它会在字符串边界之间插入替换值，并返回实际替换次数。
func replaceEmptyStringMatches(text string, replacement string, limit int) (string, int) {
	if limit <= 0 {
		return text, 0
	}

	var builder strings.Builder
	replacements := 0
	if replacements < limit {
		builder.WriteString(replacement)
		replacements++
	}

	for index := 0; index < len(text); index++ {
		builder.WriteByte(text[index])
		if replacements < limit {
			builder.WriteString(replacement)
			replacements++
		}
	}

	return builder.String(), replacements
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

// registerBuiltinError 注册最小 `error` 内建函数。
func (s *State) registerBuiltinError() {
	_ = s.RegisterFunction("error", func(args []Value) ([]Value, error) {
		if len(args) < 1 {
			return nil, fmt.Errorf("error")
		}

		return nil, fmt.Errorf("%s", valueToString(args[0]))
	})
}

// registerBuiltinNext 注册 `next`。
// 它是 generic for、`pairs` 等最小迭代能力使用的基础原语。
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

// registerBuiltinPairs 注册最小 `pairs`，其实现直接基于 `next`。
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

// registerBuiltinIPairs 注册最小 `ipairs` 顺序数组迭代器。
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

// registerBuiltinPCall 注册最小 `pcall` 保护调用内建函数。
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

// registerBuiltinRawEqual 注册最小 `rawequal`。
// 它会绕过 `__eq` 元方法，直接执行原始相等性判断。
func (s *State) registerBuiltinRawEqual() {
	_ = s.RegisterFunction("rawequal", func(args []Value) ([]Value, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("rawequal expects 2 arguments")
		}

		return []Value{{Type: ValueTypeBoolean, Data: valuesEqual(args[0], args[1])}}, nil
	})
}

// registerBuiltinXPCall 注册最小 Lua 5.1 `xpcall`。
// 当前支持传入错误处理函数，并在失败时走对应回调。
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

// registerBuiltinGetMetatable 注册最小 `getmetatable`。
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

// registerBuiltinSetMetatable 注册最小 `setmetatable`。
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

// registerBuiltinRawGet 注册 `rawget`。
// 它会直接读取 table 字段，绕过 `__index` 元方法逻辑。
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

// registerBuiltinRawSet 注册 `rawset`。
// 它会直接写入 table 字段，绕过 `__newindex` 元方法逻辑。
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

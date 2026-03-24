package vm

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/yunfanli-dev/aLuavmWithGo/internal/ir"
)

type executionResult struct {
	returnValues []Value
	breakLoop    bool
}

type executor struct {
	scopes         []map[string]*valueCell
	varargs        []Value
	stepLimit      int
	remainingSteps int
	ctx            context.Context
}

// executeProgram 执行当前 IR 子集程序，并返回脚本显式产生的返回值。
// 它会创建执行器、驱动语句求值，并把最终 `return` 结果整理成统一结构返回。
func executeProgram(ctx context.Context, state *State, program *ir.Program) (*executionResult, error) {
	if program == nil {
		return nil, fmt.Errorf("execute nil IR program")
	}
	if state == nil {
		return nil, fmt.Errorf("execute nil VM state")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	exec := &executor{
		scopes:         []map[string]*valueCell{state.globals},
		stepLimit:      state.stepLimit,
		remainingSteps: state.stepLimit,
		ctx:            ctx,
	}

	for _, statement := range program.Statements {
		result, done, err := exec.executeStatement(statement)
		if err != nil {
			return nil, err
		}

		if done {
			if result.breakLoop {
				return nil, fmt.Errorf("break outside loop")
			}

			return result, nil
		}
	}

	return &executionResult{}, nil
}

func (e *executor) executeStatement(statement ir.Statement) (*executionResult, bool, error) {
	if err := e.consumeStep(); err != nil {
		return nil, false, err
	}

	switch node := statement.(type) {
	case *ir.CallStatement:
		_, err := e.evaluateCallExpression(node.Call)
		return nil, false, err
	case *ir.AssignStatement:
		values, err := e.evaluateExpressionList(node.Values)
		if err != nil {
			return nil, false, err
		}

		for index, target := range node.Targets {
			value := NilValue()
			if index < len(values) {
				value = values[index]
			}

			if err := e.assignTarget(target, value); err != nil {
				return nil, false, err
			}
		}

		return nil, false, nil
	case *ir.LocalAssignStatement:
		values, err := e.evaluateExpressionList(node.Values)
		if err != nil {
			return nil, false, err
		}

		for index, name := range node.Names {
			value := NilValue()
			if index < len(values) {
				value = values[index]
			}

			e.defineLocal(name, value)
		}

		return nil, false, nil
	case *ir.DoStatement:
		return e.executeBlock(node.Body)
	case *ir.BreakStatement:
		return &executionResult{breakLoop: true}, true, nil
	case *ir.FunctionDeclarationStatement:
		e.assign(node.Name, e.makeUserFunctionValue(node.Name, node.Parameters, node.IsVararg, node.Body))

		return nil, false, nil
	case *ir.LocalFunctionDeclarationStatement:
		placeholder := NilValue()
		e.defineLocal(node.Name, placeholder)
		functionValue := e.makeUserFunctionValue(node.Name, node.Parameters, node.IsVararg, node.Body)
		e.assign(node.Name, functionValue)

		if userFn, ok := functionValue.Data.(*userFunction); ok {
			if cell, ok := e.currentScope()[node.Name]; ok {
				userFn.captured[node.Name] = cell
			}
		}

		return nil, false, nil
	case *ir.IfStatement:
		return e.executeIfStatement(node)
	case *ir.WhileStatement:
		return e.executeWhileStatement(node)
	case *ir.RepeatStatement:
		return e.executeRepeatStatement(node)
	case *ir.NumericForStatement:
		return e.executeNumericForStatement(node)
	case *ir.GenericForStatement:
		return e.executeGenericForStatement(node)
	case *ir.ReturnStatement:
		values, err := e.evaluateExpressionList(node.Values)
		if err != nil {
			return nil, false, err
		}

		return &executionResult{returnValues: values}, true, nil
	default:
		return nil, false, fmt.Errorf("execute unsupported IR statement %T", statement)
	}
}

func (e *executor) executeIfStatement(statement *ir.IfStatement) (*executionResult, bool, error) {
	for _, clause := range statement.Clauses {
		condition, err := e.evaluateExpression(clause.Condition)
		if err != nil {
			return nil, false, err
		}

		if isTruthy(condition) {
			return e.executeBlock(clause.Body)
		}
	}

	return e.executeBlock(statement.ElseBody)
}

func (e *executor) executeWhileStatement(statement *ir.WhileStatement) (*executionResult, bool, error) {
	for {
		condition, err := e.evaluateExpression(statement.Condition)
		if err != nil {
			return nil, false, err
		}

		if !isTruthy(condition) {
			return nil, false, nil
		}

		result, done, err := e.executeBlock(statement.Body)
		if err != nil {
			return nil, false, err
		}

		if done {
			if result.breakLoop {
				return nil, false, nil
			}

			return result, true, nil
		}
	}
}

// executeRepeatStatement 执行 `repeat ... until` 循环。
// 终止条件会在循环体作用域内求值，以保持与 Lua 作用域规则一致。
func (e *executor) executeRepeatStatement(statement *ir.RepeatStatement) (*executionResult, bool, error) {
	for {
		e.pushScope()
		for _, child := range statement.Body {
			result, done, err := e.executeStatement(child)
			if err != nil {
				e.popScope()
				return nil, false, err
			}

			if done {
				if result.breakLoop {
					e.popScope()
					return nil, false, nil
				}

				e.popScope()
				return result, true, nil
			}
		}

		condition, err := e.evaluateExpression(statement.Condition)
		e.popScope()
		if err != nil {
			return nil, false, err
		}

		if isTruthy(condition) {
			return nil, false, nil
		}
	}
}

// executeNumericForStatement 执行当前支持的数值 for 循环。
// 它会先求值起始值、终止值和步长，再按 Lua 风格边界规则推进循环变量。
func (e *executor) executeNumericForStatement(statement *ir.NumericForStatement) (*executionResult, bool, error) {
	startValue, err := e.evaluateExpression(statement.Start)
	if err != nil {
		return nil, false, err
	}

	limitValue, err := e.evaluateExpression(statement.Limit)
	if err != nil {
		return nil, false, err
	}

	stepValue, err := e.evaluateExpression(statement.Step)
	if err != nil {
		return nil, false, err
	}

	current, err := requireNumber(startValue, "for")
	if err != nil {
		return nil, false, err
	}

	limit, err := requireNumber(limitValue, "for")
	if err != nil {
		return nil, false, err
	}

	step, err := requireNumber(stepValue, "for")
	if err != nil {
		return nil, false, err
	}

	if step == 0 {
		return nil, false, fmt.Errorf("numeric for step cannot be zero")
	}

	e.pushScope()
	defer e.popScope()
	e.defineLocal(statement.Name, Value{Type: ValueTypeNumber, Data: current})

	for numericForContinues(current, limit, step) {
		e.assign(statement.Name, Value{Type: ValueTypeNumber, Data: current})
		result, done, err := e.executeBlock(statement.Body)
		if err != nil {
			return nil, false, err
		}

		if done {
			if result.breakLoop {
				return nil, false, nil
			}

			return result, true, nil
		}

		current += step
	}

	return nil, false, nil
}

// executeGenericForStatement 执行当前支持的 generic for 循环。
// 它基于 iterator、state 和 control value 这三元组驱动循环展开。
func (e *executor) executeGenericForStatement(statement *ir.GenericForStatement) (*executionResult, bool, error) {
	iteratorValues, err := e.evaluateExpressionList(statement.Iterators)
	if err != nil {
		return nil, false, err
	}

	iterator := NilValue()
	stateValue := NilValue()
	controlValue := NilValue()
	if len(iteratorValues) > 0 {
		iterator = iteratorValues[0]
	}
	if len(iteratorValues) > 1 {
		stateValue = iteratorValues[1]
	}
	if len(iteratorValues) > 2 {
		controlValue = iteratorValues[2]
	}

	for {
		returnValues, err := e.callFunctionValue(iterator, []Value{stateValue, controlValue})
		if err != nil {
			return nil, false, err
		}

		if len(returnValues) == 0 || returnValues[0].Type == ValueTypeNil {
			return nil, false, nil
		}

		controlValue = returnValues[0]

		e.pushScope()
		for index, name := range statement.Names {
			value := NilValue()
			if index < len(returnValues) {
				value = returnValues[index]
			}

			e.defineLocal(name, value)
		}

		result, done, err := e.executeBlock(statement.Body)
		e.popScope()
		if err != nil {
			return nil, false, err
		}

		if done {
			if result.breakLoop {
				return nil, false, nil
			}

			return result, true, nil
		}
	}
}

func (e *executor) executeBlock(statements []ir.Statement) (*executionResult, bool, error) {
	e.pushScope()
	defer e.popScope()

	for _, statement := range statements {
		result, done, err := e.executeStatement(statement)
		if err != nil {
			return nil, false, err
		}

		if done {
			return result, true, nil
		}
	}

	return nil, false, nil
}

func (e *executor) evaluateExpressionList(expressions []ir.Expression) ([]Value, error) {
	values := make([]Value, 0, len(expressions))
	for index, expression := range expressions {
		expanded, err := e.evaluateExpressionValues(expression, index == len(expressions)-1)
		if err != nil {
			return nil, err
		}

		values = append(values, expanded...)
	}

	return values, nil
}

func (e *executor) evaluateExpressionValues(expression ir.Expression, expandCall bool) ([]Value, error) {
	if expandCall {
		if call, ok := expression.(*ir.CallExpression); ok {
			return e.evaluateCallExpressionValues(call)
		}

		if _, ok := expression.(*ir.VarargExpression); ok {
			return append([]Value(nil), e.varargs...), nil
		}
	}

	value, err := e.evaluateExpression(expression)
	if err != nil {
		return nil, err
	}

	return []Value{value}, nil
}

func (e *executor) evaluateExpression(expression ir.Expression) (Value, error) {
	if err := e.consumeStep(); err != nil {
		return NilValue(), err
	}

	switch node := expression.(type) {
	case *ir.IdentifierExpression:
		value, ok := e.lookup(node.Name)
		if !ok {
			return NilValue(), nil
		}

		return value, nil
	case *ir.NilExpression:
		return NilValue(), nil
	case *ir.BooleanExpression:
		return Value{Type: ValueTypeBoolean, Data: node.Value}, nil
	case *ir.NumberExpression:
		number, err := parseNumberLiteral(node.Literal)
		if err != nil {
			return NilValue(), fmt.Errorf("parse number literal %q: %w", node.Literal, err)
		}

		return Value{Type: ValueTypeNumber, Data: number}, nil
	case *ir.StringExpression:
		return Value{Type: ValueTypeString, Data: node.Value}, nil
	case *ir.CallExpression:
		return e.evaluateCallExpression(node)
	case *ir.IndexExpression:
		target, err := e.evaluateExpression(node.Target)
		if err != nil {
			return NilValue(), err
		}

		if target.Type != ValueTypeTable {
			return NilValue(), fmt.Errorf("attempt to index non-table value of type %s", target.Type)
		}

		index, err := e.evaluateExpression(node.Index)
		if err != nil {
			return NilValue(), err
		}

		tableValue, ok := target.Data.(*table)
		if !ok {
			return NilValue(), fmt.Errorf("invalid table payload %T", target.Data)
		}

		return e.readTableIndex(tableValue, index)
	case *ir.TableConstructorExpression:
		tableValue := newTable()
		for index, field := range node.Fields {
			key, err := e.evaluateExpression(field.Key)
			if err != nil {
				return NilValue(), err
			}

			if field.IsListField && index == len(node.Fields)-1 {
				values, err := e.evaluateExpressionValues(field.Value, true)
				if err != nil {
					return NilValue(), err
				}

				baseIndex, err := requireNumber(key, "table constructor list field")
				if err != nil {
					return NilValue(), err
				}

				if len(values) == 0 {
					values = []Value{NilValue()}
				}

				for offset, value := range values {
					listKey := Value{Type: ValueTypeNumber, Data: baseIndex + float64(offset)}
					if err := tableValue.set(listKey, value); err != nil {
						return NilValue(), err
					}
				}

				continue
			}

			value, err := e.evaluateExpression(field.Value)
			if err != nil {
				return NilValue(), err
			}

			if err := tableValue.set(key, value); err != nil {
				return NilValue(), err
			}
		}

		return Value{Type: ValueTypeTable, Data: tableValue}, nil
	case *ir.FunctionExpression:
		return e.makeUserFunctionValue("", node.Parameters, node.IsVararg, node.Body), nil
	case *ir.VarargExpression:
		if len(e.varargs) == 0 {
			return NilValue(), nil
		}

		return e.varargs[0], nil
	case *ir.ParenthesizedExpression:
		return e.evaluateExpression(node.Inner)
	case *ir.UnaryExpression:
		return e.evaluateUnaryExpression(node)
	case *ir.BinaryExpression:
		return e.evaluateBinaryExpression(node)
	default:
		return NilValue(), fmt.Errorf("evaluate unsupported IR expression %T", expression)
	}
}

// consumeStep 在启用步数预算时扣减当前脚本的剩余执行步数。
// 当预算耗尽时会直接返回错误，用于阻止明显的无限循环长期占用执行线程。
func (e *executor) consumeStep() error {
	if e.ctx != nil {
		if err := e.ctx.Err(); err != nil {
			return err
		}
	}

	if e.stepLimit <= 0 {
		return nil
	}

	if e.remainingSteps <= 0 {
		return fmt.Errorf("execution step limit exceeded")
	}

	e.remainingSteps--
	return nil
}

func (e *executor) makeUserFunctionValue(name string, parameters []string, isVararg bool, body []ir.Statement) Value {
	return Value{
		Type: ValueTypeFunction,
		Data: &userFunction{
			name:       name,
			parameters: append([]string(nil), parameters...),
			isVararg:   isVararg,
			body:       append([]ir.Statement(nil), body...),
			captured:   e.snapshotVisibleCells(),
		},
	}
}

func (e *executor) assignTarget(target ir.Expression, value Value) error {
	switch node := target.(type) {
	case *ir.IdentifierExpression:
		e.assign(node.Name, value)
		return nil
	case *ir.IndexExpression:
		targetValue, err := e.evaluateExpression(node.Target)
		if err != nil {
			return err
		}

		if targetValue.Type != ValueTypeTable {
			return fmt.Errorf("attempt to index-assign non-table value of type %s", targetValue.Type)
		}

		index, err := e.evaluateExpression(node.Index)
		if err != nil {
			return err
		}

		tableValue, ok := targetValue.Data.(*table)
		if !ok {
			return fmt.Errorf("invalid table payload %T", targetValue.Data)
		}

		return e.writeTableIndex(tableValue, index, value)
	default:
		return fmt.Errorf("unsupported assignment target %T", target)
	}
}

// readTableIndex 读取一个 table 字段，并在需要时应用最小 `__index` 元方法回退。
// 这条路径同时服务于方括号索引和点语法字段读取。
func (e *executor) readTableIndex(tableValue *table, index Value) (Value, error) {
	value, exists, err := tableValue.get(index)
	if err != nil {
		return NilValue(), err
	}

	if exists {
		return value, nil
	}

	metatable := tableValue.getMetatable()
	if metatable == nil {
		return NilValue(), nil
	}

	metaIndex, exists, err := metatable.get(Value{Type: ValueTypeString, Data: "__index"})
	if err != nil {
		return NilValue(), err
	}

	if !exists {
		return NilValue(), nil
	}

	switch metaIndex.Type {
	case ValueTypeTable:
		fallbackTable, ok := metaIndex.Data.(*table)
		if !ok {
			return NilValue(), fmt.Errorf("invalid __index table payload %T", metaIndex.Data)
		}

		return e.readTableIndex(fallbackTable, index)
	case ValueTypeFunction:
		returnValues, err := e.callFunctionValue(metaIndex, []Value{
			{Type: ValueTypeTable, Data: tableValue},
			index,
		})
		if err != nil {
			return NilValue(), err
		}

		if len(returnValues) == 0 {
			return NilValue(), nil
		}

		return returnValues[0], nil
	default:
		// TODO: 后续按需要补齐 Lua 5.1 更完整的 `__index` 形态，
		// 当前只覆盖最小可用回退路径。
		return NilValue(), nil
	}
}

// writeTableIndex 写入一个 table 字段，并在需要时应用最小 `__newindex` 元方法回退。
// 这条路径统一处理直接赋值和可能触发的元方法转发。
func (e *executor) writeTableIndex(tableValue *table, index Value, value Value) error {
	_, exists, err := tableValue.get(index)
	if err != nil {
		return err
	}

	if exists {
		return tableValue.set(index, value)
	}

	metatable := tableValue.getMetatable()
	if metatable == nil {
		return tableValue.set(index, value)
	}

	metaNewIndex, exists, err := metatable.get(Value{Type: ValueTypeString, Data: "__newindex"})
	if err != nil {
		return err
	}

	if !exists {
		return tableValue.set(index, value)
	}

	switch metaNewIndex.Type {
	case ValueTypeTable:
		fallbackTable, ok := metaNewIndex.Data.(*table)
		if !ok {
			return fmt.Errorf("invalid __newindex table payload %T", metaNewIndex.Data)
		}

		return e.writeTableIndex(fallbackTable, index, value)
	case ValueTypeFunction:
		_, err := e.callFunctionValue(metaNewIndex, []Value{
			{Type: ValueTypeTable, Data: tableValue},
			index,
			value,
		})
		return err
	default:
		// TODO: 后续按需要补齐 Lua 5.1 更完整的 `__newindex` 形态，
		// 当前只覆盖最小可用回退路径。
		return tableValue.set(index, value)
	}
}

func (e *executor) evaluateCallExpression(expression *ir.CallExpression) (Value, error) {
	values, err := e.evaluateCallExpressionValues(expression)
	if err != nil {
		return NilValue(), err
	}

	if len(values) == 0 {
		return NilValue(), nil
	}

	return values[0], nil
}

func (e *executor) evaluateCallExpressionValues(expression *ir.CallExpression) ([]Value, error) {
	if expression.Receiver != nil {
		receiver, err := e.evaluateExpression(expression.Receiver)
		if err != nil {
			return nil, err
		}

		method, err := e.lookupMethod(receiver, expression.Method)
		if err != nil {
			return nil, err
		}

		arguments, err := e.evaluateExpressionList(expression.Arguments)
		if err != nil {
			return nil, err
		}

		callArgs := make([]Value, 0, len(arguments)+1)
		callArgs = append(callArgs, receiver)
		callArgs = append(callArgs, arguments...)
		return e.callFunctionValue(method, callArgs)
	}

	callee, err := e.evaluateExpression(expression.Callee)
	if err != nil {
		return nil, err
	}

	arguments, err := e.evaluateExpressionList(expression.Arguments)
	if err != nil {
		return nil, err
	}

	return e.callFunctionValue(callee, arguments)
}

// lookupMethod 基于现有索引语义从接收者上解析一个方法名。
// 这让 `obj:method(...)` 可以复用 table 读取和 metatable 回退的既有逻辑。
func (e *executor) lookupMethod(receiver Value, method string) (Value, error) {
	if receiver.Type != ValueTypeTable {
		return NilValue(), fmt.Errorf("attempt to call method %q on non-table value of type %s", method, receiver.Type)
	}

	tableValue, ok := receiver.Data.(*table)
	if !ok {
		return NilValue(), fmt.Errorf("invalid table payload %T", receiver.Data)
	}

	return e.readTableIndex(tableValue, Value{Type: ValueTypeString, Data: method})
}

func (e *executor) pushScope() {
	e.scopes = append(e.scopes, map[string]*valueCell{})
}

func (e *executor) currentScope() map[string]*valueCell {
	return e.scopes[len(e.scopes)-1]
}

func (e *executor) popScope() {
	if len(e.scopes) <= 1 {
		e.scopes[0] = map[string]*valueCell{}
		return
	}

	e.scopes = e.scopes[:len(e.scopes)-1]
}

func (e *executor) callFunctionValue(callee Value, arguments []Value) ([]Value, error) {
	if callee.Type != ValueTypeFunction {
		if callee.Type == ValueTypeTable {
			tableValue, ok := callee.Data.(*table)
			if !ok {
				return nil, fmt.Errorf("invalid table payload %T", callee.Data)
			}

			metaCall, exists, err := e.lookupMetamethod(callee, "__call")
			if err != nil {
				return nil, err
			}

			if exists {
				callArgs := make([]Value, 0, len(arguments)+1)
				callArgs = append(callArgs, Value{Type: ValueTypeTable, Data: tableValue})
				callArgs = append(callArgs, arguments...)
				return e.callFunctionValue(metaCall, callArgs)
			}
		}

		return nil, fmt.Errorf("attempt to call non-function value of type %s", callee.Type)
	}

	switch functionValue := callee.Data.(type) {
	case *userFunction:
		return e.callUserFunction(functionValue, arguments)
	case *nativeFunction:
		return e.callNativeFunction(functionValue, arguments)
	default:
		return nil, fmt.Errorf("invalid function payload %T", callee.Data)
	}
}

func (e *executor) callUserFunction(functionValue *userFunction, arguments []Value) ([]Value, error) {
	if functionValue == nil {
		return nil, fmt.Errorf("call nil function")
	}

	savedScopes := e.scopes
	savedVarargs := e.varargs
	globalScope := savedScopes[0]
	capturedScope := copyCellMap(functionValue.captured)
	e.scopes = []map[string]*valueCell{globalScope, capturedScope, {}}
	defer func() {
		e.scopes = savedScopes
		e.varargs = savedVarargs
	}()

	for index, parameter := range functionValue.parameters {
		value := NilValue()
		if index < len(arguments) {
			value = arguments[index]
		}

		e.defineLocal(parameter, value)
	}

	if functionValue.isVararg && len(arguments) > len(functionValue.parameters) {
		e.varargs = append([]Value(nil), arguments[len(functionValue.parameters):]...)
	} else {
		e.varargs = nil
	}

	for _, statement := range functionValue.body {
		result, done, err := e.executeStatement(statement)
		if err != nil {
			return nil, err
		}

		if done {
			return append([]Value(nil), result.returnValues...), nil
		}
	}

	return nil, nil
}

func (e *executor) callNativeFunction(functionValue *nativeFunction, arguments []Value) ([]Value, error) {
	if functionValue == nil || (functionValue.fn == nil && functionValue.contextualImpl == nil) {
		return nil, fmt.Errorf("call nil native function")
	}

	if functionValue.contextualImpl != nil {
		return functionValue.contextualImpl(e, arguments)
	}

	return functionValue.fn(arguments)
}

// lookupMetamethod 从当前运行时值上解析一个受支持的 metatable 字段。
// 当前实现只覆盖已落地的最小元方法集合。
func (e *executor) lookupMetamethod(value Value, field string) (Value, bool, error) {
	if value.Type != ValueTypeTable {
		return NilValue(), false, nil
	}

	tableValue, ok := value.Data.(*table)
	if !ok {
		return NilValue(), false, fmt.Errorf("invalid table payload %T", value.Data)
	}

	metatable := tableValue.getMetatable()
	if metatable == nil {
		return NilValue(), false, nil
	}

	return metatable.get(Value{Type: ValueTypeString, Data: field})
}

// valueToString 把一个运行时值渲染成字符串。
// 对 table 会优先尝试最小 `__tostring` 元方法钩子，再回退到默认格式。
func (e *executor) valueToString(value Value) (string, error) {
	metaToString, exists, err := e.lookupMetamethod(value, "__tostring")
	if err != nil {
		return "", err
	}

	if exists {
		returnValues, err := e.callFunctionValue(metaToString, []Value{value})
		if err != nil {
			return "", err
		}

		if len(returnValues) == 0 {
			return "", nil
		}

		return valueToString(returnValues[0]), nil
	}

	return valueToString(value), nil
}

func (e *executor) defineLocal(name string, value Value) {
	e.scopes[len(e.scopes)-1][name] = &valueCell{value: value}
}

func (e *executor) assign(name string, value Value) {
	for index := len(e.scopes) - 1; index >= 0; index-- {
		if cell, ok := e.scopes[index][name]; ok {
			cell.value = value
			return
		}
	}

	e.scopes[len(e.scopes)-1][name] = &valueCell{value: value}
}

func (e *executor) lookup(name string) (Value, bool) {
	for index := len(e.scopes) - 1; index >= 0; index-- {
		cell, ok := e.scopes[index][name]
		if ok {
			return cell.value, true
		}
	}

	return NilValue(), false
}

func (e *executor) snapshotVisibleCells() map[string]*valueCell {
	snapshot := make(map[string]*valueCell)
	for _, scope := range e.scopes {
		for name, cell := range scope {
			snapshot[name] = cell
		}
	}

	return snapshot
}

func copyCellMap(input map[string]*valueCell) map[string]*valueCell {
	if input == nil {
		return map[string]*valueCell{}
	}

	output := make(map[string]*valueCell, len(input))
	for key, value := range input {
		output[key] = value
	}

	return output
}

func (e *executor) evaluateUnaryExpression(expression *ir.UnaryExpression) (Value, error) {
	operand, err := e.evaluateExpression(expression.Operand)
	if err != nil {
		return NilValue(), err
	}

	switch expression.Operator {
	case "-":
		number, err := requireNumber(operand, "unary '-'")
		if err != nil {
			return e.evaluateUnaryMetamethod(operand, "__unm", err)
		}

		return Value{Type: ValueTypeNumber, Data: -number}, nil
	case "not":
		return Value{Type: ValueTypeBoolean, Data: !isTruthy(operand)}, nil
	case "#":
		if operand.Type == ValueTypeString {
			return Value{Type: ValueTypeNumber, Data: float64(len(operand.Data.(string)))}, nil
		}

		if operand.Type == ValueTypeTable {
			tableValue, ok := operand.Data.(*table)
			if !ok {
				return NilValue(), fmt.Errorf("invalid table payload %T", operand.Data)
			}

			length, err := tableValue.sequenceLength()
			if err != nil {
				return NilValue(), err
			}

			return Value{Type: ValueTypeNumber, Data: float64(length)}, nil
		}

		return NilValue(), fmt.Errorf("operator '#' expects string or table operand, got %s", operand.Type)
	default:
		return NilValue(), fmt.Errorf("unsupported unary operator %q", expression.Operator)
	}
}

func (e *executor) evaluateBinaryExpression(expression *ir.BinaryExpression) (Value, error) {
	switch expression.Operator {
	case "and":
		left, err := e.evaluateExpression(expression.Left)
		if err != nil {
			return NilValue(), err
		}

		if !isTruthy(left) {
			return left, nil
		}

		return e.evaluateExpression(expression.Right)
	case "or":
		left, err := e.evaluateExpression(expression.Left)
		if err != nil {
			return NilValue(), err
		}

		if isTruthy(left) {
			return left, nil
		}

		return e.evaluateExpression(expression.Right)
	}

	left, err := e.evaluateExpression(expression.Left)
	if err != nil {
		return NilValue(), err
	}

	right, err := e.evaluateExpression(expression.Right)
	if err != nil {
		return NilValue(), err
	}

	switch expression.Operator {
	case "+":
		return e.evaluateBinaryArithmetic(left, right, expression.Operator, "__add", func(a, b float64) float64 { return a + b })
	case "-":
		return e.evaluateBinaryArithmetic(left, right, expression.Operator, "__sub", func(a, b float64) float64 { return a - b })
	case "*":
		return e.evaluateBinaryArithmetic(left, right, expression.Operator, "__mul", func(a, b float64) float64 { return a * b })
	case "/":
		return e.evaluateBinaryArithmetic(left, right, expression.Operator, "__div", func(a, b float64) float64 { return a / b })
	case "%":
		return e.evaluateBinaryArithmetic(left, right, expression.Operator, "__mod", math.Mod)
	case "^":
		return e.evaluateBinaryArithmetic(left, right, expression.Operator, "__pow", math.Pow)
	case "..":
		if left.Type == ValueTypeString || left.Type == ValueTypeNumber || right.Type == ValueTypeString || right.Type == ValueTypeNumber {
			return Value{Type: ValueTypeString, Data: valueToString(left) + valueToString(right)}, nil
		}

		return e.evaluateBinaryMetamethod(left, right, "__concat", fmt.Errorf("operator %q expects string-like operands, got %s and %s", expression.Operator, left.Type, right.Type))
	case "<":
		return e.evaluateOrderedComparison(left, right, expression.Operator, "__lt", func(a, b float64) bool { return a < b })
	case "<=":
		return e.evaluateOrderedComparison(left, right, expression.Operator, "__le", func(a, b float64) bool { return a <= b })
	case ">":
		return e.evaluateOrderedComparison(right, left, expression.Operator, "__lt", func(a, b float64) bool { return a < b })
	case ">=":
		return e.evaluateOrderedComparison(right, left, expression.Operator, "__le", func(a, b float64) bool { return a <= b })
	case "==":
		return e.evaluateEquality(left, right)
	case "~=":
		value, err := e.evaluateEquality(left, right)
		if err != nil {
			return NilValue(), err
		}

		return Value{Type: ValueTypeBoolean, Data: !value.Data.(bool)}, nil
	default:
		return NilValue(), fmt.Errorf("unsupported binary operator %q", expression.Operator)
	}
}

// evaluateUnaryMetamethod 在一元运算直接类型检查失败时尝试走元方法回退。
// 当前只对已支持的一元元方法做最小兼容。
func (e *executor) evaluateUnaryMetamethod(operand Value, metamethod string, cause error) (Value, error) {
	metaFn, exists, err := e.lookupMetamethod(operand, metamethod)
	if err != nil {
		return NilValue(), err
	}

	if !exists {
		return NilValue(), cause
	}

	returnValues, err := e.callFunctionValue(metaFn, []Value{operand})
	if err != nil {
		return NilValue(), err
	}

	if len(returnValues) == 0 {
		return NilValue(), nil
	}

	return returnValues[0], nil
}

// evaluateBinaryArithmetic 先尝试直接做数值算术运算。
// 如果操作数不满足直接计算条件，再回退到已支持的二元元方法路径。
func (e *executor) evaluateBinaryArithmetic(left, right Value, operator string, metamethod string, fn func(float64, float64) float64) (Value, error) {
	value, err := numericBinary(left, right, operator, fn)
	if err == nil {
		return value, nil
	}

	return e.evaluateBinaryMetamethod(left, right, metamethod, err)
}

// evaluateBinaryMetamethod 按“先左后右”的顺序尝试解析并调用二元元方法。
// 这与 Lua 常见的双操作数元方法查找顺序保持一致。
func (e *executor) evaluateBinaryMetamethod(left, right Value, metamethod string, cause error) (Value, error) {
	metaFn, exists, err := e.lookupBinaryMetamethod(left, right, metamethod)
	if err != nil {
		return NilValue(), err
	}

	if !exists {
		if cause == nil {
			return NilValue(), nil
		}

		return NilValue(), cause
	}

	returnValues, err := e.callFunctionValue(metaFn, []Value{left, right})
	if err != nil {
		return NilValue(), err
	}

	if len(returnValues) == 0 {
		return NilValue(), nil
	}

	return returnValues[0], nil
}

// evaluateOrderedComparison 先尝试直接做数值比较，再回退到已支持的比较元方法。
// 当前主要覆盖 `<`、`<=` 及其派生比较链路。
func (e *executor) evaluateOrderedComparison(left, right Value, operator string, metamethod string, fn func(float64, float64) bool) (Value, error) {
	value, err := comparisonBinary(left, right, operator, fn)
	if err == nil {
		return value, nil
	}

	return e.evaluateBinaryMetamethod(left, right, metamethod, err)
}

// evaluateEquality 先执行原始相等性判断，再在需要时回退到最小 `__eq` 元方法路径。
// 当前元方法相等性主要面向 table 值。
func (e *executor) evaluateEquality(left, right Value) (Value, error) {
	if valuesEqual(left, right) {
		return Value{Type: ValueTypeBoolean, Data: true}, nil
	}

	if left.Type != right.Type || left.Type != ValueTypeTable {
		return Value{Type: ValueTypeBoolean, Data: false}, nil
	}

	value, err := e.evaluateBinaryMetamethod(left, right, "__eq", nil)
	if err != nil {
		return NilValue(), err
	}

	if value.Type == ValueTypeNil {
		return Value{Type: ValueTypeBoolean, Data: false}, nil
	}

	return Value{Type: ValueTypeBoolean, Data: isTruthy(value)}, nil
}

// lookupBinaryMetamethod 按先左后右的顺序查找二元元方法入口。
// 这样可以把 Lua 的常见元方法分派规则集中收敛在一个位置处理。
func (e *executor) lookupBinaryMetamethod(left, right Value, field string) (Value, bool, error) {
	metaFn, exists, err := e.lookupMetamethod(left, field)
	if err != nil {
		return NilValue(), false, err
	}

	if exists {
		return metaFn, true, nil
	}

	return e.lookupMetamethod(right, field)
}

func numericBinary(left, right Value, operator string, fn func(float64, float64) float64) (Value, error) {
	leftNumber, err := requireNumber(left, operator)
	if err != nil {
		return NilValue(), err
	}

	rightNumber, err := requireNumber(right, operator)
	if err != nil {
		return NilValue(), err
	}

	return Value{Type: ValueTypeNumber, Data: fn(leftNumber, rightNumber)}, nil
}

func comparisonBinary(left, right Value, operator string, fn func(float64, float64) bool) (Value, error) {
	leftNumber, err := requireNumber(left, operator)
	if err != nil {
		return NilValue(), err
	}

	rightNumber, err := requireNumber(right, operator)
	if err != nil {
		return NilValue(), err
	}

	return Value{Type: ValueTypeBoolean, Data: fn(leftNumber, rightNumber)}, nil
}

// numericForContinues 根据步长正负应用 Lua 风格的数值 for 边界判断。
// 正步长使用 `<=`，负步长使用 `>=`，从而决定循环是否继续。
func numericForContinues(current float64, limit float64, step float64) bool {
	if step > 0 {
		return current <= limit
	}

	return current >= limit
}

func requireNumber(value Value, operator string) (float64, error) {
	if value.Type != ValueTypeNumber {
		return 0, fmt.Errorf("operator %q expects number operand, got %s", operator, value.Type)
	}

	number, ok := value.Data.(float64)
	if !ok {
		return 0, fmt.Errorf("operator %q received invalid number payload %T", operator, value.Data)
	}

	return number, nil
}

func parseNumberLiteral(literal string) (float64, error) {
	if strings.HasPrefix(literal, "0x") || strings.HasPrefix(literal, "0X") {
		number, err := strconv.ParseUint(literal[2:], 16, 64)
		if err != nil {
			return 0, err
		}

		return float64(number), nil
	}

	return strconv.ParseFloat(literal, 64)
}

func isTruthy(value Value) bool {
	switch value.Type {
	case ValueTypeNil:
		return false
	case ValueTypeBoolean:
		booleanValue, _ := value.Data.(bool)
		return booleanValue
	default:
		return true
	}
}

func valuesEqual(left, right Value) bool {
	if left.Type != right.Type {
		return false
	}

	switch left.Type {
	case ValueTypeNil:
		return true
	default:
		return left.Data == right.Data
	}
}

func valueToString(value Value) string {
	switch value.Type {
	case ValueTypeNil:
		return "nil"
	case ValueTypeBoolean:
		if value.Data.(bool) {
			return "true"
		}

		return "false"
	case ValueTypeNumber:
		return strconv.FormatFloat(value.Data.(float64), 'f', -1, 64)
	case ValueTypeString:
		return value.Data.(string)
	default:
		return fmt.Sprintf("%v", value.Data)
	}
}

package vm

import (
	"fmt"
	"math"
	"strconv"

	"github.com/yunfanli-dev/aLuavmWithGo/internal/ir"
)

type executionResult struct {
	returnValues []Value
}

type executor struct {
	scopes []map[string]*valueCell
}

// executeProgram evaluates the current IR subset and returns any explicit return values.
func executeProgram(state *State, program *ir.Program) (*executionResult, error) {
	if program == nil {
		return nil, fmt.Errorf("execute nil IR program")
	}
	if state == nil {
		return nil, fmt.Errorf("execute nil VM state")
	}

	exec := &executor{
		scopes: []map[string]*valueCell{state.globals},
	}

	for _, statement := range program.Statements {
		result, done, err := exec.executeStatement(statement)
		if err != nil {
			return nil, err
		}

		if done {
			return result, nil
		}
	}

	return &executionResult{}, nil
}

func (e *executor) executeStatement(statement ir.Statement) (*executionResult, bool, error) {
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
	case *ir.FunctionDeclarationStatement:
		e.assign(node.Name, e.makeUserFunctionValue(node.Name, node.Parameters, node.Body))

		return nil, false, nil
	case *ir.LocalFunctionDeclarationStatement:
		placeholder := NilValue()
		e.defineLocal(node.Name, placeholder)
		functionValue := e.makeUserFunctionValue(node.Name, node.Parameters, node.Body)
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
			return result, true, nil
		}
	}
}

// executeRepeatStatement runs a repeat-until loop and evaluates the until condition in the loop body's scope.
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

// executeNumericForStatement evaluates the current numeric for-loop subset with start, limit, and step expressions.
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
			return result, true, nil
		}

		current += step
	}

	return nil, false, nil
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
	}

	value, err := e.evaluateExpression(expression)
	if err != nil {
		return nil, err
	}

	return []Value{value}, nil
}

func (e *executor) evaluateExpression(expression ir.Expression) (Value, error) {
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
		number, err := strconv.ParseFloat(node.Literal, 64)
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

		value, exists, err := tableValue.get(index)
		if err != nil {
			return NilValue(), err
		}

		if !exists {
			return NilValue(), nil
		}

		return value, nil
	case *ir.TableConstructorExpression:
		tableValue := newTable()
		for _, field := range node.Fields {
			key, err := e.evaluateExpression(field.Key)
			if err != nil {
				return NilValue(), err
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
		return e.makeUserFunctionValue("", node.Parameters, node.Body), nil
	case *ir.UnaryExpression:
		return e.evaluateUnaryExpression(node)
	case *ir.BinaryExpression:
		return e.evaluateBinaryExpression(node)
	default:
		return NilValue(), fmt.Errorf("evaluate unsupported IR expression %T", expression)
	}
}

func (e *executor) makeUserFunctionValue(name string, parameters []string, body []ir.Statement) Value {
	return Value{
		Type: ValueTypeFunction,
		Data: &userFunction{
			name:       name,
			parameters: append([]string(nil), parameters...),
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

		return tableValue.set(index, value)
	default:
		return fmt.Errorf("unsupported assignment target %T", target)
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
	globalScope := savedScopes[0]
	capturedScope := copyCellMap(functionValue.captured)
	e.scopes = []map[string]*valueCell{globalScope, capturedScope, {}}
	defer func() {
		e.scopes = savedScopes
	}()

	for index, parameter := range functionValue.parameters {
		value := NilValue()
		if index < len(arguments) {
			value = arguments[index]
		}

		e.defineLocal(parameter, value)
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
			return NilValue(), err
		}

		return Value{Type: ValueTypeNumber, Data: -number}, nil
	case "not":
		return Value{Type: ValueTypeBoolean, Data: !isTruthy(operand)}, nil
	case "#":
		if operand.Type != ValueTypeString {
			return NilValue(), fmt.Errorf("operator '#' expects string operand, got %s", operand.Type)
		}

		return Value{Type: ValueTypeNumber, Data: float64(len(operand.Data.(string)))}, nil
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
		return numericBinary(left, right, expression.Operator, func(a, b float64) float64 { return a + b })
	case "-":
		return numericBinary(left, right, expression.Operator, func(a, b float64) float64 { return a - b })
	case "*":
		return numericBinary(left, right, expression.Operator, func(a, b float64) float64 { return a * b })
	case "/":
		return numericBinary(left, right, expression.Operator, func(a, b float64) float64 { return a / b })
	case "%":
		return numericBinary(left, right, expression.Operator, math.Mod)
	case "^":
		return numericBinary(left, right, expression.Operator, math.Pow)
	case "..":
		return Value{Type: ValueTypeString, Data: valueToString(left) + valueToString(right)}, nil
	case "<":
		return comparisonBinary(left, right, expression.Operator, func(a, b float64) bool { return a < b })
	case "<=":
		return comparisonBinary(left, right, expression.Operator, func(a, b float64) bool { return a <= b })
	case ">":
		return comparisonBinary(left, right, expression.Operator, func(a, b float64) bool { return a > b })
	case ">=":
		return comparisonBinary(left, right, expression.Operator, func(a, b float64) bool { return a >= b })
	case "==":
		return Value{Type: ValueTypeBoolean, Data: valuesEqual(left, right)}, nil
	case "~=":
		return Value{Type: ValueTypeBoolean, Data: !valuesEqual(left, right)}, nil
	default:
		return NilValue(), fmt.Errorf("unsupported binary operator %q", expression.Operator)
	}
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

// numericForContinues applies Lua-style numeric for-loop bounds for positive and negative steps.
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

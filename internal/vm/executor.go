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
	locals map[string]Value
}

// executeProgram evaluates the current IR subset and returns any explicit return values.
func executeProgram(program *ir.Program) (*executionResult, error) {
	if program == nil {
		return nil, fmt.Errorf("execute nil IR program")
	}

	exec := &executor{
		locals: make(map[string]Value),
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

			e.locals[name] = value
		}

		return nil, false, nil
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

func (e *executor) evaluateExpressionList(expressions []ir.Expression) ([]Value, error) {
	values := make([]Value, 0, len(expressions))
	for _, expression := range expressions {
		value, err := e.evaluateExpression(expression)
		if err != nil {
			return nil, err
		}

		values = append(values, value)
	}

	return values, nil
}

func (e *executor) evaluateExpression(expression ir.Expression) (Value, error) {
	switch node := expression.(type) {
	case *ir.IdentifierExpression:
		value, ok := e.locals[node.Name]
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
	case *ir.UnaryExpression:
		return e.evaluateUnaryExpression(node)
	case *ir.BinaryExpression:
		return e.evaluateBinaryExpression(node)
	default:
		return NilValue(), fmt.Errorf("evaluate unsupported IR expression %T", expression)
	}
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

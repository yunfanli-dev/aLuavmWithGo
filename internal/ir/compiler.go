package ir

import (
	"fmt"

	"github.com/yunfanli-dev/aLuavmWithGo/internal/parser"
)

// CompileChunk compiles a parsed Lua 5.1 subset chunk into the current IR form.
func CompileChunk(chunk *parser.Chunk) (*Program, error) {
	compiler := &compiler{}
	return compiler.compileChunk(chunk)
}

type compiler struct{}

func (c *compiler) compileChunk(chunk *parser.Chunk) (*Program, error) {
	if chunk == nil {
		return nil, fmt.Errorf("compile nil parser chunk")
	}

	statements := make([]Statement, 0, len(chunk.Statements))
	for _, statement := range chunk.Statements {
		compiled, err := c.compileStatement(statement)
		if err != nil {
			return nil, err
		}

		statements = append(statements, compiled)
	}

	return &Program{Statements: statements}, nil
}

func (c *compiler) compileStatement(statement parser.Statement) (Statement, error) {
	switch node := statement.(type) {
	case *parser.LocalAssignStatement:
		names := make([]string, 0, len(node.Names))
		for _, name := range node.Names {
			names = append(names, name.Name)
		}

		values, err := c.compileExpressions(node.Values)
		if err != nil {
			return nil, err
		}

		return &LocalAssignStatement{
			Names:  names,
			Values: values,
		}, nil
	case *parser.ReturnStatement:
		values, err := c.compileExpressions(node.Values)
		if err != nil {
			return nil, err
		}

		return &ReturnStatement{Values: values}, nil
	default:
		return nil, fmt.Errorf("compile unsupported statement type %T", statement)
	}
}

func (c *compiler) compileExpressions(expressions []parser.Expression) ([]Expression, error) {
	compiled := make([]Expression, 0, len(expressions))
	for _, expression := range expressions {
		value, err := c.compileExpression(expression)
		if err != nil {
			return nil, err
		}

		compiled = append(compiled, value)
	}

	return compiled, nil
}

func (c *compiler) compileExpression(expression parser.Expression) (Expression, error) {
	switch node := expression.(type) {
	case *parser.Identifier:
		return &IdentifierExpression{Name: node.Name}, nil
	case *parser.NilExpression:
		return &NilExpression{}, nil
	case *parser.BooleanExpression:
		return &BooleanExpression{Value: node.Value}, nil
	case *parser.NumberExpression:
		return &NumberExpression{Literal: node.Literal}, nil
	case *parser.StringExpression:
		return &StringExpression{Value: node.Value}, nil
	case *parser.UnaryExpression:
		operand, err := c.compileExpression(node.Operand)
		if err != nil {
			return nil, err
		}

		return &UnaryExpression{
			Operator: string(node.Operator),
			Operand:  operand,
		}, nil
	case *parser.BinaryExpression:
		left, err := c.compileExpression(node.Left)
		if err != nil {
			return nil, err
		}

		right, err := c.compileExpression(node.Right)
		if err != nil {
			return nil, err
		}

		return &BinaryExpression{
			Left:     left,
			Operator: string(node.Operator),
			Right:    right,
		}, nil
	default:
		return nil, fmt.Errorf("compile unsupported expression type %T", expression)
	}
}

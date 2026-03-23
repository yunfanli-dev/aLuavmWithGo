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
	case *parser.CallStatement:
		call, err := c.compileCallExpression(node.Call)
		if err != nil {
			return nil, err
		}

		return &CallStatement{Call: call}, nil
	case *parser.AssignStatement:
		names := make([]string, 0, len(node.Names))
		for _, name := range node.Names {
			names = append(names, name.Name)
		}

		values, err := c.compileExpressions(node.Values)
		if err != nil {
			return nil, err
		}

		return &AssignStatement{
			Names:  names,
			Values: values,
		}, nil
	case *parser.FunctionDeclarationStatement:
		parameters := make([]string, 0, len(node.Parameters))
		for _, parameter := range node.Parameters {
			parameters = append(parameters, parameter.Name)
		}

		body, err := c.compileStatements(node.Body)
		if err != nil {
			return nil, err
		}

		return &FunctionDeclarationStatement{
			Name:       node.Name.Name,
			Parameters: parameters,
			Body:       body,
		}, nil
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
	case *parser.IfStatement:
		clauses := make([]IfClause, 0, len(node.Clauses))
		for _, clause := range node.Clauses {
			condition, err := c.compileExpression(clause.Condition)
			if err != nil {
				return nil, err
			}

			body, err := c.compileStatements(clause.Body)
			if err != nil {
				return nil, err
			}

			clauses = append(clauses, IfClause{
				Condition: condition,
				Body:      body,
			})
		}

		elseBody, err := c.compileStatements(node.ElseBody)
		if err != nil {
			return nil, err
		}

		return &IfStatement{
			Clauses:  clauses,
			ElseBody: elseBody,
		}, nil
	case *parser.WhileStatement:
		condition, err := c.compileExpression(node.Condition)
		if err != nil {
			return nil, err
		}

		body, err := c.compileStatements(node.Body)
		if err != nil {
			return nil, err
		}

		return &WhileStatement{
			Condition: condition,
			Body:      body,
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

func (c *compiler) compileStatements(statements []parser.Statement) ([]Statement, error) {
	compiled := make([]Statement, 0, len(statements))
	for _, statement := range statements {
		value, err := c.compileStatement(statement)
		if err != nil {
			return nil, err
		}

		compiled = append(compiled, value)
	}

	return compiled, nil
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
	case *parser.CallExpression:
		return c.compileCallExpression(node)
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

func (c *compiler) compileCallExpression(expression *parser.CallExpression) (*CallExpression, error) {
	callee, err := c.compileExpression(expression.Callee)
	if err != nil {
		return nil, err
	}

	arguments, err := c.compileExpressions(expression.Arguments)
	if err != nil {
		return nil, err
	}

	return &CallExpression{
		Callee:    callee,
		Arguments: arguments,
	}, nil
}

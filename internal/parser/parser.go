package parser

import (
	"fmt"
	"strconv"

	"github.com/yunfanli-dev/aLuavmWithGo/internal/lexer"
)

// Parser consumes lexer tokens and builds a Lua 5.1 subset AST.
type Parser struct {
	source       string
	tokens       []lexer.Token
	index        int
	varargScopes []bool
}

// ParseString tokenizes and parses a Lua source string into a chunk AST.
func ParseString(sourceName, input string) (*Chunk, error) {
	tokens, err := lexer.Tokenize(sourceName, input)
	if err != nil {
		return nil, err
	}

	return ParseTokens(sourceName, tokens)
}

// ParseTokens parses a pre-tokenized Lua source stream into a chunk AST.
func ParseTokens(sourceName string, tokens []lexer.Token) (*Chunk, error) {
	parser := New(sourceName, tokens)
	return parser.ParseChunk()
}

// New creates a parser for a token stream.
func New(sourceName string, tokens []lexer.Token) *Parser {
	return &Parser{
		source: sourceName,
		tokens: tokens,
		// Top-level chunk is never a vararg function body.
		varargScopes: []bool{false},
	}
}

// ParseChunk parses the root chunk node from the current token stream.
func (p *Parser) ParseChunk() (*Chunk, error) {
	statements, end, err := p.parseBlock(lexer.TokenEOF)
	if err != nil {
		return nil, err
	}

	start := p.current().Start
	if len(statements) > 0 {
		start = statements[0].Span().Start
	}

	return &Chunk{
		Statements: statements,
		span: Span{
			Start: start,
			End:   end,
		},
	}, nil
}

func (p *Parser) parseBlock(terminators ...lexer.TokenType) ([]Statement, lexer.Position, error) {
	statements := make([]Statement, 0)
	for !p.check(terminators...) {
		statement, err := p.parseStatement()
		if err != nil {
			return nil, lexer.Position{}, err
		}

		statements = append(statements, statement)
		p.match(lexer.TokenSemicolon)
	}

	return statements, p.current().End, nil
}

func (p *Parser) parseStatement() (Statement, error) {
	switch p.current().Type {
	case lexer.TokenFunction:
		return p.parseFunctionDeclarationStatement()
	case lexer.TokenLocal:
		if p.peekType(1) == lexer.TokenFunction {
			return p.parseLocalFunctionDeclarationStatement()
		}

		return p.parseLocalAssignStatement()
	case lexer.TokenDo:
		return p.parseDoStatement()
	case lexer.TokenBreak:
		return p.parseBreakStatement()
	case lexer.TokenIdentifier:
		return p.parseIdentifierLedStatement()
	case lexer.TokenIf:
		return p.parseIfStatement()
	case lexer.TokenWhile:
		return p.parseWhileStatement()
	case lexer.TokenRepeat:
		return p.parseRepeatStatement()
	case lexer.TokenFor:
		return p.parseForStatement()
	case lexer.TokenReturn:
		return p.parseReturnStatement()
	default:
		// TODO: Extend statement parsing with remaining Lua 5.1 subset forms like repeatable block statements.
		return nil, p.errorAtCurrent(fmt.Sprintf("unsupported statement starting with %q", p.current().Type))
	}
}

// parseDoStatement parses a block scoped by `do ... end`.
func (p *Parser) parseDoStatement() (Statement, error) {
	startToken, err := p.expect(lexer.TokenDo, "expected 'do'")
	if err != nil {
		return nil, err
	}

	body, _, err := p.parseBlock(lexer.TokenEnd)
	if err != nil {
		return nil, err
	}

	endToken, err := p.expect(lexer.TokenEnd, "expected 'end' after do block")
	if err != nil {
		return nil, err
	}

	return &DoStatement{
		Body: body,
		span: Span{
			Start: startToken.Start,
			End:   endToken.End,
		},
	}, nil
}

// parseBreakStatement parses a `break` control-flow statement.
func (p *Parser) parseBreakStatement() (Statement, error) {
	breakToken, err := p.expect(lexer.TokenBreak, "expected 'break'")
	if err != nil {
		return nil, err
	}

	return &BreakStatement{
		span: tokenSpan(breakToken),
	}, nil
}

func (p *Parser) parseIdentifierLedStatement() (Statement, error) {
	startIndex := p.index
	target, err := p.parseAssignableExpression()
	if err != nil {
		return nil, err
	}

	if p.check(lexer.TokenAssign, lexer.TokenComma) {
		return p.finishAssignStatement(target)
	}

	p.index = startIndex
	expression, err := p.parsePrefixExpression()
	if err != nil {
		return nil, err
	}

	call, ok := expression.(*CallExpression)
	if !ok {
		return nil, p.errorAtCurrent("only assignment and function call statements are supported after identifier")
	}

	return &CallStatement{
		Call: call,
		span: call.Span(),
	}, nil
}

func (p *Parser) finishAssignStatement(firstTarget Expression) (Statement, error) {
	targets := []Expression{firstTarget}
	for p.match(lexer.TokenComma) {
		target, err := p.parseAssignableExpression()
		if err != nil {
			return nil, err
		}

		targets = append(targets, target)
	}

	if _, err := p.expect(lexer.TokenAssign, "expected '=' in assignment"); err != nil {
		return nil, err
	}

	values, err := p.parseExpressionList()
	if err != nil {
		return nil, err
	}

	return &AssignStatement{
		Targets: targets,
		Values:  values,
		span: Span{
			Start: firstTarget.Span().Start,
			End:   values[len(values)-1].Span().End,
		},
	}, nil
}

func (p *Parser) parseFunctionDeclarationStatement() (Statement, error) {
	startToken, err := p.expect(lexer.TokenFunction, "expected 'function'")
	if err != nil {
		return nil, err
	}

	target, methodSelf, err := p.parseFunctionName()
	if err != nil {
		return nil, err
	}

	parameters, isVararg, body, endToken, err := p.parseFunctionBody()
	if err != nil {
		return nil, err
	}

	if methodSelf {
		selfToken := lexer.Token{
			Type:    lexer.TokenIdentifier,
			Literal: "self",
			Start:   startToken.Start,
			End:     startToken.End,
		}
		parameters = append([]Identifier{{Name: "self", span: tokenSpan(selfToken)}}, parameters...)
	}

	if identifier, ok := target.(*Identifier); ok && !methodSelf {
		return &FunctionDeclarationStatement{
			Name:       *identifier,
			Parameters: parameters,
			IsVararg:   isVararg,
			Body:       body,
			span: Span{
				Start: startToken.Start,
				End:   endToken.End,
			},
		}, nil
	}

	functionExpr := &FunctionExpression{
		Parameters: parameters,
		IsVararg:   isVararg,
		Body:       body,
		span: Span{
			Start: startToken.Start,
			End:   endToken.End,
		},
	}

	return &AssignStatement{
		Targets: []Expression{target},
		Values:  []Expression{functionExpr},
		span: Span{
			Start: startToken.Start,
			End:   endToken.End,
		},
	}, nil
}

func (p *Parser) parseLocalFunctionDeclarationStatement() (Statement, error) {
	startToken, err := p.expect(lexer.TokenLocal, "expected 'local'")
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokenFunction, "expected 'function' after 'local'"); err != nil {
		return nil, err
	}

	nameToken, err := p.expect(lexer.TokenIdentifier, "expected local function name")
	if err != nil {
		return nil, err
	}

	parameters, isVararg, body, endToken, err := p.parseFunctionBody()
	if err != nil {
		return nil, err
	}

	return &LocalFunctionDeclarationStatement{
		Name:       Identifier{Name: nameToken.Literal, span: tokenSpan(nameToken)},
		Parameters: parameters,
		IsVararg:   isVararg,
		Body:       body,
		span: Span{
			Start: startToken.Start,
			End:   endToken.End,
		},
	}, nil
}

// parseFunctionName parses a function declaration name and method suffix in Lua function sugar.
func (p *Parser) parseFunctionName() (Expression, bool, error) {
	nameToken, err := p.expect(lexer.TokenIdentifier, "expected function name")
	if err != nil {
		return nil, false, err
	}

	var target Expression = &Identifier{Name: nameToken.Literal, span: tokenSpan(nameToken)}
	methodSelf := false

	for p.match(lexer.TokenDot) {
		fieldToken, err := p.expect(lexer.TokenIdentifier, "expected field name after '.' in function name")
		if err != nil {
			return nil, false, err
		}

		target = &IndexExpression{
			Target: target,
			Index:  &StringExpression{Value: fieldToken.Literal, span: tokenSpan(fieldToken)},
			span: Span{
				Start: target.Span().Start,
				End:   fieldToken.End,
			},
		}
	}

	if p.match(lexer.TokenColon) {
		fieldToken, err := p.expect(lexer.TokenIdentifier, "expected method name after ':' in function name")
		if err != nil {
			return nil, false, err
		}

		methodSelf = true
		target = &IndexExpression{
			Target: target,
			Index:  &StringExpression{Value: fieldToken.Literal, span: tokenSpan(fieldToken)},
			span: Span{
				Start: target.Span().Start,
				End:   fieldToken.End,
			},
		}
	}

	return target, methodSelf, nil
}

func (p *Parser) parseLocalAssignStatement() (Statement, error) {
	startToken, err := p.expect(lexer.TokenLocal, "expected 'local'")
	if err != nil {
		return nil, err
	}

	names := make([]Identifier, 0, 1)
	firstName, err := p.expect(lexer.TokenIdentifier, "expected local variable name")
	if err != nil {
		return nil, err
	}
	names = append(names, Identifier{Name: firstName.Literal, span: tokenSpan(firstName)})

	for p.match(lexer.TokenComma) {
		nameToken, err := p.expect(lexer.TokenIdentifier, "expected local variable name after ','")
		if err != nil {
			return nil, err
		}

		names = append(names, Identifier{Name: nameToken.Literal, span: tokenSpan(nameToken)})
	}

	values := make([]Expression, 0)
	end := firstName.End
	if p.match(lexer.TokenAssign) {
		values, err = p.parseExpressionList()
		if err != nil {
			return nil, err
		}

		end = values[len(values)-1].Span().End
	}

	return &LocalAssignStatement{
		Names:  names,
		Values: values,
		span: Span{
			Start: startToken.Start,
			End:   end,
		},
	}, nil
}

func (p *Parser) parseIfStatement() (Statement, error) {
	startToken, err := p.expect(lexer.TokenIf, "expected 'if'")
	if err != nil {
		return nil, err
	}

	clauses := make([]IfClause, 0, 1)
	elseBody := make([]Statement, 0)

	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokenThen, "expected 'then' after if condition"); err != nil {
		return nil, err
	}

	body, end, err := p.parseBlock(lexer.TokenElseIf, lexer.TokenElse, lexer.TokenEnd)
	if err != nil {
		return nil, err
	}

	clauses = append(clauses, IfClause{
		Condition: condition,
		Body:      body,
		span: Span{
			Start: startToken.Start,
			End:   end,
		},
	})

	for p.match(lexer.TokenElseIf) {
		elseifToken := p.previous()
		elseifCondition, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		if _, err := p.expect(lexer.TokenThen, "expected 'then' after elseif condition"); err != nil {
			return nil, err
		}

		elseifBody, elseifEnd, err := p.parseBlock(lexer.TokenElseIf, lexer.TokenElse, lexer.TokenEnd)
		if err != nil {
			return nil, err
		}

		clauses = append(clauses, IfClause{
			Condition: elseifCondition,
			Body:      elseifBody,
			span: Span{
				Start: elseifToken.Start,
				End:   elseifEnd,
			},
		})
	}

	if p.match(lexer.TokenElse) {
		var elseEnd lexer.Position
		elseBody, elseEnd, err = p.parseBlock(lexer.TokenEnd)
		if err != nil {
			return nil, err
		}
		_ = elseEnd
	}

	endToken, err := p.expect(lexer.TokenEnd, "expected 'end' after if statement")
	if err != nil {
		return nil, err
	}

	return &IfStatement{
		Clauses:  clauses,
		ElseBody: elseBody,
		span: Span{
			Start: startToken.Start,
			End:   endToken.End,
		},
	}, nil
}

func (p *Parser) parseWhileStatement() (Statement, error) {
	startToken, err := p.expect(lexer.TokenWhile, "expected 'while'")
	if err != nil {
		return nil, err
	}

	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokenDo, "expected 'do' after while condition"); err != nil {
		return nil, err
	}

	body, _, err := p.parseBlock(lexer.TokenEnd)
	if err != nil {
		return nil, err
	}

	endToken, err := p.expect(lexer.TokenEnd, "expected 'end' after while loop")
	if err != nil {
		return nil, err
	}

	return &WhileStatement{
		Condition: condition,
		Body:      body,
		span: Span{
			Start: startToken.Start,
			End:   endToken.End,
		},
	}, nil
}

// parseRepeatStatement parses a repeat-until loop and keeps the body visible to the terminating condition.
func (p *Parser) parseRepeatStatement() (Statement, error) {
	startToken, err := p.expect(lexer.TokenRepeat, "expected 'repeat'")
	if err != nil {
		return nil, err
	}

	body, _, err := p.parseBlock(lexer.TokenUntil)
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokenUntil, "expected 'until' after repeat body"); err != nil {
		return nil, err
	}

	condition, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	return &RepeatStatement{
		Body:      body,
		Condition: condition,
		span: Span{
			Start: startToken.Start,
			End:   condition.Span().End,
		},
	}, nil
}

// parseForStatement parses the numeric and generic for-loop forms in the current Lua 5.1 subset.
func (p *Parser) parseForStatement() (Statement, error) {
	startToken, err := p.expect(lexer.TokenFor, "expected 'for'")
	if err != nil {
		return nil, err
	}

	nameToken, err := p.expect(lexer.TokenIdentifier, "expected for-loop variable name")
	if err != nil {
		return nil, err
	}

	if p.match(lexer.TokenAssign) {
		return p.finishNumericForStatement(startToken, nameToken)
	}

	names := []Identifier{{Name: nameToken.Literal, span: tokenSpan(nameToken)}}
	for p.match(lexer.TokenComma) {
		nextName, err := p.expect(lexer.TokenIdentifier, "expected generic for-loop variable name after ','")
		if err != nil {
			return nil, err
		}

		names = append(names, Identifier{Name: nextName.Literal, span: tokenSpan(nextName)})
	}

	if _, err := p.expect(lexer.TokenIn, "expected 'in' after generic for-loop variables"); err != nil {
		return nil, err
	}

	iterators, err := p.parseExpressionList()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokenDo, "expected 'do' after generic for-loop iterators"); err != nil {
		return nil, err
	}

	body, _, err := p.parseBlock(lexer.TokenEnd)
	if err != nil {
		return nil, err
	}

	endToken, err := p.expect(lexer.TokenEnd, "expected 'end' after for loop")
	if err != nil {
		return nil, err
	}

	return &GenericForStatement{
		Names:     names,
		Iterators: iterators,
		Body:      body,
		span: Span{
			Start: startToken.Start,
			End:   endToken.End,
		},
	}, nil
}

// finishNumericForStatement parses the numeric `for name = start, limit[, step] do ... end` form.
func (p *Parser) finishNumericForStatement(startToken lexer.Token, nameToken lexer.Token) (Statement, error) {
	startExpr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokenComma, "expected ',' after for-loop start expression"); err != nil {
		return nil, err
	}

	limitExpr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	var stepExpr Expression
	if p.match(lexer.TokenComma) {
		stepExpr, err = p.parseExpression()
		if err != nil {
			return nil, err
		}
	} else {
		stepExpr = &NumberExpression{Literal: "1", span: limitExpr.Span()}
	}

	if _, err := p.expect(lexer.TokenDo, "expected 'do' after for-loop range"); err != nil {
		return nil, err
	}

	body, _, err := p.parseBlock(lexer.TokenEnd)
	if err != nil {
		return nil, err
	}

	endToken, err := p.expect(lexer.TokenEnd, "expected 'end' after for loop")
	if err != nil {
		return nil, err
	}

	return &NumericForStatement{
		Name:  Identifier{Name: nameToken.Literal, span: tokenSpan(nameToken)},
		Start: startExpr,
		Limit: limitExpr,
		Step:  stepExpr,
		Body:  body,
		span: Span{
			Start: startToken.Start,
			End:   endToken.End,
		},
	}, nil
}

func (p *Parser) parseReturnStatement() (Statement, error) {
	startToken, err := p.expect(lexer.TokenReturn, "expected 'return'")
	if err != nil {
		return nil, err
	}

	if p.check(lexer.TokenEOF, lexer.TokenEnd, lexer.TokenElse, lexer.TokenElseIf, lexer.TokenUntil, lexer.TokenSemicolon) {
		return &ReturnStatement{
			Values: nil,
			span: Span{
				Start: startToken.Start,
				End:   startToken.End,
			},
		}, nil
	}

	values, err := p.parseExpressionList()
	if err != nil {
		return nil, err
	}

	return &ReturnStatement{
		Values: values,
		span: Span{
			Start: startToken.Start,
			End:   values[len(values)-1].Span().End,
		},
	}, nil
}

func (p *Parser) parseExpressionList() ([]Expression, error) {
	expressions := make([]Expression, 0, 1)
	firstExpr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	expressions = append(expressions, firstExpr)

	for p.match(lexer.TokenComma) {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		expressions = append(expressions, expr)
	}

	return expressions, nil
}

func (p *Parser) parseExpression() (Expression, error) {
	return p.parseBinaryExpression(1)
}

func (p *Parser) parseBinaryExpression(minPrecedence int) (Expression, error) {
	left, err := p.parseUnaryExpression()
	if err != nil {
		return nil, err
	}

	for {
		operator := p.current()
		precedence, rightAssociative := binaryPrecedence(operator.Type)
		if precedence < minPrecedence {
			return left, nil
		}

		p.advance()
		nextMin := precedence + 1
		if rightAssociative {
			nextMin = precedence
		}

		right, err := p.parseBinaryExpression(nextMin)
		if err != nil {
			return nil, err
		}

		left = &BinaryExpression{
			Left:     left,
			Operator: operator.Type,
			Right:    right,
			span: Span{
				Start: left.Span().Start,
				End:   right.Span().End,
			},
		}
	}
}

func (p *Parser) parseUnaryExpression() (Expression, error) {
	switch p.current().Type {
	case lexer.TokenMinus, lexer.TokenNot, lexer.TokenHash:
		operator := p.current()
		p.advance()

		operand, err := p.parseUnaryExpression()
		if err != nil {
			return nil, err
		}

		return &UnaryExpression{
			Operator: operator.Type,
			Operand:  operand,
			span: Span{
				Start: operator.Start,
				End:   operand.Span().End,
			},
		}, nil
	default:
		return p.parsePrimaryExpression()
	}
}

func (p *Parser) parsePrimaryExpression() (Expression, error) {
	return p.parsePrefixExpression()
}

func (p *Parser) parsePrefixExpression() (Expression, error) {
	return p.parseSuffixedExpression(true)
}

func (p *Parser) parseAssignableExpression() (Expression, error) {
	return p.parseSuffixedExpression(false)
}

func (p *Parser) parseSuffixedExpression(allowCalls bool) (Expression, error) {
	token := p.current()
	var expression Expression

	switch token.Type {
	case lexer.TokenNil:
		p.advance()
		expression = &NilExpression{span: tokenSpan(token)}
	case lexer.TokenTrue:
		p.advance()
		expression = &BooleanExpression{Value: true, span: tokenSpan(token)}
	case lexer.TokenFalse:
		p.advance()
		expression = &BooleanExpression{Value: false, span: tokenSpan(token)}
	case lexer.TokenNumber:
		p.advance()
		expression = &NumberExpression{Literal: token.Literal, span: tokenSpan(token)}
	case lexer.TokenString:
		p.advance()
		expression = &StringExpression{Value: token.Literal, span: tokenSpan(token)}
	case lexer.TokenVararg:
		if !p.currentVarargScope() {
			return nil, p.errorAtCurrent("vararg expression is only allowed inside a vararg function")
		}

		p.advance()
		expression = &VarargExpression{span: tokenSpan(token)}
	case lexer.TokenIdentifier:
		p.advance()
		expression = &Identifier{Name: token.Literal, span: tokenSpan(token)}
	case lexer.TokenFunction:
		functionExpr, err := p.parseFunctionExpression()
		if err != nil {
			return nil, err
		}

		expression = functionExpr
	case lexer.TokenLeftBrace:
		return p.parseTableConstructorExpression()
	case lexer.TokenLeftParen:
		startToken := p.advance()
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		endToken, err := p.expect(lexer.TokenRightParen, "expected ')' after expression")
		if err != nil {
			return nil, err
		}

		expression = &ParenthesizedExpression{
			Inner: expr,
			span: Span{
				Start: startToken.Start,
				End:   endToken.End,
			},
		}
	default:
		// TODO: Extend primary parsing with Lua 5.1 subset forms like function expressions and table constructors.
		return nil, p.errorAtCurrent(fmt.Sprintf("unexpected token %q in expression", token.Type))
	}

	for {
		switch {
		case allowCalls && p.check(lexer.TokenLeftParen):
			call, err := p.finishCallExpression(expression)
			if err != nil {
				return nil, err
			}

			expression = call
		case allowCalls && p.check(lexer.TokenLeftBrace):
			call, err := p.finishTableCallExpression(expression)
			if err != nil {
				return nil, err
			}

			expression = call
		case allowCalls && p.check(lexer.TokenString):
			call, err := p.finishStringCallExpression(expression)
			if err != nil {
				return nil, err
			}

			expression = call
		case allowCalls && p.match(lexer.TokenColon):
			call, err := p.finishMethodCallExpression(expression)
			if err != nil {
				return nil, err
			}

			expression = call
		case p.match(lexer.TokenLeftBracket):
			index, err := p.parseExpression()
			if err != nil {
				return nil, err
			}

			endToken, err := p.expect(lexer.TokenRightBracket, "expected ']' after index expression")
			if err != nil {
				return nil, err
			}

			expression = &IndexExpression{
				Target: expression,
				Index:  index,
				span: Span{
					Start: expression.Span().Start,
					End:   endToken.End,
				},
			}
		case p.match(lexer.TokenDot):
			nameToken, err := p.expect(lexer.TokenIdentifier, "expected field name after '.'")
			if err != nil {
				return nil, err
			}

			expression = &IndexExpression{
				Target: expression,
				Index:  &StringExpression{Value: nameToken.Literal, span: tokenSpan(nameToken)},
				span: Span{
					Start: expression.Span().Start,
					End:   nameToken.End,
				},
			}
		default:
			return expression, nil
		}
	}
}

func (p *Parser) parseFunctionExpression() (*FunctionExpression, error) {
	startToken, err := p.expect(lexer.TokenFunction, "expected 'function'")
	if err != nil {
		return nil, err
	}

	parameters, isVararg, body, endToken, err := p.parseFunctionBody()
	if err != nil {
		return nil, err
	}

	return &FunctionExpression{
		Parameters: parameters,
		IsVararg:   isVararg,
		Body:       body,
		span: Span{
			Start: startToken.Start,
			End:   endToken.End,
		},
	}, nil
}

func (p *Parser) parseFunctionBody() ([]Identifier, bool, []Statement, lexer.Token, error) {
	if _, err := p.expect(lexer.TokenLeftParen, "expected '(' after function name"); err != nil {
		return nil, false, nil, lexer.Token{}, err
	}

	parameters := make([]Identifier, 0)
	isVararg := false
	if !p.check(lexer.TokenRightParen) {
		for {
			if p.check(lexer.TokenVararg) {
				p.advance()
				isVararg = true
				break
			}

			parameterToken, err := p.expect(lexer.TokenIdentifier, "expected parameter name")
			if err != nil {
				return nil, false, nil, lexer.Token{}, err
			}

			parameters = append(parameters, Identifier{Name: parameterToken.Literal, span: tokenSpan(parameterToken)})
			if !p.match(lexer.TokenComma) {
				break
			}
		}
	}

	if _, err := p.expect(lexer.TokenRightParen, "expected ')' after parameter list"); err != nil {
		return nil, false, nil, lexer.Token{}, err
	}

	p.pushVarargScope(isVararg)
	defer p.popVarargScope()

	body, _, err := p.parseBlock(lexer.TokenEnd)
	if err != nil {
		return nil, false, nil, lexer.Token{}, err
	}

	endToken, err := p.expect(lexer.TokenEnd, "expected 'end' after function declaration")
	if err != nil {
		return nil, false, nil, lexer.Token{}, err
	}

	return parameters, isVararg, body, endToken, nil
}

func (p *Parser) pushVarargScope(isVararg bool) {
	p.varargScopes = append(p.varargScopes, isVararg)
}

func (p *Parser) popVarargScope() {
	if len(p.varargScopes) <= 1 {
		return
	}

	p.varargScopes = p.varargScopes[:len(p.varargScopes)-1]
}

func (p *Parser) currentVarargScope() bool {
	if len(p.varargScopes) == 0 {
		return false
	}

	return p.varargScopes[len(p.varargScopes)-1]
}

func (p *Parser) current() lexer.Token {
	if len(p.tokens) == 0 {
		return lexer.Token{Type: lexer.TokenEOF}
	}

	if p.index >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}

	return p.tokens[p.index]
}

func (p *Parser) advance() lexer.Token {
	token := p.current()
	if p.index < len(p.tokens)-1 {
		p.index++
	}

	return token
}

func (p *Parser) previous() lexer.Token {
	if p.index == 0 {
		return p.current()
	}

	return p.tokens[p.index-1]
}

func (p *Parser) finishCallExpression(callee Expression) (*CallExpression, error) {
	if _, err := p.expect(lexer.TokenLeftParen, "expected '(' after callee"); err != nil {
		return nil, err
	}

	arguments := make([]Expression, 0)
	if !p.check(lexer.TokenRightParen) {
		values, err := p.parseExpressionList()
		if err != nil {
			return nil, err
		}

		arguments = values
	}

	endToken, err := p.expect(lexer.TokenRightParen, "expected ')' after argument list")
	if err != nil {
		return nil, err
	}

	return p.newCallExpression(callee, nil, "", arguments, endToken.End), nil
}

// finishTableCallExpression parses the Lua 5.1 `callee{...}` call sugar as a single table argument.
func (p *Parser) finishTableCallExpression(callee Expression) (*CallExpression, error) {
	argument, err := p.parseTableConstructorExpression()
	if err != nil {
		return nil, err
	}

	return p.newCallExpression(callee, nil, "", []Expression{argument}, argument.Span().End), nil
}

// finishStringCallExpression parses the Lua 5.1 `callee"literal"` call sugar as a single string argument.
func (p *Parser) finishStringCallExpression(callee Expression) (*CallExpression, error) {
	token, err := p.expect(lexer.TokenString, "expected string literal after callee")
	if err != nil {
		return nil, err
	}

	argument := &StringExpression{Value: token.Literal, span: tokenSpan(token)}
	return p.newCallExpression(callee, nil, "", []Expression{argument}, token.End), nil
}

// newCallExpression normalizes the parser's call sugar into the shared call AST shape.
func (p *Parser) newCallExpression(callee Expression, receiver Expression, method string, arguments []Expression, end lexer.Position) *CallExpression {
	return &CallExpression{
		Callee:    callee,
		Receiver:  receiver,
		Method:    method,
		Arguments: arguments,
		span: Span{
			Start: callStart(callee, receiver),
			End:   end,
		},
	}
}

// finishMethodCallExpression parses `receiver:name(args)` and keeps receiver evaluation single-shot.
func (p *Parser) finishMethodCallExpression(receiver Expression) (*CallExpression, error) {
	methodToken, err := p.expect(lexer.TokenIdentifier, "expected method name after ':'")
	if err != nil {
		return nil, err
	}

	switch p.current().Type {
	case lexer.TokenLeftParen:
		if _, err := p.expect(lexer.TokenLeftParen, "expected '(' after method name"); err != nil {
			return nil, err
		}

		arguments := make([]Expression, 0)
		if !p.check(lexer.TokenRightParen) {
			values, err := p.parseExpressionList()
			if err != nil {
				return nil, err
			}

			arguments = values
		}

		endToken, err := p.expect(lexer.TokenRightParen, "expected ')' after argument list")
		if err != nil {
			return nil, err
		}

		return p.newCallExpression(nil, receiver, methodToken.Literal, arguments, endToken.End), nil
	case lexer.TokenLeftBrace:
		argument, err := p.parseTableConstructorExpression()
		if err != nil {
			return nil, err
		}

		return p.newCallExpression(nil, receiver, methodToken.Literal, []Expression{argument}, argument.Span().End), nil
	case lexer.TokenString:
		argumentToken, err := p.expect(lexer.TokenString, "expected string literal after method name")
		if err != nil {
			return nil, err
		}

		argument := &StringExpression{Value: argumentToken.Literal, span: tokenSpan(argumentToken)}
		return p.newCallExpression(nil, receiver, methodToken.Literal, []Expression{argument}, argumentToken.End), nil
	default:
		return nil, p.errorAtCurrent("expected call arguments after method name")
	}
}

func callStart(callee Expression, receiver Expression) lexer.Position {
	if receiver != nil {
		return receiver.Span().Start
	}

	return callee.Span().Start
}

func (p *Parser) parseTableConstructorExpression() (Expression, error) {
	startToken, err := p.expect(lexer.TokenLeftBrace, "expected '{'")
	if err != nil {
		return nil, err
	}

	fields := make([]TableField, 0)
	arrayIndex := 1
	for !p.check(lexer.TokenRightBrace) {
		field, err := p.parseTableField(arrayIndex)
		if err != nil {
			return nil, err
		}

		fields = append(fields, field)
		if field.Key != nil {
			arrayIndex++
		}

		if !(p.match(lexer.TokenComma) || p.match(lexer.TokenSemicolon)) {
			break
		}
	}

	endToken, err := p.expect(lexer.TokenRightBrace, "expected '}' after table constructor")
	if err != nil {
		return nil, err
	}

	return &TableConstructorExpression{
		Fields: fields,
		span: Span{
			Start: startToken.Start,
			End:   endToken.End,
		},
	}, nil
}

func (p *Parser) parseTableField(arrayIndex int) (TableField, error) {
	if p.match(lexer.TokenLeftBracket) {
		key, err := p.parseExpression()
		if err != nil {
			return TableField{}, err
		}

		if _, err := p.expect(lexer.TokenRightBracket, "expected ']' after table key"); err != nil {
			return TableField{}, err
		}

		if _, err := p.expect(lexer.TokenAssign, "expected '=' after table key"); err != nil {
			return TableField{}, err
		}

		value, err := p.parseExpression()
		if err != nil {
			return TableField{}, err
		}

		return TableField{
			Key:   key,
			Value: value,
			span: Span{
				Start: key.Span().Start,
				End:   value.Span().End,
			},
		}, nil
	}

	if p.check(lexer.TokenIdentifier) && p.peekType(1) == lexer.TokenAssign {
		nameToken, _ := p.expect(lexer.TokenIdentifier, "expected field name")
		if _, err := p.expect(lexer.TokenAssign, "expected '=' after field name"); err != nil {
			return TableField{}, err
		}

		value, err := p.parseExpression()
		if err != nil {
			return TableField{}, err
		}

		return TableField{
			Key:   &StringExpression{Value: nameToken.Literal, span: tokenSpan(nameToken)},
			Value: value,
			span: Span{
				Start: nameToken.Start,
				End:   value.Span().End,
			},
		}, nil
	}

	value, err := p.parseExpression()
	if err != nil {
		return TableField{}, err
	}

	numberKey := &NumberExpression{
		Literal: strconv.Itoa(arrayIndex),
		span:    value.Span(),
	}

	return TableField{
		Key:         numberKey,
		Value:       value,
		IsListField: true,
		span: Span{
			Start: value.Span().Start,
			End:   value.Span().End,
		},
	}, nil
}

func (p *Parser) peekType(offset int) lexer.TokenType {
	index := p.index + offset
	if index >= len(p.tokens) {
		return lexer.TokenEOF
	}

	return p.tokens[index].Type
}

func (p *Parser) check(types ...lexer.TokenType) bool {
	currentType := p.current().Type
	for _, tokenType := range types {
		if currentType == tokenType {
			return true
		}
	}

	return false
}

func (p *Parser) match(types ...lexer.TokenType) bool {
	if !p.check(types...) {
		return false
	}

	p.advance()
	return true
}

func (p *Parser) expect(tokenType lexer.TokenType, msg string) (lexer.Token, error) {
	if !p.check(tokenType) {
		return lexer.Token{}, p.errorAtCurrent(msg)
	}

	return p.advance(), nil
}

func (p *Parser) errorAtCurrent(msg string) error {
	return &Error{
		Source: p.source,
		Token:  p.current(),
		Msg:    msg,
	}
}

func tokenSpan(token lexer.Token) Span {
	return Span{
		Start: token.Start,
		End:   token.End,
	}
}

func binaryPrecedence(tokenType lexer.TokenType) (int, bool) {
	switch tokenType {
	case lexer.TokenOr:
		return 1, false
	case lexer.TokenAnd:
		return 2, false
	case lexer.TokenLess, lexer.TokenLessEqual, lexer.TokenGreater, lexer.TokenGreaterEqual, lexer.TokenEqual, lexer.TokenNotEqual:
		return 3, false
	case lexer.TokenConcat:
		return 4, true
	case lexer.TokenPlus, lexer.TokenMinus:
		return 5, false
	case lexer.TokenStar, lexer.TokenSlash, lexer.TokenPercent:
		return 6, false
	case lexer.TokenCaret:
		return 7, true
	default:
		return 0, false
	}
}

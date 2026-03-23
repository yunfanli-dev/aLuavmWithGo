package parser

import (
	"fmt"

	"github.com/yunfanli-dev/aLuavmWithGo/internal/lexer"
)

// Parser consumes lexer tokens and builds a Lua 5.1 subset AST.
type Parser struct {
	source string
	tokens []lexer.Token
	index  int
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
	case lexer.TokenIdentifier:
		return p.parseIdentifierLedStatement()
	case lexer.TokenLocal:
		return p.parseLocalAssignStatement()
	case lexer.TokenIf:
		return p.parseIfStatement()
	case lexer.TokenWhile:
		return p.parseWhileStatement()
	case lexer.TokenReturn:
		return p.parseReturnStatement()
	default:
		// TODO: Extend statement parsing with Lua 5.1 subset forms like function declarations and repeat loops.
		return nil, p.errorAtCurrent(fmt.Sprintf("unsupported statement starting with %q", p.current().Type))
	}
}

func (p *Parser) parseIdentifierLedStatement() (Statement, error) {
	if p.isAssignmentStatement() {
		return p.parseAssignStatement()
	}

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

func (p *Parser) parseAssignStatement() (Statement, error) {
	firstName, err := p.expect(lexer.TokenIdentifier, "expected assignment target")
	if err != nil {
		return nil, err
	}

	names := []Identifier{{Name: firstName.Literal, span: tokenSpan(firstName)}}
	for p.match(lexer.TokenComma) {
		nameToken, err := p.expect(lexer.TokenIdentifier, "expected assignment target after ','")
		if err != nil {
			return nil, err
		}

		names = append(names, Identifier{Name: nameToken.Literal, span: tokenSpan(nameToken)})
	}

	if _, err := p.expect(lexer.TokenAssign, "expected '=' in assignment"); err != nil {
		return nil, err
	}

	values, err := p.parseExpressionList()
	if err != nil {
		return nil, err
	}

	return &AssignStatement{
		Names:  names,
		Values: values,
		span: Span{
			Start: firstName.Start,
			End:   values[len(values)-1].Span().End,
		},
	}, nil
}

func (p *Parser) parseFunctionDeclarationStatement() (Statement, error) {
	startToken, err := p.expect(lexer.TokenFunction, "expected 'function'")
	if err != nil {
		return nil, err
	}

	nameToken, err := p.expect(lexer.TokenIdentifier, "expected function name")
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(lexer.TokenLeftParen, "expected '(' after function name"); err != nil {
		return nil, err
	}

	parameters := make([]Identifier, 0)
	if !p.check(lexer.TokenRightParen) {
		for {
			if p.check(lexer.TokenVararg) {
				return nil, p.errorAtCurrent("vararg parameters are not implemented yet")
			}

			parameterToken, err := p.expect(lexer.TokenIdentifier, "expected parameter name")
			if err != nil {
				return nil, err
			}

			parameters = append(parameters, Identifier{Name: parameterToken.Literal, span: tokenSpan(parameterToken)})
			if !p.match(lexer.TokenComma) {
				break
			}
		}
	}

	if _, err := p.expect(lexer.TokenRightParen, "expected ')' after parameter list"); err != nil {
		return nil, err
	}

	body, _, err := p.parseBlock(lexer.TokenEnd)
	if err != nil {
		return nil, err
	}

	endToken, err := p.expect(lexer.TokenEnd, "expected 'end' after function declaration")
	if err != nil {
		return nil, err
	}

	return &FunctionDeclarationStatement{
		Name:       Identifier{Name: nameToken.Literal, span: tokenSpan(nameToken)},
		Parameters: parameters,
		Body:       body,
		span: Span{
			Start: startToken.Start,
			End:   endToken.End,
		},
	}, nil
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
	case lexer.TokenIdentifier:
		p.advance()
		expression = &Identifier{Name: token.Literal, span: tokenSpan(token)}
	case lexer.TokenLeftParen:
		p.advance()
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		if _, err := p.expect(lexer.TokenRightParen, "expected ')' after expression"); err != nil {
			return nil, err
		}

		expression = expr
	default:
		// TODO: Extend primary parsing with Lua 5.1 subset forms like function expressions and table constructors.
		return nil, p.errorAtCurrent(fmt.Sprintf("unexpected token %q in expression", token.Type))
	}

	for p.check(lexer.TokenLeftParen) {
		call, err := p.finishCallExpression(expression)
		if err != nil {
			return nil, err
		}

		expression = call
	}

	return expression, nil
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

func (p *Parser) isAssignmentStatement() bool {
	index := p.index
	if index >= len(p.tokens) || p.tokens[index].Type != lexer.TokenIdentifier {
		return false
	}

	index++
	for index < len(p.tokens) && p.tokens[index].Type == lexer.TokenComma {
		index++
		if index >= len(p.tokens) || p.tokens[index].Type != lexer.TokenIdentifier {
			return false
		}
		index++
	}

	return index < len(p.tokens) && p.tokens[index].Type == lexer.TokenAssign
}

func (p *Parser) finishCallExpression(callee Expression) (*CallExpression, error) {
	start := callee.Span().Start
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

	return &CallExpression{
		Callee:    callee,
		Arguments: arguments,
		span: Span{
			Start: start,
			End:   endToken.End,
		},
	}, nil
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

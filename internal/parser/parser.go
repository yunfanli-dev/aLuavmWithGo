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
	statements := make([]Statement, 0)
	start := p.current().Start

	for !p.check(lexer.TokenEOF) {
		statement, err := p.parseStatement()
		if err != nil {
			return nil, err
		}

		statements = append(statements, statement)
		p.match(lexer.TokenSemicolon)
	}

	end := p.current().End
	return &Chunk{
		Statements: statements,
		span: Span{
			Start: start,
			End:   end,
		},
	}, nil
}

func (p *Parser) parseStatement() (Statement, error) {
	switch p.current().Type {
	case lexer.TokenLocal:
		return p.parseLocalAssignStatement()
	case lexer.TokenReturn:
		return p.parseReturnStatement()
	default:
		// TODO: Extend statement parsing with Lua 5.1 subset forms like assignment, if, while, and function declarations.
		return nil, p.errorAtCurrent(fmt.Sprintf("unsupported statement starting with %q", p.current().Type))
	}
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
	token := p.current()
	switch token.Type {
	case lexer.TokenNil:
		p.advance()
		return &NilExpression{span: tokenSpan(token)}, nil
	case lexer.TokenTrue:
		p.advance()
		return &BooleanExpression{Value: true, span: tokenSpan(token)}, nil
	case lexer.TokenFalse:
		p.advance()
		return &BooleanExpression{Value: false, span: tokenSpan(token)}, nil
	case lexer.TokenNumber:
		p.advance()
		return &NumberExpression{Literal: token.Literal, span: tokenSpan(token)}, nil
	case lexer.TokenString:
		p.advance()
		return &StringExpression{Value: token.Literal, span: tokenSpan(token)}, nil
	case lexer.TokenIdentifier:
		p.advance()
		return &Identifier{Name: token.Literal, span: tokenSpan(token)}, nil
	case lexer.TokenLeftParen:
		p.advance()
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		if _, err := p.expect(lexer.TokenRightParen, "expected ')' after expression"); err != nil {
			return nil, err
		}

		return expr, nil
	default:
		// TODO: Extend primary parsing with Lua 5.1 subset forms like function expressions and table constructors.
		return nil, p.errorAtCurrent(fmt.Sprintf("unexpected token %q in expression", token.Type))
	}
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

package lexer

import "testing"

func TestTokenizeLua51SubsetScript(t *testing.T) {
	source := "local x = 12.5\nif x >= 10 then return \"ok\" end\n"

	tokens, err := Tokenize("sample.lua", source)
	if err != nil {
		t.Fatalf("tokenize source: %v", err)
	}

	expected := []TokenType{
		TokenLocal,
		TokenIdentifier,
		TokenAssign,
		TokenNumber,
		TokenIf,
		TokenIdentifier,
		TokenGreaterEqual,
		TokenNumber,
		TokenThen,
		TokenReturn,
		TokenString,
		TokenEnd,
		TokenEOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, tokenType := range expected {
		if tokens[i].Type != tokenType {
			t.Fatalf("token %d: expected %s, got %s (%q)", i, tokenType, tokens[i].Type, tokens[i].Literal)
		}
	}

	if tokens[3].Literal != "12.5" {
		t.Fatalf("expected decimal number literal, got %q", tokens[3].Literal)
	}

	if tokens[10].Literal != "ok" {
		t.Fatalf("expected unescaped string literal, got %q", tokens[10].Literal)
	}
}

func TestTokenizeSkipsShortComments(t *testing.T) {
	source := "-- comment\nlocal value = foo .. bar\n"

	tokens, err := Tokenize("comments.lua", source)
	if err != nil {
		t.Fatalf("tokenize source: %v", err)
	}

	expected := []TokenType{
		TokenLocal,
		TokenIdentifier,
		TokenAssign,
		TokenIdentifier,
		TokenConcat,
		TokenIdentifier,
		TokenEOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, tokenType := range expected {
		if tokens[i].Type != tokenType {
			t.Fatalf("token %d: expected %s, got %s", i, tokenType, tokens[i].Type)
		}
	}
}

func TestTokenizeSupportsExponentNumbers(t *testing.T) {
	source := "local a = 1e3\nlocal b = 2.5E-2\n"

	tokens, err := Tokenize("exponent.lua", source)
	if err != nil {
		t.Fatalf("tokenize source: %v", err)
	}

	numberLiterals := make([]string, 0, 2)
	for _, token := range tokens {
		if token.Type == TokenNumber {
			numberLiterals = append(numberLiterals, token.Literal)
		}
	}

	if len(numberLiterals) != 2 {
		t.Fatalf("expected 2 number literals, got %#v", numberLiterals)
	}

	if numberLiterals[0] != "1e3" || numberLiterals[1] != "2.5E-2" {
		t.Fatalf("unexpected exponent literals: %#v", numberLiterals)
	}
}

func TestTokenizeRejectsMalformedExponentNumber(t *testing.T) {
	_, err := Tokenize("bad_exponent.lua", "local value = 1e+\n")
	if err == nil {
		t.Fatal("expected malformed exponent error")
	}
}

func TestTokenizeSupportsHexNumbers(t *testing.T) {
	source := "local a = 0xff\nlocal b = 0X10\n"

	tokens, err := Tokenize("hex.lua", source)
	if err != nil {
		t.Fatalf("tokenize source: %v", err)
	}

	numberLiterals := make([]string, 0, 2)
	for _, token := range tokens {
		if token.Type == TokenNumber {
			numberLiterals = append(numberLiterals, token.Literal)
		}
	}

	if len(numberLiterals) != 2 {
		t.Fatalf("expected 2 number literals, got %#v", numberLiterals)
	}

	if numberLiterals[0] != "0xff" || numberLiterals[1] != "0X10" {
		t.Fatalf("unexpected hex literals: %#v", numberLiterals)
	}
}

func TestTokenizeSupportsLongString(t *testing.T) {
	tokens, err := Tokenize("long_string.lua", "return [[hello\nworld]]\n")
	if err != nil {
		t.Fatalf("tokenize source: %v", err)
	}

	if tokens[1].Type != TokenString || tokens[1].Literal != "hello\nworld" {
		t.Fatalf("expected long string literal, got %#v", tokens[1])
	}
}

func TestTokenizeSupportsLeveledLongString(t *testing.T) {
	tokens, err := Tokenize("leveled_long_string.lua", "return [==[hello [world] ]=] test]==]\n")
	if err != nil {
		t.Fatalf("tokenize source: %v", err)
	}

	if tokens[1].Type != TokenString || tokens[1].Literal != "hello [world] ]=] test" {
		t.Fatalf("expected leveled long string literal, got %#v", tokens[1])
	}
}

func TestTokenizeSupportsLongComment(t *testing.T) {
	source := "--[[skip\nthis]]\nlocal value = 1\n"

	tokens, err := Tokenize("long_comment.lua", source)
	if err != nil {
		t.Fatalf("tokenize source: %v", err)
	}

	expected := []TokenType{
		TokenLocal,
		TokenIdentifier,
		TokenAssign,
		TokenNumber,
		TokenEOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for index, tokenType := range expected {
		if tokens[index].Type != tokenType {
			t.Fatalf("token %d: expected %s, got %s", index, tokenType, tokens[index].Type)
		}
	}
}

func TestTokenizeSupportsLeveledLongComment(t *testing.T) {
	source := "--[==[skip ]=] this]==]\nlocal value = 1\n"

	tokens, err := Tokenize("leveled_long_comment.lua", source)
	if err != nil {
		t.Fatalf("tokenize source: %v", err)
	}

	expected := []TokenType{
		TokenLocal,
		TokenIdentifier,
		TokenAssign,
		TokenNumber,
		TokenEOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for index, tokenType := range expected {
		if tokens[index].Type != tokenType {
			t.Fatalf("token %d: expected %s, got %s", index, tokenType, tokens[index].Type)
		}
	}
}

func TestTokenizeRejectsMalformedHexNumber(t *testing.T) {
	_, err := Tokenize("bad_hex.lua", "local value = 0x\n")
	if err == nil {
		t.Fatal("expected malformed hexadecimal error")
	}
}

func TestTokenizeReturnsLocationForUnexpectedCharacter(t *testing.T) {
	_, err := Tokenize("bad.lua", "@")
	if err == nil {
		t.Fatal("expected tokenize error")
	}

	lexErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected lexer error, got %T", err)
	}

	if lexErr.Pos.Line != 1 || lexErr.Pos.Column != 1 {
		t.Fatalf("expected error at 1:1, got %d:%d", lexErr.Pos.Line, lexErr.Pos.Column)
	}
}

func TestTokenizeRejectsUnterminatedLongString(t *testing.T) {
	_, err := Tokenize("unterminated_long_string.lua", "return [[todo")
	if err == nil {
		t.Fatal("expected unterminated long string error")
	}
}

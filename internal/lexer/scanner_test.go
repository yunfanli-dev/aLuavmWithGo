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

func TestTokenizeRejectsLongCommentsForNow(t *testing.T) {
	_, err := Tokenize("long_comment.lua", "--[[todo]]")
	if err == nil {
		t.Fatal("expected long comment error")
	}
}

package parser

import (
	"fmt"

	"github.com/yunfanli-dev/aLuavmWithGo/internal/lexer"
)

// Error describes a parser failure with source token context.
type Error struct {
	Source string
	Token  lexer.Token
	Msg    string
}

// Error formats the parser failure for callers and tests.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}

	source := e.Source
	if source == "" {
		source = "<memory>"
	}

	return fmt.Sprintf("%s:%d:%d: %s", source, e.Token.Start.Line, e.Token.Start.Column, e.Msg)
}

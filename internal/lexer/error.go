package lexer

import "fmt"

// Error describes a lexical failure with source location information.
type Error struct {
	Source string
	Pos    Position
	Msg    string
}

// Error formats the lexical failure for callers and tests.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}

	source := e.Source
	if source == "" {
		source = "<memory>"
	}

	return fmt.Sprintf("%s:%d:%d: %s", source, e.Pos.Line, e.Pos.Column, e.Msg)
}

package vm

import (
	"fmt"

	"github.com/yunfanli-dev/aLuavmWithGo/internal/ir"
	"github.com/yunfanli-dev/aLuavmWithGo/internal/parser"
)

// FrontendResult carries the current non-executing compilation pipeline output.
type FrontendResult struct {
	Program *ir.Program
}

// compileSource runs the current lexer -> parser -> IR frontend pipeline.
func compileSource(source Source) (*FrontendResult, error) {
	chunk, err := parser.ParseString(source.Name, source.Content)
	if err != nil {
		return nil, err
	}

	program, err := ir.CompileChunk(chunk)
	if err != nil {
		return nil, fmt.Errorf("compile source %q to IR: %w", source.Name, err)
	}

	return &FrontendResult{
		Program: program,
	}, nil
}

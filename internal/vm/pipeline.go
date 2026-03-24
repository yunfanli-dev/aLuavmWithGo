package vm

import (
	"fmt"

	"github.com/yunfanli-dev/aLuavmWithGo/internal/ir"
	"github.com/yunfanli-dev/aLuavmWithGo/internal/parser"
)

// FrontendResult 表示前端编译链路产出的结果。
// 当前只包含 IR Program，但后续也可以在这里挂更多中间调试信息。
type FrontendResult struct {
	Program *ir.Program
}

// compileSource 依次执行 lexer、parser 和 IR 编译这三步前端流程。
// 该函数只负责把源码转成可执行的 Program，本身不触发运行。
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

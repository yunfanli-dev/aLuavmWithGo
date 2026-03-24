package parser

import (
	"fmt"

	"github.com/yunfanli-dev/aLuavmWithGo/internal/lexer"
)

// Error 描述解析阶段的一次失败。
// 它携带当前源码名、出错 token 和具体错误消息，便于直接生成可读报错。
type Error struct {
	Source string
	Token  lexer.Token
	Msg    string
}

// Error 按统一格式输出解析错误文本。
// 这样调用方和测试都能稳定拿到“文件:行:列:消息”的结构。
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

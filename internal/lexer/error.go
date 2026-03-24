package lexer

import "fmt"

// Error 描述词法扫描阶段的一次失败。
// 它会携带源码名、位置信息和具体错误消息，便于上层直接展示给用户。
type Error struct {
	Source string
	Pos    Position
	Msg    string
}

// Error 按统一格式输出词法错误文本。
// 这样调用方和测试可以稳定地拿到“文件:行:列:消息”这种结构。
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

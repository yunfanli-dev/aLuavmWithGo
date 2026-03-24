package api

import (
	"fmt"
	"os"
	"path/filepath"
)

// Source 表示进入前端编译链路之前的一份 Lua 源码载荷。
// Name 用于报错和调试定位，Content 保存真正要被编译和执行的源码文本。
type Source struct {
	Name    string
	Content string
}

// NewStringSource 构造一份来自内存的 Lua 源码载荷。
// 如果调用方没有提供名字，会自动回退为 `<memory>`，方便错误信息定位。
func NewStringSource(name, content string) Source {
	sourceName := name
	if sourceName == "" {
		sourceName = "<memory>"
	}

	return Source{
		Name:    sourceName,
		Content: content,
	}
}

// NewFileSource 从磁盘读取 Lua 文件并构造成统一的源码载荷。
// 返回结果会保留清理后的文件路径，便于后续错误信息直接引用原始文件名。
func NewFileSource(path string) (Source, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Source{}, fmt.Errorf("read lua source file %q: %w", path, err)
	}

	return Source{
		Name:    filepath.Clean(path),
		Content: string(content),
	}, nil
}

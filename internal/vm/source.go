package vm

// Source 表示传入 VM 编译执行链路的一份 Lua 源码。
// Name 用于报错和调试，Content 保存实际源码文本。
type Source struct {
	Name    string
	Content string
}

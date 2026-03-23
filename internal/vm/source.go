package vm

// Source carries Lua script metadata and source code into the VM pipeline.
type Source struct {
	Name    string
	Content string
}

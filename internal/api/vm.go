package api

import (
	"context"
	"io"

	"github.com/yunfanli-dev/aLuavmWithGo/internal/vm"
)

// ModuleLoader 描述一个宿主侧模块 loader。
// 它会在 searcher 命中后被 `require` 调用，并接收当前模块名。
type ModuleLoader func(moduleName string) ([]Value, error)

// ModuleSearcher 描述一个宿主侧模块 searcher。
// 命中时返回 loader；未命中时返回一段可拼进 `require` 错误文本的 message。
type ModuleSearcher func(moduleName string) (ModuleLoader, string, error)

// VM 是提供给 Go 宿主使用的高层入口。
// 它把内部运行时状态封装起来，对外暴露脚本执行、宿主函数注册和运行参数配置等能力。
type VM struct {
	state *vm.State
}

// NewVM 创建一个新的 VM 实例，并初始化当前最小可用运行时。
// 返回的实例已经具备内建函数和基础全局环境，可以直接执行脚本。
func NewVM() *VM {
	return &VM{
		state: vm.NewState(),
	}
}

// ExecString 使用默认背景上下文执行一段内存中的 Lua 源码。
// 这是最直接的入口，适合简单调用或不需要取消控制的场景。
func (v *VM) ExecString(source string) error {
	return v.ExecStringWithContext(context.Background(), source)
}

// ExecStringWithContext 在给定上下文控制下执行一段内存中的 Lua 源码。
// 宿主可以通过 ctx 实现超时或主动取消，从而避免脚本长期占用执行线程。
func (v *VM) ExecStringWithContext(ctx context.Context, source string) error {
	return v.ExecSourceWithContext(ctx, NewStringSource("<memory>", source))
}

// ExecSource 执行一个已经构造好的源码载荷。
// 这个入口适合宿主先自行准备名字和内容，再交给 VM 统一执行。
func (v *VM) ExecSource(source Source) error {
	return v.ExecSourceWithContext(context.Background(), source)
}

// ExecSourceWithContext 在给定上下文控制下执行一个源码载荷。
// 它负责把公开 API 层的 Source 转成内部 VM 使用的 Source 结构。
func (v *VM) ExecSourceWithContext(ctx context.Context, source Source) error {
	return v.state.ExecSourceWithContext(ctx, vm.Source{
		Name:    source.Name,
		Content: source.Content,
	})
}

// ExecFile 从磁盘读取一个 Lua 文件并立即执行。
// 如果调用方不需要显式上下文控制，可以直接使用这个便捷入口。
func (v *VM) ExecFile(path string) error {
	return v.ExecFileWithContext(context.Background(), path)
}

// ExecFileWithContext 在给定上下文控制下读取并执行 Lua 文件。
// 它先把文件转成统一的 Source，再复用源码执行链路。
func (v *VM) ExecFileWithContext(ctx context.Context, path string) error {
	source, err := NewFileSource(path)
	if err != nil {
		return err
	}

	return v.ExecSourceWithContext(ctx, source)
}

// RegisterFunction 把一个 Go 宿主函数注册到 Lua 全局环境中。
// 注册后脚本可以按名称调用该函数，参数和返回值都通过统一的 Value 类型传递。
func (v *VM) RegisterFunction(name string, fn func(args []Value) ([]Value, error)) error {
	return v.state.RegisterFunction(name, func(args []vm.Value) ([]vm.Value, error) {
		return fn(args)
	})
}

// RegisterPreloadFunction 把一个 Go 宿主函数注册到 Lua `package.preload`。
// 注册后脚本可以通过 `require(name)` 直接加载这份宿主提供的内存模块。
func (v *VM) RegisterPreloadFunction(name string, fn func(args []Value) ([]Value, error)) error {
	return v.state.RegisterPreloadFunction(name, func(args []vm.Value) ([]vm.Value, error) {
		return fn(args)
	})
}

// RegisterLoadedModule 直接把一个固定模块值注册到 Lua `package.loaded`。
// 注册后脚本侧 `require(name)` 会直接返回这份缓存值。
func (v *VM) RegisterLoadedModule(name string, value Value) error {
	return v.state.RegisterLoadedModule(name, value)
}

// RegisterSearcherFunction 把一个 Go 宿主 searcher 注册到 Lua `package.loaders`。
// 注册后，`require` 会按当前顺序调用它，让宿主可以参与模块解析。
func (v *VM) RegisterSearcherFunction(searcher ModuleSearcher) error {
	return v.state.RegisterSearcherFunction(func(moduleName string) (vm.ModuleLoader, string, error) {
		loader, message, err := searcher(moduleName)
		if err != nil || loader == nil {
			return nil, message, err
		}

		return func(moduleName string) ([]vm.Value, error) {
			return loader(moduleName)
		}, message, nil
	})
}

// SetOutput 修改内建输出函数使用的目标 writer。
// 当前主要影响 `print` 这类需要把文本写出到宿主侧的能力。
func (v *VM) SetOutput(writer io.Writer) {
	v.state.SetOutput(writer)
}

// SetStepLimit 设置单次脚本执行允许消耗的最大步数。
// 传入正数时会启用最小预算保护；小于等于 0 时按“不限制”处理。
func (v *VM) SetStepLimit(limit int) {
	v.state.SetStepLimit(limit)
}

package vm

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"
)

// State 表示一次 Lua 执行上下文对应的 VM 运行时状态。
// 它持有全局环境、输出目标、随机数状态、最近一次编译结果以及执行返回值等信息。
type State struct {
	stack        *Stack
	globals      map[string]*valueCell
	output       io.Writer
	random       *rand.Rand
	stepLimit    int
	lastProgram  *FrontendResult
	lastReturned []Value
}

// NewState 创建一个新的 VM 状态对象，并初始化当前最小可用运行时。
// 该过程会准备栈、全局表、随机源以及内建函数注册。
func NewState() *State {
	state := &State{
		stack:   NewStack(),
		globals: make(map[string]*valueCell),
		output:  io.Discard,
		random:  rand.New(rand.NewSource(1)),
	}

	state.registerBuiltins()
	return state
}

// ExecString 使用默认背景上下文执行一段内存中的 Lua 源码。
// 这是 VM 内部最直接的字符串执行入口。
func (s *State) ExecString(source string) error {
	return s.ExecStringWithContext(context.Background(), source)
}

// ExecStringWithContext 在给定上下文控制下执行一段 Lua 源码。
// 该入口允许宿主通过 ctx 触发超时或主动取消。
func (s *State) ExecStringWithContext(ctx context.Context, source string) error {
	return s.ExecSourceWithContext(ctx, Source{
		Name:    "<memory>",
		Content: source,
	})
}

// ExecSource 执行一份已经构造好的源码载荷。
// 当调用方已经准备好名称和内容时，可以直接使用这个入口。
func (s *State) ExecSource(source Source) error {
	return s.ExecSourceWithContext(context.Background(), source)
}

// ExecSourceWithContext 在给定上下文控制下执行一份源码载荷。
// 它会先完成前端编译，再驱动执行器运行生成的 IR。
func (s *State) ExecSourceWithContext(ctx context.Context, source Source) error {
	trimmed := strings.TrimSpace(source.Content)
	if trimmed == "" {
		return nil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	sourceName := source.Name
	if sourceName == "" {
		sourceName = "<unknown>"
	}

	frontendResult, err := compileSource(Source{
		Name:    sourceName,
		Content: source.Content,
	})
	if err != nil {
		return err
	}

	s.lastProgram = frontendResult
	s.lastReturned = nil

	result, err := executeProgram(ctx, s, frontendResult.Program)
	if err != nil {
		return fmt.Errorf("execute compiled Lua source %q: %w", sourceName, err)
	}

	s.lastReturned = append([]Value(nil), result.returnValues...)
	return nil
}

// StackSize 返回当前操作数栈大小。
// 该方法主要用于测试验证和调试观察。
func (s *State) StackSize() int {
	return s.stack.Len()
}

// LastProgram 返回最近一次成功编译得到的前端结果。
// 它主要服务于测试和调试，不属于面向脚本作者的运行时能力。
func (s *State) LastProgram() *FrontendResult {
	return s.lastProgram
}

// LastReturnValues 返回最近一次脚本执行显式产生的返回值列表。
// 结果会复制一份返回，避免外部直接修改内部切片。
func (s *State) LastReturnValues() []Value {
	return append([]Value(nil), s.lastReturned...)
}

// SetOutput 修改内建输出函数使用的 writer。
// 传入 nil 时会回退到丢弃输出，避免调用方额外处理空 writer。
func (s *State) SetOutput(writer io.Writer) {
	if writer == nil {
		s.output = io.Discard
		return
	}

	s.output = writer
}

// SetStepLimit 配置单次脚本执行允许消耗的最大步数。
// 当前传入正数时启用预算保护，非正数则表示不限制。
func (s *State) SetStepLimit(limit int) {
	s.stepLimit = limit
}

// RegisterFunction 把 Go 宿主函数注册到 Lua 全局环境。
// 注册完成后，脚本可以通过给定名称直接调用这项宿主能力。
func (s *State) RegisterFunction(name string, fn NativeFunction) error {
	return s.registerNativeFunction(name, &nativeFunction{
		name: name,
		fn:   fn,
	})
}

func (s *State) registerContextualFunction(name string, fn contextualNativeFunction) error {
	return s.registerNativeFunction(name, &nativeFunction{
		name:           name,
		contextualImpl: fn,
	})
}

func (s *State) registerNativeFunction(name string, function *nativeFunction) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("register function with empty name")
	}

	if function == nil || (function.fn == nil && function.contextualImpl == nil) {
		return fmt.Errorf("register function %q with nil handler", name)
	}

	s.globals[name] = &valueCell{
		value: Value{
			Type: ValueTypeFunction,
			Data: function,
		},
	}

	return nil
}

func (s *State) registerBuiltinPrint() {
	_ = s.registerContextualFunction("print", func(exec *executor, args []Value) ([]Value, error) {
		parts := make([]string, 0, len(args))
		for _, arg := range args {
			text, err := exec.valueToString(arg)
			if err != nil {
				return nil, err
			}

			parts = append(parts, text)
		}

		if _, err := fmt.Fprintln(s.output, strings.Join(parts, "\t")); err != nil {
			return nil, err
		}

		return nil, nil
	})
}

// registerBuiltinClockMillis 注册一个最小 wall-clock 毫秒计时函数。
// 它主要用于样例脚本和手工回归，不等同于完整 Lua 时间库。
func (s *State) registerBuiltinClockMillis() {
	_ = s.RegisterFunction("clock_ms", func(args []Value) ([]Value, error) {
		if len(args) != 0 {
			return nil, fmt.Errorf("clock_ms expects no arguments")
		}

		return []Value{{Type: ValueTypeNumber, Data: float64(time.Now().UnixNano()) / 1e6}}, nil
	})
}

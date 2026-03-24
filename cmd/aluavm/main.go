package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/yunfanli-dev/aLuavmWithGo/internal/api"
)

// cliConfig 汇总当前 CLI 支持的最小运行参数集合。
// 这些字段只描述命令行层需要解析和传递给 VM 的配置，
// 不负责保存运行过程中派生出的临时状态。
type cliConfig struct {
	inlineSource string
	scriptPath   string
	stepLimit    int
	timeout      time.Duration
}

const (
	cliExitCodeSuccess = 0
	cliExitCodeRuntime = 1
	cliExitCodeUsage   = 2
)

func main() {
	vm := api.NewVM()
	vm.SetOutput(os.Stdout)

	config, err := parseArgs(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}

		fmt.Fprintln(os.Stderr, formatCLIError(err, cliErrorKindUsage))
		os.Exit(exitCodeForError(cliErrorKindUsage))
	}

	if err := run(vm, config); err != nil {
		kind := classifyRuntimeError(err)
		fmt.Fprintln(os.Stderr, formatCLIError(err, kind))
		os.Exit(exitCodeForError(kind))
	}

	if message := successMessage(config); message != "" {
		fmt.Println(message)
	}
}

// parseArgs 负责解析本次命令行调用的参数。
// 它会读取运行开关、可选脚本路径，并复用标准错误输出 help 信息。
func parseArgs(args []string) (cliConfig, error) {
	return parseArgsWithOutput(args, os.Stderr)
}

// parseArgsWithOutput 解析 CLI 参数，并把 help / usage 文本写到指定输出流。
// 这样测试可以捕获帮助文本，实际命令行入口也可以继续沿用同一套解析逻辑。
func parseArgsWithOutput(args []string, output io.Writer) (cliConfig, error) {
	flagSet := flag.NewFlagSet("aluavm", flag.ContinueOnError)
	flagSet.SetOutput(output)

	config := cliConfig{}
	flagSet.StringVar(&config.inlineSource, "e", "", "execute the given inline Lua source")
	flagSet.IntVar(&config.stepLimit, "step-limit", 0, "stop script execution after the given number of execution steps")
	flagSet.DurationVar(&config.timeout, "timeout", 0, "cancel script execution after the given duration, for example 500ms or 2s")
	flagSet.Usage = func() {
		writeUsage(flagSet.Output())
	}

	if err := flagSet.Parse(args); err != nil {
		return cliConfig{}, err
	}

	rest := flagSet.Args()
	if len(rest) > 1 {
		return cliConfig{}, fmt.Errorf("expected at most one script path")
	}
	if config.timeout < 0 {
		return cliConfig{}, fmt.Errorf("timeout must be >= 0")
	}
	if config.stepLimit < 0 {
		return cliConfig{}, fmt.Errorf("step limit must be >= 0")
	}
	if len(rest) == 1 {
		config.scriptPath = rest[0]
	}
	if config.inlineSource != "" && config.scriptPath != "" {
		return cliConfig{}, fmt.Errorf("inline source and script path cannot be used together")
	}

	return config, nil
}

// writeUsage 输出当前 CLI 的最小使用说明。
// 这里会集中列出脚本执行方式、内联执行方式和当前支持的运行限制参数，
// 便于用户在不读源码的情况下快速确认入口形状。
func writeUsage(output io.Writer) {
	if output == nil {
		return
	}

	_, _ = fmt.Fprintln(output, "Usage: aluavm [flags] [script.lua]")
	_, _ = fmt.Fprintln(output, "")
	_, _ = fmt.Fprintln(output, "Examples:")
	_, _ = fmt.Fprintln(output, `  aluavm -e 'print("hello")'`)
	_, _ = fmt.Fprintln(output, "  aluavm -timeout 50ms script.lua")
	_, _ = fmt.Fprintln(output, "  aluavm -step-limit 100 script.lua")
	_, _ = fmt.Fprintln(output, "")
	_, _ = fmt.Fprintln(output, "Flags:")
}

type cliErrorKind string

const (
	cliErrorKindUsage   cliErrorKind = "usage"
	cliErrorKindRuntime cliErrorKind = "runtime"
	cliErrorKindTimeout cliErrorKind = "timeout"
)

// classifyRuntimeError 把运行时错误归类为稳定的 CLI 错误种类。
// 当前主要区分“超时/取消”和“一般运行失败”，
// 以便 stderr 前缀和退出码能够保持可预期。
func classifyRuntimeError(err error) cliErrorKind {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return cliErrorKindTimeout
	}

	return cliErrorKindRuntime
}

// formatCLIError 按错误种类拼装面向用户的稳定错误文本。
// 这里故意把 usage、取消和普通执行失败分成不同前缀，
// 方便手工排障和脚本化调用时做简单分类。
func formatCLIError(err error, kind cliErrorKind) string {
	if err == nil {
		return ""
	}

	switch kind {
	case cliErrorKindUsage:
		return fmt.Sprintf("aluavm usage error: %v", err)
	case cliErrorKindTimeout:
		return fmt.Sprintf("aluavm execution canceled: %v", err)
	default:
		return fmt.Sprintf("aluavm execution failed: %v", err)
	}
}

// exitCodeForError 把当前 CLI 错误种类映射到进程退出码。
// 约定上参数错误返回 2，其余执行失败类错误返回 1，
// 成功则由主流程保持 0。
func exitCodeForError(kind cliErrorKind) int {
	switch kind {
	case cliErrorKindUsage:
		return cliExitCodeUsage
	case cliErrorKindRuntime, cliErrorKindTimeout:
		return cliExitCodeRuntime
	default:
		return cliExitCodeRuntime
	}
}

// successMessage 决定本次调用成功后是否需要额外输出成功提示。
// 当前只有“空跑 bootstrap 检查”会输出状态行；
// 真正执行脚本或内联源码成功时保持安静，避免污染脚本输出。
func successMessage(config cliConfig) string {
	if config.inlineSource != "" || config.scriptPath != "" {
		return ""
	}

	return "aluavm bootstrap ready"
}

// run 按当前 CLI 配置分发执行路径。
// 它会先把步数预算和超时上下文注入 VM，
// 然后在“执行文件”“执行内联源码”“空源码自检”之间选择一条路径。
func run(vm *api.VM, config cliConfig) error {
	vm.SetStepLimit(config.stepLimit)

	ctx := context.Background()
	cancel := func() {}
	if config.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, config.timeout)
	}
	defer cancel()

	if config.scriptPath == "" {
		if config.inlineSource != "" {
			return vm.ExecStringWithContext(ctx, config.inlineSource)
		}

		return vm.ExecStringWithContext(ctx, "")
	}

	// TODO: 后续把 CLI 扩展为支持更丰富的运行参数以及多源码输入工作流，
	// 例如组合多个脚本源、显式设置更多 VM 运行选项等。
	return vm.ExecFileWithContext(ctx, config.scriptPath)
}

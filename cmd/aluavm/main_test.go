package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yunfanli-dev/aLuavmWithGo/internal/api"
)

func TestParseArgsAcceptsTimeoutAndScriptPath(t *testing.T) {
	config, err := parseArgsWithOutput([]string{"-timeout", "250ms", "-step-limit", "123", "test.lua"}, io.Discard)
	if err != nil {
		t.Fatalf("parse args: %v", err)
	}

	if config.timeout != 250*time.Millisecond {
		t.Fatalf("unexpected timeout: %v", config.timeout)
	}
	if config.stepLimit != 123 {
		t.Fatalf("unexpected step limit: %d", config.stepLimit)
	}
	if config.scriptPath != "test.lua" {
		t.Fatalf("unexpected script path: %q", config.scriptPath)
	}
}

func TestParseArgsAcceptsInlineSource(t *testing.T) {
	config, err := parseArgsWithOutput([]string{"-e", "return 42"}, io.Discard)
	if err != nil {
		t.Fatalf("parse args: %v", err)
	}

	if config.inlineSource != "return 42" {
		t.Fatalf("unexpected inline source: %q", config.inlineSource)
	}
	if config.scriptPath != "" {
		t.Fatalf("expected empty script path, got %q", config.scriptPath)
	}
}

func TestParseArgsRejectsNegativeTimeout(t *testing.T) {
	if _, err := parseArgsWithOutput([]string{"-timeout", "-1s"}, io.Discard); err == nil {
		t.Fatal("expected negative timeout to fail")
	}
}

func TestParseArgsRejectsNegativeStepLimit(t *testing.T) {
	if _, err := parseArgsWithOutput([]string{"-step-limit", "-1"}, io.Discard); err == nil {
		t.Fatal("expected negative step limit to fail")
	}
}

func TestParseArgsRejectsInlineSourceAndScriptPathTogether(t *testing.T) {
	if _, err := parseArgsWithOutput([]string{"-e", "return 1", "test.lua"}, io.Discard); err == nil {
		t.Fatal("expected inline source and script path combination to fail")
	}
}

func TestParseArgsWritesCustomHelp(t *testing.T) {
	var output bytes.Buffer

	if _, err := parseArgsWithOutput([]string{"-h"}, &output); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("expected help error, got %v", err)
	}

	helpText := output.String()
	if !strings.Contains(helpText, "Usage: aluavm [flags] [script.lua]") {
		t.Fatalf("unexpected help text: %q", helpText)
	}
	if !strings.Contains(helpText, `aluavm -e 'print("hello")'`) {
		t.Fatalf("missing inline example in help text: %q", helpText)
	}
	if !strings.Contains(helpText, "-step-limit") || !strings.Contains(helpText, "-timeout") {
		t.Fatalf("missing runtime flags in help text: %q", helpText)
	}
}

func TestFormatCLIErrorForUsage(t *testing.T) {
	message := formatCLIError(errors.New("timeout must be >= 0"), cliErrorKindUsage)
	if message != "aluavm usage error: timeout must be >= 0" {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestFormatCLIErrorForRuntime(t *testing.T) {
	message := formatCLIError(errors.New("execution step limit exceeded"), cliErrorKindRuntime)
	if message != "aluavm execution failed: execution step limit exceeded" {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestFormatCLIErrorForTimeout(t *testing.T) {
	message := formatCLIError(context.DeadlineExceeded, classifyRuntimeError(context.DeadlineExceeded))
	if message != "aluavm execution canceled: context deadline exceeded" {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestExitCodeForUsageError(t *testing.T) {
	if code := exitCodeForError(cliErrorKindUsage); code != cliExitCodeUsage {
		t.Fatalf("unexpected exit code: %d", code)
	}
}

func TestExitCodeForRuntimeError(t *testing.T) {
	if code := exitCodeForError(cliErrorKindRuntime); code != cliExitCodeRuntime {
		t.Fatalf("unexpected exit code: %d", code)
	}
}

func TestExitCodeForTimeoutError(t *testing.T) {
	if code := exitCodeForError(cliErrorKindTimeout); code != cliExitCodeRuntime {
		t.Fatalf("unexpected exit code: %d", code)
	}
}

func TestSuccessMessageForBootstrapOnly(t *testing.T) {
	if message := successMessage(cliConfig{}); message != "aluavm bootstrap ready" {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestSuccessMessageIsSilentForInlineSource(t *testing.T) {
	if message := successMessage(cliConfig{inlineSource: "return 1"}); message != "" {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestSuccessMessageIsSilentForScriptPath(t *testing.T) {
	if message := successMessage(cliConfig{scriptPath: "test.lua"}); message != "" {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestCLIProcessBootstrapOutputAndExitCode(t *testing.T) {
	result := runCLIProcess(t)

	if result.exitCode != cliExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if result.stdout != "aluavm bootstrap ready\n" {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}
}

func TestCLIProcessInlineSourceStaysSilentOnSuccess(t *testing.T) {
	result := runCLIProcess(t, "-e", `print("hello")`)

	if result.exitCode != cliExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if result.stdout != "hello\n" {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}
}

func TestCLIProcessUsageErrorExitCode(t *testing.T) {
	result := runCLIProcess(t, "-step-limit", "-1")

	if result.exitCode != cliExitCodeUsage {
		t.Fatalf("unexpected exit code: %d", result.exitCode)
	}
	if !strings.Contains(result.stderr, "aluavm usage error: step limit must be >= 0") {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}
}

func TestCLIProcessRuntimeErrorExitCode(t *testing.T) {
	scriptPath := filepath.Join(t.TempDir(), "loop.lua")
	if err := os.WriteFile(scriptPath, []byte("while true do end"), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	result := runCLIProcess(t, "-step-limit", "20", scriptPath)

	if result.exitCode != cliExitCodeRuntime {
		t.Fatalf("unexpected exit code: %d", result.exitCode)
	}
	if !strings.Contains(result.stderr, "aluavm execution failed:") {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}
}

func TestCLIProcessRuntimeShowcaseExample(t *testing.T) {
	result := runCLIProcess(t, repoRelativePath("examples", "runtime_showcase.lua"))

	if result.exitCode != cliExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}

	expectedLines := []string{
		"sort\t1,2,3\n",
		"method\t42\n",
		"meta_index\tfrom_meta\n",
		"meta_tostring\tobject:40\n",
		"string\tdesserts\tLUA\n",
		"math\t9\t7\n",
	}
	for _, line := range expectedLines {
		if !strings.Contains(result.stdout, line) {
			t.Fatalf("missing line %q in stdout %q", line, result.stdout)
		}
	}
	if strings.Contains(result.stdout, "aluavm bootstrap ready") {
		t.Fatalf("unexpected bootstrap status in script stdout: %q", result.stdout)
	}
}

func TestCLIProcessMultivalueShowcaseExample(t *testing.T) {
	result := runCLIProcess(t, repoRelativePath("examples", "multivalue_showcase.lua"))

	if result.exitCode != cliExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}

	expectedLines := []string{
		"assign\t1\tleft\tright\n",
		"paren\tleft\tleft\n",
		"pcall\tfalse\tboom\n",
		"table\thead|left|right\n",
		"byte\t65\t90\n",
	}
	for _, line := range expectedLines {
		if !strings.Contains(result.stdout, line) {
			t.Fatalf("missing line %q in stdout %q", line, result.stdout)
		}
	}
	if strings.Contains(result.stdout, "aluavm bootstrap ready") {
		t.Fatalf("unexpected bootstrap status in script stdout: %q", result.stdout)
	}
}

func TestCLIProcessTableMetatableShowcaseExample(t *testing.T) {
	result := runCLIProcess(t, repoRelativePath("examples", "table_metatable_showcase.lua"))

	if result.exitCode != cliExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}

	expectedLines := []string{
		"identity\ttable-a\ttable-b\tfn-a\tfn-b\n",
		"chain\tfrom-chain\t42\n",
	}
	for _, line := range expectedLines {
		if !strings.Contains(result.stdout, line) {
			t.Fatalf("missing line %q in stdout %q", line, result.stdout)
		}
	}
	if strings.Contains(result.stdout, "aluavm bootstrap ready") {
		t.Fatalf("unexpected bootstrap status in script stdout: %q", result.stdout)
	}
}

func TestCLIProcessGenericForShowcaseExample(t *testing.T) {
	result := runCLIProcess(t, repoRelativePath("examples", "generic_for_showcase.lua"))

	if result.exitCode != cliExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}

	expectedLines := []string{
		"custom_iter\t66\n",
		"gmatch_iter\ta|a\n",
	}
	for _, line := range expectedLines {
		if !strings.Contains(result.stdout, line) {
			t.Fatalf("missing line %q in stdout %q", line, result.stdout)
		}
	}
	if strings.Contains(result.stdout, "aluavm bootstrap ready") {
		t.Fatalf("unexpected bootstrap status in script stdout: %q", result.stdout)
	}
}

func TestCLIProcessLoopTimingScript(t *testing.T) {
	result := runCLIProcess(t, repoRelativePath("test3.lua"))

	if result.exitCode != cliExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}

	if !strings.Contains(result.stdout, "iterations\t1000\n") {
		t.Fatalf("missing iterations line in stdout %q", result.stdout)
	}
	if !strings.Contains(result.stdout, "sum\t500500\n") {
		t.Fatalf("missing sum line in stdout %q", result.stdout)
	}
	if !strings.Contains(result.stdout, "elapsed_ms\t") {
		t.Fatalf("missing elapsed_ms line in stdout %q", result.stdout)
	}
	if strings.Contains(result.stdout, "aluavm bootstrap ready") {
		t.Fatalf("unexpected bootstrap status in script stdout: %q", result.stdout)
	}
}

func TestCLIProcessMethodScript(t *testing.T) {
	result := runCLIProcess(t, repoRelativePath("test.lua"))

	if result.exitCode != cliExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}
	if result.stdout != "result:\t42\n" {
		t.Fatalf("unexpected stdout: %q", result.stdout)
	}
}

func TestCLIProcessMixedRuntimeScript(t *testing.T) {
	result := runCLIProcess(t, repoRelativePath("test2.lua"))

	if result.exitCode != cliExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.exitCode, result.stderr)
	}
	if result.stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.stderr)
	}

	expectedLines := []string{
		"counter:\t1\t3\n",
		"pcall success:\ttrue\t7\t8\n",
		"pcall fail:\tfalse\tboom\n",
		"table total:\t6\n",
		"metatable:\tobj(15)\tmissing:name\n",
		"mixed:\t22\n",
	}
	for _, line := range expectedLines {
		if !strings.Contains(result.stdout, line) {
			t.Fatalf("missing line %q in stdout %q", line, result.stdout)
		}
	}
	if strings.Contains(result.stdout, "aluavm bootstrap ready") {
		t.Fatalf("unexpected bootstrap status in script stdout: %q", result.stdout)
	}
}

func TestRunExecutesInlineSource(t *testing.T) {
	vm := api.NewVM()
	var output bytes.Buffer
	vm.SetOutput(&output)

	if err := run(vm, cliConfig{inlineSource: `print("inline-ok")`}); err != nil {
		t.Fatalf("run inline source: %v", err)
	}

	if output.String() != "inline-ok\n" {
		t.Fatalf("unexpected output: %q", output.String())
	}
}

func TestRunExecutesScriptWithStepLimit(t *testing.T) {
	vm := api.NewVM()
	scriptPath := filepath.Join(t.TempDir(), "loop.lua")

	if err := os.WriteFile(scriptPath, []byte("while true do end"), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	err := run(vm, cliConfig{
		scriptPath: scriptPath,
		stepLimit:  20,
	})
	if err == nil {
		t.Fatal("expected step limit error")
	}
	if err.Error() != `execute compiled Lua source "`+scriptPath+`": execution step limit exceeded` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunExecutesScriptWithTimeoutContext(t *testing.T) {
	vm := api.NewVM()
	scriptPath := filepath.Join(t.TempDir(), "loop.lua")

	if err := os.WriteFile(scriptPath, []byte("while true do end"), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	err := run(vm, cliConfig{
		scriptPath: scriptPath,
		timeout:    time.Nanosecond,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !containsContextDeadline(err) {
		t.Fatalf("unexpected error: %v", err)
	}
}

// containsContextDeadline 判断当前错误是否来自 CLI 超时上下文到期。
// 这里单独抽成 helper，便于测试在不依赖完整错误字符串的情况下校验取消路径。
func containsContextDeadline(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}

func TestCLIHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	args := helperProcessArgs(os.Args)
	os.Args = append([]string{"aluavm"}, args...)
	main()
	os.Exit(cliExitCodeSuccess)
}

type cliProcessResult struct {
	exitCode int
	stdout   string
	stderr   string
}

// runCLIProcess 把当前测试二进制再次拉起为子进程执行。
// 这样可以覆盖真实 CLI 进程的 stdout、stderr 和退出码，
// 而不仅仅是直接调用内部函数后的返回值。
func runCLIProcess(t *testing.T, args ...string) cliProcessResult {
	t.Helper()

	commandArgs := append([]string{"-test.run=TestCLIHelperProcess", "--"}, args...)
	cmd := exec.Command(os.Args[0], commandArgs...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := cliExitCodeSuccess
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run helper process: %v", err)
		}
		exitCode = exitErr.ExitCode()
	}

	return cliProcessResult{
		exitCode: exitCode,
		stdout:   stdout.String(),
		stderr:   stderr.String(),
	}
}

// helperProcessArgs 提取 helper 子进程分隔符 `--` 之后的 CLI 参数。
// 测试会把真正要传给 main 的参数挂在这个分隔符后面，
// 从而避免和 `go test` 自己的参数混在一起。
func helperProcessArgs(args []string) []string {
	for index, arg := range args {
		if arg == "--" {
			return append([]string(nil), args[index+1:]...)
		}
	}

	return nil
}

// repoRelativePath 从 `cmd/aluavm` 包目录回退到仓库根目录再拼接目标路径。
// 这让测试在当前包目录下执行时，仍然能稳定定位仓库内的样例脚本。
func repoRelativePath(parts ...string) string {
	pathParts := append([]string{"..", ".."}, parts...)
	return filepath.Join(pathParts...)
}

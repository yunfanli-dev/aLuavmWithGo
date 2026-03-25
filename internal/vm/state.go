package vm

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ModuleLoader 描述一个宿主侧模块 loader。
// 它会在 `require` 解析命中后被调用，并接收当前模块名。
type ModuleLoader func(moduleName string) ([]Value, error)

// ModuleSearcher 描述一个宿主侧模块 searcher。
// 命中时返回 loader；未命中时返回一段可拼进 `require` 错误文本的 message。
type ModuleSearcher func(moduleName string) (ModuleLoader, string, error)

// State 表示一次 Lua 执行上下文对应的 VM 运行时状态。
// 它持有全局环境、输出目标、随机数状态、最近一次编译结果以及执行返回值等信息。
type State struct {
	stack   *Stack
	globals map[string]*valueCell
	// globalEnv 保存对外暴露给 Lua 的最小 `_G` 全局环境表。
	// 普通全局名和 `_G.name` 会同步读写到同一份全局状态。
	globalEnv    *table
	output       io.Writer
	random       *rand.Rand
	stepLimit    int
	lastProgram  *FrontendResult
	lastReturned []Value
	// loadingModules 记录当前仍处于加载中的模块文件路径。
	// 它用于在 `require` 形成直接或间接环时及时报错。
	loadingModules map[string]struct{}
	// sourceStack 维护当前执行链路上的源码名栈。
	// `require` 会依赖它定位“当前源码目录”来解析相对模块路径。
	sourceStack []string
}

// NewState 创建一个新的 VM 状态对象，并初始化当前最小可用运行时。
// 该过程会准备栈、全局表、随机源以及内建函数注册。
func NewState() *State {
	state := &State{
		stack:          NewStack(),
		globals:        make(map[string]*valueCell),
		globalEnv:      newTable(),
		output:         io.Discard,
		random:         rand.New(rand.NewSource(1)),
		loadingModules: make(map[string]struct{}),
	}

	state.setGlobalValue("_G", Value{Type: ValueTypeTable, Data: state.globalEnv})
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
	_, err := s.executeSourceWithEnv(ctx, source, true, s.globalEnv)
	return err
}

// executeSourceWithContext 在给定上下文控制下执行一份源码载荷，并按需更新对外可见的最近执行结果。
// `require` 等内部链路会复用它执行子模块，但不会覆盖宿主视角的 `LastProgram` / `LastReturnValues`。
func (s *State) executeSourceWithContext(ctx context.Context, source Source, recordLast bool) ([]Value, error) {
	return s.executeSourceWithEnv(ctx, source, recordLast, s.globalEnv)
}

// executeSourceWithEnv 在给定上下文控制下执行一份源码载荷，并允许调用方指定线程环境表。
// `require` 等内部链路会用它把调用者线程环境继续传给子模块。
func (s *State) executeSourceWithEnv(ctx context.Context, source Source, recordLast bool, rootEnv *table) ([]Value, error) {
	trimmed := strings.TrimSpace(source.Content)
	if trimmed == "" {
		return nil, nil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
		return nil, err
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
		return nil, err
	}

	if recordLast {
		s.lastProgram = frontendResult
		s.lastReturned = nil
	}

	s.pushSourceName(sourceName)
	defer s.popSourceName()

	result, err := executeProgramWithEnv(ctx, s, frontendResult.Program, rootEnv)
	if err != nil {
		return nil, fmt.Errorf("execute compiled Lua source %q: %w", sourceName, err)
	}

	returnValues := append([]Value(nil), result.returnValues...)
	if recordLast {
		s.lastReturned = append([]Value(nil), returnValues...)
	}

	return returnValues, nil
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

// pushSourceName 把一份正在执行的源码名压入当前源码栈。
// 这样嵌套 `require` 时仍能知道“当前模块”是从哪一个文件继续解析出来的。
func (s *State) pushSourceName(name string) {
	s.sourceStack = append(s.sourceStack, name)
}

// popSourceName 弹出当前最内层的一份源码名。
// 当执行链路返回上层模块或顶层脚本时，会配对调用这个 helper。
func (s *State) popSourceName() {
	if len(s.sourceStack) == 0 {
		return
	}

	s.sourceStack = s.sourceStack[:len(s.sourceStack)-1]
}

// currentSourceName 返回当前最内层正在执行的源码名。
// 如果当前没有活动源码，则返回空字符串。
func (s *State) currentSourceName() string {
	if len(s.sourceStack) == 0 {
		return ""
	}

	return s.sourceStack[len(s.sourceStack)-1]
}

// requireModule 按当前最小模块加载规则解析、执行并缓存一个 Lua 模块。
// 当前会优先相对正在执行的源码目录查找，再回退到工作目录，并支持 `.lua` 后缀补全。
func (s *State) requireModule(ctx context.Context, rootEnv *table, moduleName string) (Value, error) {
	if strings.TrimSpace(moduleName) == "" {
		return NilValue(), fmt.Errorf("require expects non-empty module name")
	}
	if rootEnv == nil {
		rootEnv = s.globalEnv
	}

	loadedModules, err := s.ensurePackageLoadedTable()
	if err != nil {
		return NilValue(), err
	}

	loadedValue, alreadyLoaded, err := loadedModules.get(Value{Type: ValueTypeString, Data: moduleName})
	if err != nil {
		return NilValue(), err
	}
	if packageLoadedHit(alreadyLoaded, loadedValue) {
		return loadedValue, nil
	}

	loaders, err := s.ensurePackageLoadersTable()
	if err != nil {
		return NilValue(), err
	}

	exec := newExecutorWithEnv(ctx, s, rootEnv)
	searchErrors := make([]string, 0, 2)
	for index := 1; ; index++ {
		searcher, exists, err := loaders.get(Value{Type: ValueTypeNumber, Data: float64(index)})
		if err != nil {
			return NilValue(), err
		}
		if !exists || searcher.Type == ValueTypeNil {
			break
		}

		searcherValues, err := exec.callFunctionValue(searcher, []Value{{Type: ValueTypeString, Data: moduleName}})
		if err != nil {
			return NilValue(), err
		}
		if len(searcherValues) == 0 || searcherValues[0].Type == ValueTypeNil {
			continue
		}
		if searcherValues[0].Type == ValueTypeString {
			searchErrors = append(searchErrors, searcherValues[0].Data.(string))
			continue
		}

		loaderValues, err := exec.callFunctionValue(searcherValues[0], []Value{{Type: ValueTypeString, Data: moduleName}})
		if err != nil {
			return NilValue(), err
		}

		loadedValue, alreadyLoaded, err = loadedModules.get(Value{Type: ValueTypeString, Data: moduleName})
		if err != nil {
			return NilValue(), err
		}
		if packageLoadedHit(alreadyLoaded, loadedValue) {
			return loadedValue, nil
		}

		moduleValue := packageModuleResult(loaderValues)

		if err := loadedModules.set(Value{Type: ValueTypeString, Data: moduleName}, moduleValue); err != nil {
			return NilValue(), err
		}

		return moduleValue, nil
	}

	return NilValue(), fmt.Errorf(`module %q not found%s`, moduleName, strings.Join(searchErrors, ""))
}

// requirePreloadModule 通过最小 `package.preload` loader 执行一份内存模块。
// 当前会把模块名作为唯一参数传给 loader，并复用 `package.loaded` 记录缓存结果。
func (s *State) requirePreloadModule(ctx context.Context, rootEnv *table, moduleName string, loader Value, loadedModules *table) (Value, error) {
	loadingKey := "preload:" + moduleName
	if _, loading := s.loadingModules[loadingKey]; loading {
		return NilValue(), fmt.Errorf("loop in require chain for module %q", moduleName)
	}
	if rootEnv == nil {
		rootEnv = s.globalEnv
	}

	s.loadingModules[loadingKey] = struct{}{}
	defer delete(s.loadingModules, loadingKey)

	exec := newExecutorWithEnv(ctx, s, rootEnv)
	returnValues, err := exec.callFunctionValue(loader, []Value{{Type: ValueTypeString, Data: moduleName}})
	if err != nil {
		return NilValue(), err
	}

	loadedValue, alreadyLoaded, err := loadedModules.get(Value{Type: ValueTypeString, Data: moduleName})
	if err != nil {
		return NilValue(), err
	}
	if packageLoadedHit(alreadyLoaded, loadedValue) {
		return loadedValue, nil
	}

	moduleValue := packageModuleResult(returnValues)

	if err := loadedModules.set(Value{Type: ValueTypeString, Data: moduleName}, moduleValue); err != nil {
		return NilValue(), err
	}

	return moduleValue, nil
}

// resolveRequirePath 根据最小 `require` 规则把模块名解析成实际模块文件路径。
// 当前支持基于 `package.path` 的 `?` 模板替换，并优先相对当前源码目录查找。
func (s *State) resolveRequirePath(moduleName string) (string, error) {
	candidates := s.resolveRequireCandidates(moduleName)
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}

		if info.IsDir() {
			continue
		}

		return candidate, nil
	}

	return "", fmt.Errorf("module %q not found", moduleName)
}

// packageLoadedHit 判断 `package.loaded[name]` 当前是否应被视为“已加载命中”。
// 这里按 Lua 5.1 的最小规则，只把非 nil 且 truthy 的值视为命中。
func packageLoadedHit(exists bool, value Value) bool {
	return exists && value.Type != ValueTypeNil && isTruthy(value)
}

// packageModuleResult 根据 loader 返回值整理最终要写回 `package.loaded` 的模块结果。
// 当 loader 没返回值，或者首值是 nil / false 时，当前最小语义都会回落成 true。
func packageModuleResult(returnValues []Value) Value {
	if len(returnValues) == 0 {
		return Value{Type: ValueTypeBoolean, Data: true}
	}

	moduleValue := returnValues[0]
	if moduleValue.Type == ValueTypeNil {
		return Value{Type: ValueTypeBoolean, Data: true}
	}

	if moduleValue.Type == ValueTypeBoolean {
		booleanValue, _ := moduleValue.Data.(bool)
		if !booleanValue {
			return Value{Type: ValueTypeBoolean, Data: true}
		}
	}

	return moduleValue
}

// setGlobalValue 写入一项全局名称，并把结果同步到 `_G` 环境表。
// 这样普通全局访问、`_G.name` 访问和相关 builtin 都能观察到同一份值。
func (s *State) setGlobalValue(name string, value Value) {
	if existing, ok := s.globals[name]; ok {
		existing.value = value
	} else {
		s.globals[name] = &valueCell{value: value}
	}

	if s.globalEnv != nil {
		_ = s.globalEnv.set(Value{Type: ValueTypeString, Data: name}, value)
	}
}

// lookupGlobalValue 按名称读取一项全局值。
// 该 helper 主要用于 `_G` 相关桥接路径，缺失时返回 `ok=false`。
func (s *State) lookupGlobalValue(name string) (Value, bool) {
	cell, ok := s.globals[name]
	if !ok {
		return NilValue(), false
	}

	return cell.value, true
}

// isGlobalEnv 判断给定 table 是否就是当前运行时暴露出去的 `_G` 环境表。
// `_G` 的特殊读写桥接会依赖这个判断。
func (s *State) isGlobalEnv(tableValue *table) bool {
	return s != nil && s.globalEnv != nil && tableValue == s.globalEnv
}

// ensureModuleTable 创建或复用一份模块表，并把它同步到给定可见环境路径和 `package.loaded`。
// 当前只负责最小模块表注册，不处理旧 Lua 5.1 的完整环境切换语义。
func (s *State) ensureModuleTable(rootEnv *table, moduleName string) (*table, error) {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return nil, fmt.Errorf("module expects non-empty name")
	}
	if rootEnv == nil {
		rootEnv = s.globalEnv
	}

	loadedTable, err := s.ensurePackageLoadedTable()
	if err != nil {
		return nil, err
	}

	moduleValue, exists, err := loadedTable.get(Value{Type: ValueTypeString, Data: moduleName})
	if err != nil {
		return nil, err
	}

	var moduleTable *table
	if exists && moduleValue.Type != ValueTypeNil {
		if moduleValue.Type != ValueTypeTable {
			return nil, fmt.Errorf("package.loaded[%q] is not a table", moduleName)
		}

		existingTable, ok := moduleValue.Data.(*table)
		if !ok {
			return nil, fmt.Errorf("invalid package.loaded[%q] payload %T", moduleName, moduleValue.Data)
		}

		moduleTable = existingTable
	} else {
		moduleTable = newTable()
	}

	if err := s.bindModulePath(rootEnv, moduleName, moduleTable); err != nil {
		return nil, err
	}

	if err := loadedTable.set(Value{Type: ValueTypeString, Data: moduleName}, Value{
		Type: ValueTypeTable,
		Data: moduleTable,
	}); err != nil {
		return nil, err
	}

	return moduleTable, nil
}

// bindModulePath 把模块表挂到给定可见环境的点分路径上。
// 例如 `foo.bar` 会确保 `root.foo.bar` 指向同一份模块表，并在缺失时补中间 table。
func (s *State) bindModulePath(rootEnv *table, moduleName string, moduleTable *table) error {
	parts := strings.Split(moduleName, ".")
	if len(parts) == 0 {
		return fmt.Errorf("module expects non-empty name")
	}

	currentTable := rootEnv
	for index, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return fmt.Errorf("module %q contains empty path segment", moduleName)
		}

		value := Value{Type: ValueTypeTable, Data: moduleTable}
		if index != len(parts)-1 {
			existing, ok, err := currentTable.get(Value{Type: ValueTypeString, Data: part})
			if err != nil {
				return err
			}

			if ok && existing.Type != ValueTypeNil {
				if existing.Type != ValueTypeTable {
					return fmt.Errorf("module path %q conflicts with non-table value", strings.Join(parts[:index+1], "."))
				}

				nextTable, ok := existing.Data.(*table)
				if !ok {
					return fmt.Errorf("invalid module path payload %T", existing.Data)
				}

				currentTable = nextTable
				continue
			}

			nextTable := newTable()
			if err := currentTable.set(Value{Type: ValueTypeString, Data: part}, Value{
				Type: ValueTypeTable,
				Data: nextTable,
			}); err != nil {
				return err
			}

			if currentTable == s.globalEnv {
				s.setGlobalValue(part, Value{Type: ValueTypeTable, Data: nextTable})
			}

			currentTable = nextTable
			continue
		}

		if err := currentTable.set(Value{Type: ValueTypeString, Data: part}, value); err != nil {
			return err
		}

		if currentTable == s.globalEnv {
			s.setGlobalValue(part, value)
		}
	}

	return nil
}

// modulePackagePrefix 计算模块名对应的包名前缀。
// 例如 `foo.bar` 会返回 `foo.`，没有点分前缀时返回空字符串。
func modulePackagePrefix(moduleName string) string {
	lastDot := strings.LastIndex(moduleName, ".")
	if lastDot < 0 {
		return ""
	}

	return moduleName[:lastDot+1]
}

// resolveRequireCandidates 根据当前最小 `package.path` 规则生成候选模块文件路径。
// 这些候选既会被文件 searcher 逐个尝试，也会用于拼装缺失模块错误文本。
func (s *State) resolveRequireCandidates(moduleName string) []string {
	return s.resolveModuleCandidates(moduleName, s.packagePath(), ".", string(os.PathSeparator))
}

// resolveModuleCandidates 按给定模板文本、分隔符和替换符生成模块候选路径列表。
// 当前 `require` 和 `package.searchpath` 都会复用这套最小路径展开规则。
func (s *State) resolveModuleCandidates(moduleName string, pathTemplate string, separator string, replacement string) []string {
	candidates := make([]string, 0, 8)
	seenCandidates := make(map[string]struct{})
	addCandidate := func(candidate string) {
		if candidate == "" {
			return
		}

		cleaned := filepath.Clean(candidate)
		if _, exists := seenCandidates[cleaned]; exists {
			return
		}

		seenCandidates[cleaned] = struct{}{}
		candidates = append(candidates, cleaned)
	}

	searchPatterns := strings.Split(pathTemplate, ";")
	modulePath := moduleName
	if separator != "" {
		modulePath = strings.ReplaceAll(modulePath, separator, replacement)
	}
	if len(searchPatterns) == 0 {
		searchPatterns = []string{"?.lua", "?/init.lua"}
	}

	searchDirs := []string{"."}
	currentSource := s.currentSourceName()
	if currentSource != "" && !strings.HasPrefix(currentSource, "<") {
		searchDirs = append([]string{filepath.Dir(currentSource)}, searchDirs...)
	}

	for _, pattern := range searchPatterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		candidatePattern := strings.ReplaceAll(pattern, "?", modulePath)
		if filepath.IsAbs(candidatePattern) {
			addCandidate(candidatePattern)
			continue
		}

		for _, searchDir := range searchDirs {
			addCandidate(filepath.Join(searchDir, candidatePattern))
			if filepath.Ext(candidatePattern) == "" {
				addCandidate(filepath.Join(searchDir, candidatePattern+".lua"))
			}
		}
	}

	return candidates
}

// ensurePackageLoadedTable 返回最小 `package.loaded` 表，并在缺失时自动补齐。
// `require` 会通过它暴露模块缓存，也会尊重脚本侧对该表的直接写入。
func (s *State) ensurePackageLoadedTable() (*table, error) {
	packageTable, err := s.ensurePackageLibrary()
	if err != nil {
		return nil, err
	}

	loadedValue, exists, err := packageTable.get(Value{Type: ValueTypeString, Data: "loaded"})
	if err != nil {
		return nil, err
	}
	if exists && loadedValue.Type == ValueTypeTable {
		loadedTable, ok := loadedValue.Data.(*table)
		if !ok {
			return nil, fmt.Errorf("invalid package.loaded payload %T", loadedValue.Data)
		}

		return loadedTable, nil
	}
	if exists && loadedValue.Type != ValueTypeNil {
		return nil, fmt.Errorf("package.loaded is not a table")
	}

	loadedTable := newTable()
	if err := packageTable.set(Value{Type: ValueTypeString, Data: "loaded"}, Value{
		Type: ValueTypeTable,
		Data: loadedTable,
	}); err != nil {
		return nil, err
	}

	return loadedTable, nil
}

// ensurePackagePreloadTable 返回最小 `package.preload` 表，并在缺失时自动补齐。
// `require` 会先查询它，允许脚本或宿主注册内存模块 loader。
func (s *State) ensurePackagePreloadTable() (*table, error) {
	packageTable, err := s.ensurePackageLibrary()
	if err != nil {
		return nil, err
	}

	preloadValue, exists, err := packageTable.get(Value{Type: ValueTypeString, Data: "preload"})
	if err != nil {
		return nil, err
	}
	if exists && preloadValue.Type == ValueTypeTable {
		preloadTable, ok := preloadValue.Data.(*table)
		if !ok {
			return nil, fmt.Errorf("invalid package.preload payload %T", preloadValue.Data)
		}

		return preloadTable, nil
	}
	if exists && preloadValue.Type != ValueTypeNil {
		return nil, fmt.Errorf("package.preload is not a table")
	}

	preloadTable := newTable()
	if err := packageTable.set(Value{Type: ValueTypeString, Data: "preload"}, Value{
		Type: ValueTypeTable,
		Data: preloadTable,
	}); err != nil {
		return nil, err
	}

	return preloadTable, nil
}

// ensurePackageLoadersTable 返回最小 `package.loaders` 表，并在缺失时自动补齐。
// `require` 会按顺序调用其中的 searcher，允许脚本插入自定义模块解析逻辑。
func (s *State) ensurePackageLoadersTable() (*table, error) {
	packageTable, err := s.ensurePackageLibrary()
	if err != nil {
		return nil, err
	}

	loadersValue, exists, err := packageTable.get(Value{Type: ValueTypeString, Data: "loaders"})
	if err != nil {
		return nil, err
	}
	if exists && loadersValue.Type == ValueTypeTable {
		loadersTable, ok := loadersValue.Data.(*table)
		if !ok {
			return nil, fmt.Errorf("invalid package.loaders payload %T", loadersValue.Data)
		}

		return loadersTable, nil
	}
	if exists && loadersValue.Type != ValueTypeNil {
		return nil, fmt.Errorf("package.loaders is not a table")
	}

	loadersTable := newTable()
	if err := packageTable.set(Value{Type: ValueTypeString, Data: "loaders"}, Value{
		Type: ValueTypeTable,
		Data: loadersTable,
	}); err != nil {
		return nil, err
	}

	return loadersTable, nil
}

// ensurePackageLibrary 返回全局 `package` 表，并在缺失时按最小运行时约定补齐。
// 当前会保证后续 `package.loaded`、`package.preload`、`package.path`
// 和 `package.loaders` 这些字段有机会被继续补齐。
func (s *State) ensurePackageLibrary() (*table, error) {
	if existing, ok := s.globals["package"]; ok {
		if existing.value.Type != ValueTypeTable {
			return nil, fmt.Errorf("package library is not a table")
		}

		packageTable, ok := existing.value.Data.(*table)
		if !ok {
			return nil, fmt.Errorf("invalid package library payload %T", existing.value.Data)
		}

		return packageTable, nil
	}

	packageTable := newTable()
	s.setGlobalValue("package", Value{Type: ValueTypeTable, Data: packageTable})
	return packageTable, nil
}

// packagePath 读取当前最小 `package.path` 文本。
// 当脚本尚未改写该字段时，会回退到默认的 Lua 文件搜索模板。
func (s *State) packagePath() string {
	packageTable, err := s.ensurePackageLibrary()
	if err != nil {
		return "?.lua;?/init.lua"
	}

	pathValue, exists, err := packageTable.get(Value{Type: ValueTypeString, Data: "path"})
	if err != nil || !exists || pathValue.Type != ValueTypeString {
		return "?.lua;?/init.lua"
	}

	path, ok := pathValue.Data.(string)
	if !ok || strings.TrimSpace(path) == "" {
		return "?.lua;?/init.lua"
	}

	return path
}

// packagePreloadSearcher 实现最小 `package.loaders` preload searcher。
// 命中时返回 loader；未命中时返回搜索失败文本，供 `require` 汇总。
func (s *State) packagePreloadSearcher(exec *executor, args []Value) ([]Value, error) {
	if len(args) < 1 || args[0].Type != ValueTypeString {
		return nil, fmt.Errorf("package preload searcher expects module name")
	}

	moduleName := args[0].Data.(string)
	preloadTable, err := s.ensurePackagePreloadTable()
	if err != nil {
		return nil, err
	}

	loader, exists, err := preloadTable.get(Value{Type: ValueTypeString, Data: moduleName})
	if err != nil {
		return nil, err
	}
	if exists && loader.Type != ValueTypeNil {
		loadedModules, err := s.ensurePackageLoadedTable()
		if err != nil {
			return nil, err
		}

		wrappedLoader := &nativeFunction{
			name: "package.loader.preload(" + moduleName + ")",
			contextualImpl: func(exec *executor, args []Value) ([]Value, error) {
				moduleRootEnv := exec.threadEnv()
				if callerEnv, err := exec.envByLevel(2); err == nil && callerEnv != nil {
					moduleRootEnv = callerEnv
				}

				moduleValue, err := s.requirePreloadModule(exec.ctx, moduleRootEnv, moduleName, loader, loadedModules)
				if err != nil {
					return nil, err
				}

				return []Value{moduleValue}, nil
			},
		}

		return []Value{{Type: ValueTypeFunction, Data: wrappedLoader}}, nil
	}

	return []Value{{Type: ValueTypeString, Data: "\n\tno field package.preload['" + moduleName + "']"}}, nil
}

// packageFileSearcher 实现最小 `package.loaders` 文件 searcher。
// 命中时返回延迟执行的文件 loader；未命中时返回所有候选路径组成的错误文本。
func (s *State) packageFileSearcher(exec *executor, args []Value) ([]Value, error) {
	if len(args) < 1 || args[0].Type != ValueTypeString {
		return nil, fmt.Errorf("package file searcher expects module name")
	}

	moduleName := args[0].Data.(string)
	modulePath, err := s.resolveRequirePath(moduleName)
	if err != nil {
		candidates := s.resolveRequireCandidates(moduleName)
		var builder strings.Builder
		for _, candidate := range candidates {
			builder.WriteString("\n\tno file '")
			builder.WriteString(candidate)
			builder.WriteString("'")
		}
		return []Value{{Type: ValueTypeString, Data: builder.String()}}, nil
	}

	loader := &nativeFunction{
		name: "package.loader.file(" + modulePath + ")",
		contextualImpl: func(exec *executor, args []Value) ([]Value, error) {
			if _, loading := s.loadingModules[modulePath]; loading {
				return nil, fmt.Errorf("loop in require chain for module %q", moduleName)
			}

			content, err := os.ReadFile(modulePath)
			if err != nil {
				return nil, fmt.Errorf("read module %q: %w", moduleName, err)
			}

			s.loadingModules[modulePath] = struct{}{}
			defer delete(s.loadingModules, modulePath)

			moduleRootEnv := exec.threadEnv()
			if callerEnv, err := exec.envByLevel(2); err == nil && callerEnv != nil {
				moduleRootEnv = callerEnv
			}

			return s.executeSourceWithEnv(exec.ctx, Source{
				Name:    modulePath,
				Content: string(content),
			}, false, moduleRootEnv)
		},
	}

	return []Value{{Type: ValueTypeFunction, Data: loader}}, nil
}

// RegisterFunction 把 Go 宿主函数注册到 Lua 全局环境。
// 注册完成后，脚本可以通过给定名称直接调用这项宿主能力。
func (s *State) RegisterFunction(name string, fn NativeFunction) error {
	return s.registerNativeFunction(name, &nativeFunction{
		name: name,
		fn:   fn,
	})
}

// RegisterPreloadFunction 把一个 Go 宿主函数注册到 `package.preload`。
// 注册完成后，脚本可以通过 `require(name)` 直接命中这份内存模块 loader。
func (s *State) RegisterPreloadFunction(name string, fn NativeFunction) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("register preload function with empty name")
	}

	if fn == nil {
		return fmt.Errorf("register preload function %q with nil handler", name)
	}

	preloadTable, err := s.ensurePackagePreloadTable()
	if err != nil {
		return err
	}

	return preloadTable.set(Value{Type: ValueTypeString, Data: name}, Value{
		Type: ValueTypeFunction,
		Data: &nativeFunction{
			name: "package.preload." + name,
			fn:   fn,
		},
	})
}

// RegisterLoadedModule 直接把一个固定模块值注册到 `package.loaded`。
// 注册完成后，脚本侧 `require(name)` 会直接命中这份缓存值，而不会再继续搜索 loader。
func (s *State) RegisterLoadedModule(name string, value Value) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("register loaded module with empty name")
	}

	loadedTable, err := s.ensurePackageLoadedTable()
	if err != nil {
		return err
	}

	return loadedTable.set(Value{Type: ValueTypeString, Data: name}, value)
}

// RegisterSearcherFunction 把一个 Go 宿主 searcher 注册到 `package.loaders` 末尾。
// 注册完成后，`require` 会在内建 searcher 之后按顺序调用这份宿主解析逻辑。
func (s *State) RegisterSearcherFunction(searcher ModuleSearcher) error {
	if searcher == nil {
		return fmt.Errorf("register searcher function with nil handler")
	}

	loadersTable, err := s.ensurePackageLoadersTable()
	if err != nil {
		return err
	}

	maximum, err := loadersTable.maxNumericKey()
	if err != nil {
		return err
	}

	index := int(maximum) + 1
	return loadersTable.set(Value{Type: ValueTypeNumber, Data: float64(index)}, Value{
		Type: ValueTypeFunction,
		Data: &nativeFunction{
			name: fmt.Sprintf("package.loaders.host.%d", index),
			contextualImpl: func(exec *executor, args []Value) ([]Value, error) {
				if len(args) < 1 || args[0].Type != ValueTypeString {
					return nil, fmt.Errorf("host package loader searcher expects module name")
				}

				moduleName := args[0].Data.(string)
				loader, message, err := searcher(moduleName)
				if err != nil {
					return nil, err
				}
				if loader == nil {
					if message == "" {
						return nil, nil
					}

					return []Value{{Type: ValueTypeString, Data: message}}, nil
				}

				wrappedLoader := &nativeFunction{
					name: "package.loader.host(" + moduleName + ")",
					fn: func(args []Value) ([]Value, error) {
						return loader(moduleName)
					},
				}

				return []Value{{Type: ValueTypeFunction, Data: wrappedLoader}}, nil
			},
		},
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

	s.setGlobalValue(name, Value{
		Type: ValueTypeFunction,
		Data: function,
	})

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

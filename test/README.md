# Test README

## 当前可重复验证方式

- 运行 `go test ./...` 验证当前 Lua 5.1 子集运行时是否可编译并通过单元测试。
- 运行 `go run ./cmd/aluavm` 验证最小 demo 入口是否可启动。
- 运行 `go run ./cmd/aluavm -h` 验证 CLI help / usage 输出是否展示当前支持的运行参数和示例。
- 运行 `go run ./cmd/aluavm -e 'print("hello")'` 验证 CLI 内联源码执行入口是否可直接运行短脚本。
- 运行 `go run ./cmd/aluavm <script.lua>` 验证文件加载、前端编译与当前运行时链路是否可执行。
- 运行 `go run ./cmd/aluavm -step-limit 50 <script.lua>` 验证 CLI 级执行步数预算入口是否能终止死循环脚本。
- 运行 `go run ./cmd/aluavm -timeout 50ms <script.lua>` 验证 CLI 级超时取消入口是否能终止长时间运行脚本。
- 运行错误参数或死循环脚本时，检查 CLI 错误前缀是否能区分 `usage error`、`execution failed` 和 `execution canceled`。
- 检查 CLI 成功输出和退出码约定：空跑保留 `aluavm bootstrap ready`，脚本成功执行保持安静，参数错误退出码为 `2`，运行时失败退出码为 `1`。
- `cmd/aluavm` 当前已补充进程级 CLI 集成测试，覆盖真实子进程下的 stdout、stderr 和 exit code 行为。
- `examples/runtime_showcase.lua` 和 `examples/multivalue_showcase.lua` 当前都已接入 CLI 进程级集成测试，关键输出会被自动回归校验。
- `test3.lua` 当前也已接入 CLI 进程级集成测试，自动校验 `iterations`、`sum` 和 `elapsed_ms` 输出形状，但不会把耗时数值写死。
- `test.lua` 和 `test2.lua` 当前也已接入 CLI 进程级集成测试，分别覆盖基础方法调用链路，以及闭包、`pcall`、`ipairs`、metatable 组合链路。
- 运行 `go run ./cmd/aluavm ./test3.lua` 做一遍基础循环性能脚本手工回归。
- 运行 `go run ./cmd/aluavm ./examples/runtime_showcase.lua` 做一遍运行时主链路手工回归。
- 运行 `go run ./cmd/aluavm ./examples/multivalue_showcase.lua` 做一遍多返回值与错误处理手工回归。
- 结合 [RegressionChecklist.md](./RegressionChecklist.md) 对照关键能力点做回归检查。
- 结合 [GapChecklist.md](./GapChecklist.md) 查看当前剩余空白，判断后续应优先补能力还是补验证。
- 如果需要评估脚本死循环和宿主中断问题，先看 [ExecutionSafetyPlan.md](./ExecutionSafetyPlan.md)。
- 如果需要验证执行预算保护，可在 Go 侧配置 `SetStepLimit(...)` 后运行循环脚本或死循环脚本测试。
- 如果需要验证 CLI help / usage 输出，可运行 `go run ./cmd/aluavm -h`，预期展示 `-e`、`-step-limit`、`-timeout` 与脚本路径示例。
- 如果需要验证 CLI 内联执行入口，可运行 `go run ./cmd/aluavm -e 'print("hello")'`，预期输出 `hello`。
- 如果需要验证 CLI 步数预算入口，可对死循环脚本运行 `go run ./cmd/aluavm -step-limit 50 <script.lua>`，预期返回 `execution step limit exceeded`。
- 如果需要验证 CLI 错误分类输出，可分别触发非法参数、步数预算耗尽和超时取消，预期前缀分别为 `aluavm usage error:`、`aluavm execution failed:` 和 `aluavm execution canceled:`。
- 如果需要验证 CLI 成功输出和退出码，可分别运行空参数、合法脚本和非法参数，预期只有空参数打印 `aluavm bootstrap ready`，并且参数错误退出码为 `2`、运行时失败退出码为 `1`。
- 如果需要验证自动化 CLI 集成测试，可运行 `go test ./cmd/aluavm`，关注 helper 子进程用例是否仍能覆盖真实进程级 stdout、stderr 和 exit code。
- 如果需要验证自动化样例回归，可运行 `go test ./cmd/aluavm`，关注 `runtime_showcase.lua` 和 `multivalue_showcase.lua` 的关键输出断言是否通过。
- 如果需要验证自动化 `test3.lua` 回归，可运行 `go test ./cmd/aluavm`，关注 `iterations	1000`、`sum	500500` 和 `elapsed_ms` 行是否通过断言。
- 如果需要验证自动化 `test.lua` / `test2.lua` 回归，可运行 `go test ./cmd/aluavm`，关注基础方法调用输出和闭包 / `pcall` / metatable 组合输出是否通过断言。
- 如果需要验证宿主主动取消，可使用 `ExecStringWithContext` / `ExecSourceWithContext` / `ExecFileWithContext` 配合已取消 `context.Context` 做回归。
- 如果需要验证 CLI 超时入口，可对死循环脚本运行 `go run ./cmd/aluavm -timeout 50ms <script.lua>`，预期返回 `context deadline exceeded`。
- `SetStepLimit(...)` 默认未开启；只有显式设置正数预算时才会生效，`limit <= 0` 会按不限制处理。

## 当前覆盖范围

- VM 实例创建
- 基础状态初始化
- 空 Lua 源码的 bootstrap 执行入口
- Lua 脚本文件读取与统一 source 入口
- Lua 5.1 子集的基础词法切分
- Lua 5.1 子集的基础 AST 解析
- Lua 5.1 子集的基础 IR 编译链路
- Lua 5.1 子集的最小 IR 执行能力
- Lua 5.1 子集的基础控制流执行
- Lua 5.1 子集的基础函数声明与调用
- Go 宿主函数注册与最小标准输出能力
- Lua 5.1 子集的最小 table 构造与索引
- Lua 5.1 子集的 `local function` 与匿名函数基础能力
- Lua 5.1 子集的最小基础内建函数
- Lua 5.1 子集的基础错误处理与 protected call
- Lua 5.1 子集的 `repeat-until` 与数值 `for`
- Lua 5.1 子集的 generic `for` 与 `pairs` / `ipairs` / `next`
- Lua 5.1 子集的最小 metatable 读写行为
- Lua 5.1 子集的 `__tostring`、`__call`、`rawget`、`rawset`
- Lua 5.1 子集的基础算术与拼接元方法
- Lua 5.1 子集的基础比较元方法
- Lua 5.1 子集的 `__metatable` 保护行为
- Lua 5.1 子集的最小 `#table` 连续数组段长度语义
- Lua 5.1 子集的 `do ... end` 与 `break`
- Lua 5.1 子集的基础 `vararg`
- Lua 5.1 子集的 table 构造器最后数组字段多返回值展开
- Lua 5.1 子集的圆括号单值语义
- Lua 5.1 子集的基础方法定义与方法调用语法
- Lua 5.1 子集的 table / string 调用语法糖
- Lua 5.1 子集的指数形式数字字面量
- Lua 5.1 子集的十六进制数字字面量
- Lua 5.1 子集的 long string / long comment

## 备注

- 当前 lexer 已支持基础关键字、标识符、十进制、指数形式和十六进制数字、短字符串、long bracket 字符串与注释、短注释和常用运算符；更完整的 Lua 词法细节仍待补齐。
- 当前 parser 已支持 `local`、赋值、`return`、`if`、`while`、`repeat-until`、数值 `for`、generic `for`、`do ... end`、`break`、`vararg`、函数声明、方法定义、table 构造、匿名函数、普通调用、方法调用以及 `fn{...}` / `fn"..."` 这类调用语法糖，并会拒绝非法作用域中的 `...`；更多 Lua 5.1 语法仍待补齐。
- 当前已支持 `local`、赋值、`return`、`if`、`while`、`repeat-until`、数值 `for`、generic `for`、`do ... end`、`break`、`vararg`、函数调用、方法调用、table / string 调用语法糖、table 读写、闭包基础能力和基础一元/二元表达式的执行，并已支持字符串长度和最小 `#table` 连续数组段长度语义；完整多返回值语义和更多 Lua 5.1 细节仍待补齐。
- 当前 `string.find` / `string.match` / `string.gmatch` / `string.gsub` / `string.format` 已支持最小可用子集；其中 `find` / `match` / `gmatch` 支持纯文本查找与 Lua 风格起始下标，`gsub` 支持纯文本字符串 / table / function 替换器与替换次数返回，`format` 支持少量高频格式符；Lua 5.1 更完整的 pattern / capture / replacer / format 语义仍未实现。
- 当前已支持最小执行步数限制 `SetStepLimit(...)`、基于 `context.Context` 的宿主主动取消，以及 CLI `-h` / `-e` / `-step-limit` / `-timeout`、基础错误分类输出和最小成功输出 / 退出码约定；更严格的 instruction budget 仍未实现。
- 当前已支持 Go 宿主向 Lua 注册基础函数，并内置最小 `print`；标准库仍远未完整。
- 当前已内置 `print`、`clock_ms`、`type`、`tostring`、`tonumber`、`select`、`unpack`、`assert`、`error`、`pcall`、`xpcall`、`next`、`pairs`、`ipairs`、`rawequal`，以及最小 `table.getn` / `table.maxn` / `table.foreach` / `table.foreachi` / `table.insert` / `table.remove` / `table.concat` / `table.sort`、`math.pi` / `math.huge`、`math.abs` / `math.floor` / `math.ceil` / `math.modf` / `math.fmod` / `math.deg` / `math.rad` / `math.frexp` / `math.ldexp` / `math.max` / `math.min` / `math.sqrt` / `math.pow` / `math.random` / `math.randomseed` / `math.log` / `math.log10` / `math.exp` / `math.sinh` / `math.cosh` / `math.tanh` / `math.sin` / `math.cos` / `math.tan` / `math.atan` / `math.atan2` / `math.asin` / `math.acos` 和 `string.find` / `string.match` / `string.gmatch` / `string.gsub` / `string.format` / `string.len` / `string.sub` / `string.lower` / `string.upper` / `string.rep` / `string.reverse` / `string.byte` / `string.char`；这仍只是较小的基础内建子集。
- 当前已支持 `error`、`pcall`、基础 `vararg`，并支持最后一个函数调用或 `...` 在返回列表中的多返回值展开、table 构造器最后一个数组字段的展开，以及圆括号抑制展开的单值语义；更完整的 Lua 多返回值规则仍未全部覆盖。
- 当前回归测试已覆盖多返回值在返回列表、赋值、空 `vararg`、`vararg` 赋值、函数实参列表、`vararg` 实参列表、方法调用、`pcall` 成功/失败路径，以及 table 构造器最后数组字段中的常见调整规则；更完整的 Lua 多返回值边界仍未全部覆盖。
- 当前已支持 `{}`、键值字段、`t[k]`、`t.name` 的最小读写，以及基础 `setmetatable` / `getmetatable`、`__metatable` 保护、`__index` / `__newindex`、`__tostring`、`__call`、`rawget`、`rawset`、`rawequal`、`__add`、`__sub`、`__mul`、`__div`、`__mod`、`__pow`、`__unm`、`__concat`、`__eq`、`__lt`、`__le`，以及最小 `#table` / `table.getn` 连续数组段长度语义和 `table.maxn` 最大数值键语义；完整 table 行为仍未实现。
- 当前已支持 `local function`、匿名函数表达式和基础 upvalue 读写；闭包仍未覆盖完整 Lua 5.1 upvalue 语义。
- 当前 generic `for` 主要面向 `pairs` / `ipairs` / `next` 这一最小可用链路；更完整的迭代器兼容性仍待补齐。

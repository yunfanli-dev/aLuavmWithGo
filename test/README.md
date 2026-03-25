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
- `examples/table_metatable_showcase.lua` 当前也已接入 CLI 进程级集成测试，覆盖对象 key、`rawget` / `rawset` 和 `__index` / `__newindex` table 回退链路。
- `examples/generic_for_showcase.lua` 当前也已接入 CLI 进程级集成测试，覆盖自定义 iterator 三元组和 `string.gmatch` 接入 generic `for` 的链路。
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
- 如果需要验证自动化 `test.lua` / `test2.lua` / `examples/table_metatable_showcase.lua` / `examples/generic_for_showcase.lua` 回归，可运行 `go test ./cmd/aluavm`，关注基础方法调用输出、闭包 / `pcall` / metatable 组合输出、对象 key / table 回退链路，以及 generic `for` 专项样例是否通过断言。
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
  说明：当前已直接覆盖受保护 metatable 既不能改成新 table，也不能通过 `setmetatable(target, nil)` 被移除。
  说明：当前已直接覆盖未设置 metatable 的普通 table 和 `_G` 都会通过 `getmetatable(target)` 返回 `nil`、未受保护 `getmetatable(target)` 返回原 metatable 对象、未受保护 metatable 可被新值直接覆盖、`setmetatable(target, nil)` 的最小移除语义、`setmetatable(_G, nil)` 的最小无副作用清空路径、移除后 `getmetatable(target)` 回到 `nil` 的路径、`setmetatable` 返回原 table 的最小约定，以及 `getmetatable` / `setmetatable` 的最小参数个数、多余参数忽略、非 table 首参和非法第二参数规则。
  说明：当前已直接覆盖 `__index` / `__newindex` 函数回退收到原 table、自身 key 和写入值的最小传参顺序。
- Lua 5.1 子集的 `__tostring`、`__call`、`rawget`、`rawset`
  说明：当前已直接覆盖 table callable 会把原 table 作为 `__call` 的第一个参数传入，后续显式实参保持原顺序。
  说明：当前也已直接覆盖 `rawget` / `rawset` 的最小参数个数、多余参数忽略、非 table 首参报错路径，以及 `rawset` 返回原 table、`rawget` 读不到返回 `nil`、缺失键 `rawget` 不触发 `__index` 回退、`rawset(target, key, nil)` 删除底层键后普通索引重新走 `__index`、`rawset(_G, key, nil)` 后普通全局读写与 `rawget(_G, key)` 一致回到 `nil`、删除后 `_G["string-key"]` 也会一致回到 `nil`、删除后 `_G[non-string-key]` 与 `rawget(_G, non-string-key)` 也会一致回到 `nil`、`_G` 的 raw access 特殊桥接只对字符串键生效、`_G["string-key"]` 会继续观察到桥接后的全局值、`rawset(_G, "name", value/nil)` 也会与 `_G.name`、裸全局和 `rawget(_G, "name")` 保持同步、`_G["string-key"] = value/nil` 也会与裸全局和 `rawget(_G, key)` 保持同步、`_G.name = value/nil` 也会与裸全局和 `rawget(_G, "name")` 保持同步、`_G.name = value` 会立即同步到裸全局、`_G["string-key"] = nil` 后 `_G.name` 也会一致回到 `nil`、普通全局赋值也会反向同步到 `_G["string-key"]`、`_G.name` 和 `rawget(_G, key)`、普通全局置 `nil` 也会反向同步清掉这些观察路径、`_G["string-key"] = value` 会立即同步到裸全局、`_G["missing"]` 和 `_G.missing` 都会与裸全局和 `rawget(_G, "missing")` 一致回到 `nil`、普通全局删除后 `_G.name` 也会一致回到 `nil`、`_G[non-string-key]` 也会继续走普通 table 路径、`_G[non-string-key] = value/nil` 也会继续只影响普通 table 存储、不会污染裸全局名、`rawget/rawset(_G, ...)` 与普通全局读写同步的最小返回值与状态语义。
- Lua 5.1 子集的基础算术与拼接元方法
  说明：当前已直接覆盖 `__concat` 在左右操作数都可提供元方法时按“先左后右”查找的最小优先级。
- Lua 5.1 子集的基础比较元方法
  说明：当前已直接覆盖共享 `__eq` / `__lt` / `__le` 规则，以及 `<=` / `>=` 优先走共享 `__le`、仅在 `__le` 缺失时才经共享 `__lt` 回退，且“不共享则报错”的最小路径。
- Lua 5.1 子集的 `__metatable` 保护行为
- Lua 5.1 子集的最小 `#table` 正整数边界长度语义
  说明：当前已直接覆盖 dense table、存在索引 `1` 的 sparse table、empty table 和缺失前缀索引的 sparse table 这几类最小边界。
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
- 当前已支持 `local`、赋值、`return`、`if`、`while`、`repeat-until`、数值 `for`、generic `for`、`do ... end`、`break`、`vararg`、函数调用、方法调用、table / string 调用语法糖、table 读写、闭包基础能力和基础一元/二元表达式的执行，并已支持未声明名称赋值默认回落到当前函数环境、最小 `_G` 全局环境同步访问、最小 `getfenv` / `setfenv` 调用栈与线程级环境切换、栈级 `setfenv(level, env)` 持续同步回函数对象、`setfenv(fn, env)` 对当前函数和当前活跃外层函数帧的即时生效、native 函数 `getfenv(fn)` 跟随当前线程环境、字符串长度、算术系对可解析数字字符串的最小强转，以及最小 `#table` 正整数边界长度语义；完整多返回值语义和更多 Lua 5.1 细节仍待补齐。
- 当前 `string.find` / `string.match` / `string.gfind` / `string.gmatch` / `string.gsub` / `string.format` 已支持最小可用子集；其中 `find` / `match` / `gfind` / `gmatch` 支持纯文本查找与 Lua 风格起始下标，`gsub` 支持纯文本字符串 / table / function 替换器与替换次数返回，`format` 支持 `%c` / `%o` / `%u` / `%x` / `%X` / `%e` / `%E` / `%g` / `%G` 在内的少量高频格式符；Lua 5.1 更完整的 pattern / capture / replacer / format 语义仍未实现。
- 当前已支持最小执行步数限制 `SetStepLimit(...)`、基于 `context.Context` 的宿主主动取消，以及 CLI `-h` / `-e` / `-step-limit` / `-timeout`、基础错误分类输出和最小成功输出 / 退出码约定；更严格的 instruction budget 仍未实现。
- 当前已支持 Go 宿主向 Lua 注册基础函数、最小 preload 模块 loader、最小自定义 module searcher、最小直接 loaded-module 注入，并内置最小 `print`；标准库仍远未完整。
- 当前已内置 `print`、`clock_ms`、`type`、`tostring`、`tonumber`、`getfenv`、`setfenv`、`module`、`require`、最小 `package` / `package.preload` / `package.loaders` / `package.searchpath` / `package.seeall`、`select`、`unpack`、`assert`、`error`、`pcall`、`xpcall`、`next`、`pairs`、`ipairs`、`rawequal`，以及最小 `table.getn` / `table.maxn` / `table.foreach` / `table.foreachi` / `table.insert` / `table.remove` / `table.concat` / `table.sort`、`math.pi` / `math.huge`、`math.abs` / `math.floor` / `math.ceil` / `math.modf` / `math.mod` / `math.fmod` / `math.deg` / `math.rad` / `math.frexp` / `math.ldexp` / `math.max` / `math.min` / `math.sqrt` / `math.pow` / `math.random` / `math.randomseed` / `math.log` / `math.log10` / `math.exp` / `math.sinh` / `math.cosh` / `math.tanh` / `math.sin` / `math.cos` / `math.tan` / `math.atan` / `math.atan2` / `math.asin` / `math.acos` 和 `string.find` / `string.match` / `string.gfind` / `string.gmatch` / `string.gsub` / `string.format` / `string.len` / `string.sub` / `string.lower` / `string.upper` / `string.rep` / `string.reverse` / `string.byte` / `string.char`；这仍只是较小的基础内建子集。
- 当前已支持 `error`、`pcall`、基础 `vararg`，并支持最后一个函数调用或 `...` 在返回列表中的多返回值展开、table 构造器最后一个数组字段的展开，以及圆括号抑制展开的单值语义；更完整的 Lua 多返回值规则仍未全部覆盖。
- 当前回归测试已覆盖多返回值在返回列表、赋值、空 `vararg`、`vararg` 赋值、函数实参列表、`vararg` 实参列表、方法调用、`pcall` / `xpcall` 成功失败路径、`select` / `unpack` 默认边界长度，以及 table 构造器最后数组字段中的常见调整规则、方法调用在返回列表末尾、嵌套表达式列表和 table 构造器最后字段中的展开 / 圆括号抑制、`select` / `unpack` / `pcall` / `xpcall` 展开抑制、`assert` / `next` 展开与圆括号抑制、`pairs` / `ipairs` 展开与圆括号抑制，和 `assert` / `next` / `pairs` / `ipairs` 在返回列表、调用实参列表里的非末尾单值 / 末尾展开调整、`...`、方法调用、`assert` / `select` / `unpack`、`select(2, pcall(...))` / `select(2, xpcall(...))` 作为 generic `for` 最后迭代表达式时的展开语义、这些路径在圆括号下的单值抑制，以及这些调用位于 generic `for` 非末尾位置时的单值语义；其中方法调用在 generic `for` 末尾展开 / 非末尾压单值这对规则也已直接锁进回归。更完整的 Lua 多返回值边界仍未全部覆盖。
- 当前 `require` / `package` / `module` 已覆盖相对当前源码目录的最小文件模块加载、`package.loaded` 缓存复用 / 手动预填、`package.preload` 内存 loader、`package.path` 搜索模板、`package.loaders` 自定义 searcher、`package.searchpath` 路径探测、`package.seeall`、最小 `module(...)` 表注册、调用它的 Lua 帧切到模块环境、点分模块路径挂到调用点当前可见环境、`package.seeall` 和 `require` 子模块继续沿用调用点当前可见环境、`require` 子文件里再走 `module(..., package.seeall)` 时也会保持这套环境视角、在自定义函数环境里定义的 `package.preload` / 脚本侧 `package.loaders` loader 也已有交叉回归覆盖其后续 `require` 调用链、宿主侧 preload 模块、host searcher 和直接 loaded-module 注入也已有 API 回归覆盖函数环境里的 `require` 调用链、后置 `package.seeall(_M)` 不会再回退到模块表自身、后置 `package.seeall(_M)` 在自定义函数环境下也会继续沿用该函数环境并保持模块路径写回同一环境、`seeall` 内部状态不会污染 Lua 可见 metatable 字段、顶层 chunk 下 `getfenv(0)` 同步观察模块环境、宿主侧 preload 模块注册、宿主侧自定义 searcher 注册、宿主侧直接 loaded-module 注入、无返回值模块回落为 `true`，以及明显循环加载、`package.loaded = nil` 后重新加载和 `package.loaded = false` 后重新加载的基础回归；完整 Lua 5.1 `package` / 环境语义仍未实现。
- 当前已支持 `{}`、键值字段、`t[k]`、`t.name` 的最小读写，以及基础 `setmetatable` / `getmetatable`、`__metatable` 保护、`__index` / `__newindex`、`__tostring`、`__call`、`rawget`、`rawset`、`rawequal`、`__add`、`__sub`、`__mul`、`__div`、`__mod`、`__pow`、`__unm`、`__concat`、`__eq`、`__lt`、`__le`，以及非法 `__index` / `__newindex` 元方法值和明显链式环、非法 `__call` 链式自引用或环的基础报错路径，还有算术系对可解析数字字符串的最小强转、原生 `..` 对“双方都必须是字符串或数字”的最小收口、同类型字符串的直接有序比较、最小 `#table` / `table.getn` 正整数边界长度语义、`table.maxn` 最大数值键语义、`table.foreach` / `table.foreachi` 对 `__call` 和当前边界长度的最小复用、`table.concat` 对当前边界长度和 `__tostring` 的最小复用、`table.insert` / `table.remove` 对当前边界长度的最小复用、`table.sort` 对当前边界长度、`__lt` 和 `__call` comparator 的最小复用、table / function key 的对象身份匹配，以及 `pcall` / `xpcall` 对 `__call` 的最小复用、`__eq` 共享元方法规则、`__lt` 共享元方法规则和 `<=` / `>=` 对共享 `__le` / 反向 `__lt` 的最小回退；完整 table 行为仍未实现。
- 当前已支持 `local function`、匿名函数表达式和基础 upvalue 读写；闭包仍未覆盖完整 Lua 5.1 upvalue 语义。
- 当前 generic `for` 已覆盖 `pairs` / `ipairs` / `next` 主链路，并已补充自定义 iterator 三元组、带 `__call` 的 iterator、`string.gmatch` 接入循环，以及 `...` / 方法调用 / builtin / protected call 作为最后迭代表达式时的多返回值调整回归和这些路径在圆括号下的单值抑制、作为非末尾迭代表达式时的单值语义回归；更完整的迭代器兼容性仍待补齐。

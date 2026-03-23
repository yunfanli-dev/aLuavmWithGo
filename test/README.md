# Test README

## 当前可重复验证方式

- 运行 `go test ./...` 验证当前 Lua 5.1 子集运行时是否可编译并通过单元测试。
- 运行 `go run ./cmd/aluavm` 验证最小 demo 入口是否可启动。
- 运行 `go run ./cmd/aluavm <script.lua>` 验证文件加载、前端编译与当前运行时链路是否可执行。
- 运行 `go run ./cmd/aluavm ./examples/runtime_showcase.lua` 做一遍运行时主链路手工回归。
- 运行 `go run ./cmd/aluavm ./examples/multivalue_showcase.lua` 做一遍多返回值与错误处理手工回归。
- 结合 [RegressionChecklist.md](./RegressionChecklist.md) 对照关键能力点做回归检查。
- 结合 [GapChecklist.md](./GapChecklist.md) 查看当前剩余空白，判断后续应优先补能力还是补验证。
- 如果需要评估脚本死循环和宿主中断问题，先看 [ExecutionSafetyPlan.md](./ExecutionSafetyPlan.md)。

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
- 当前已支持 `local`、赋值、`return`、`if`、`while`、`repeat-until`、数值 `for`、generic `for`、`do ... end`、`break`、`vararg`、函数调用、方法调用、table / string 调用语法糖、table 读写、闭包基础能力和基础一元/二元表达式的执行；完整多返回值语义和更多 Lua 5.1 细节仍待补齐。
- 当前已支持 Go 宿主向 Lua 注册基础函数，并内置最小 `print`；标准库仍远未完整。
- 当前已内置 `print`、`type`、`tostring`、`tonumber`、`select`、`unpack`、`assert`、`error`、`pcall`、`xpcall`、`next`、`pairs`、`ipairs`、`rawequal`，以及最小 `table.insert` / `table.remove` / `table.concat` / `table.sort`、`math.abs` / `math.floor` / `math.ceil` / `math.max` / `math.min` / `math.sqrt` / `math.pow` / `math.random` / `math.randomseed` / `math.log` / `math.exp` / `math.sin` / `math.cos` 和 `string.len` / `string.sub` / `string.lower` / `string.upper` / `string.rep` / `string.reverse` / `string.byte` / `string.char`；这仍只是较小的基础内建子集。
- 当前已支持 `error`、`pcall`、基础 `vararg`，并支持最后一个函数调用或 `...` 在返回列表中的多返回值展开、table 构造器最后一个数组字段的展开，以及圆括号抑制展开的单值语义；更完整的 Lua 多返回值规则仍未全部覆盖。
- 当前回归测试已覆盖多返回值在返回列表、赋值、空 `vararg`、`vararg` 赋值、函数实参列表、`vararg` 实参列表、方法调用、`pcall` 成功/失败路径，以及 table 构造器最后数组字段中的常见调整规则；更完整的 Lua 多返回值边界仍未全部覆盖。
- 当前已支持 `{}`、键值字段、`t[k]`、`t.name` 的最小读写，以及基础 `setmetatable` / `getmetatable`、`__metatable` 保护、`__index` / `__newindex`、`__tostring`、`__call`、`rawget`、`rawset`、`rawequal`、`__add`、`__sub`、`__mul`、`__div`、`__mod`、`__pow`、`__unm`、`__concat`、`__eq`、`__lt`、`__le`；完整 table 行为仍未实现。
- 当前已支持 `local function`、匿名函数表达式和基础 upvalue 读写；闭包仍未覆盖完整 Lua 5.1 upvalue 语义。
- 当前 generic `for` 主要面向 `pairs` / `ipairs` / `next` 这一最小可用链路；更完整的迭代器兼容性仍待补齐。

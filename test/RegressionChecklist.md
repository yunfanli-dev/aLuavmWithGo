# Regression Checklist

本清单用于把当前 Lua 5.1 子集 VM 的关键能力，与对应的自动化测试和手工样例验证方式对齐。

## 自动化主入口

- `go test ./...`

## 手工样例主入口

- `go run ./cmd/aluavm ./examples/runtime_showcase.lua`
- `go run ./cmd/aluavm ./examples/multivalue_showcase.lua`

## 检查项

- 源码加载与前端链路
  对应：`go test ./...`
  关注：lexer、parser、IR 编译是否仍能覆盖当前 Lua 5.1 子集主链路。

- 基础表达式、赋值、条件与循环
  对应：`go test ./...`
  关注：算术、布尔逻辑、`if`、`while`、`repeat-until`、数值 `for`、generic `for` 是否仍可执行。

- 函数、闭包、方法调用与 `vararg`
  对应：`go test ./...`
  关注：命名函数、`local function`、匿名函数、upvalue、方法定义/调用、`...` 作用域和基础多返回值规则。

- table 与 metatable
  对应：`go test ./...`
  对应：`go run ./cmd/aluavm ./examples/runtime_showcase.lua`
  关注：table 构造、索引、`table.sort`、`__index`、`__tostring`、raw 接口和基础元方法。
  通过标准：样例输出中应看到 `sort	1,2,3`、`meta_index	from_meta`、`meta_tostring	object:40`。

- 标准库子集
  对应：`go test ./...`
  对应：`go run ./cmd/aluavm ./examples/runtime_showcase.lua`
  关注：`print`、`type`、`tostring`、`tonumber`、`pcall` / `xpcall`、`table.*`、`math.*`、`string.*` 当前已支持的子集。
  通过标准：样例输出中应看到 `string	desserts	LUA` 和 `math	9	7`。

- 多返回值与错误处理
  对应：`go test ./...`
  对应：`go run ./cmd/aluavm ./examples/multivalue_showcase.lua`
  关注：赋值调整、`return`、函数实参、table 构造器最后字段展开、圆括号抑制展开、`pcall` 错误返回。
  通过标准：样例输出中应看到 `assign	1	left	right`、`pcall	false	boom`、`table	head|left|right`、`byte	65	90`。

- CLI 与文件执行入口
  对应：`go run ./cmd/aluavm ./examples/runtime_showcase.lua`
  对应：`go run ./cmd/aluavm ./examples/multivalue_showcase.lua`
  关注：本地文件加载、脚本执行、标准输出、CLI 收尾输出。
  通过标准：两份样例最后都应输出 `aluavm bootstrap ready`。

## 使用建议

- 每次较大的运行时修改后，先跑 `go test ./...`。
- 如果改动影响了多返回值、标准库或 metatable，再补跑两份 `examples/*.lua` 样例。
- 如果新增能力没有被本清单覆盖，应同步补充本文件、`test/README.md` 和必要样例。

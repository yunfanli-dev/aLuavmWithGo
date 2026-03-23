# Examples

本目录提供当前 Lua 5.1 子集运行时的手工回归样例。

## 运行方式

- `go run ./cmd/aluavm ./examples/runtime_showcase.lua`
- `go run ./cmd/aluavm ./examples/multivalue_showcase.lua`

## 样例说明

- `runtime_showcase.lua`：覆盖函数、table、排序、metatable、字符串和数学库的基础链路。
- `multivalue_showcase.lua`：覆盖 `vararg`、多返回值、`pcall`、table 构造器展开和括号抑制展开。

## 预期输出要点

- `runtime_showcase.lua`
  关键输出应包含：
  `sort	1,2,3`
  `method	42`
  `meta_index	from_meta`
  `meta_tostring	object:40`
  `string	desserts	LUA`
  `math	9	7`
  最后一行应为：`aluavm bootstrap ready`

- `multivalue_showcase.lua`
  关键输出应包含：
  `assign	1	left	right`
  `paren	left	left`
  `pcall	false	boom`
  `table	head|left|right`
  `byte	65	90`
  最后一行应为：`aluavm bootstrap ready`

## 备注

- 这两份样例都以 `print` 输出人工可检查的结果，便于在不看单元测试的情况下快速验证主链路。
- 当前 CLI 仍会在脚本执行后输出 `aluavm bootstrap ready`，这是预期行为。

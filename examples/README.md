# Examples

本目录提供当前 Lua 5.1 子集运行时的手工回归样例。

## 运行方式

- `go run ./cmd/aluavm ./examples/runtime_showcase.lua`
- `go run ./cmd/aluavm ./examples/multivalue_showcase.lua`
- `go run ./cmd/aluavm ./examples/table_metatable_showcase.lua`
- `go run ./cmd/aluavm ./examples/generic_for_showcase.lua`

## 样例说明

- `runtime_showcase.lua`：覆盖函数、table、排序、metatable、字符串和数学库的基础链路。
- `multivalue_showcase.lua`：覆盖 `vararg`、多返回值、`pcall`、table 构造器展开和括号抑制展开。
- `table_metatable_showcase.lua`：覆盖 table / function 对象 key、`rawget` / `rawset`，以及 `__index` / `__newindex` 的 table 回退链路。
- `generic_for_showcase.lua`：覆盖自定义 iterator 三元组和 `string.gmatch` 接入 generic `for` 的链路。

## 预期输出要点

- `runtime_showcase.lua`
  关键输出应包含：
  `sort	1,2,3`
  `method	42`
  `meta_index	from_meta`
  `meta_tostring	object:40`
  `string	desserts	LUA`
  `math	9	7`

- `multivalue_showcase.lua`
  关键输出应包含：
  `assign	1	left	right`
  `paren	left	left`
  `pcall	false	boom`
  `table	head|left|right`
  `byte	65	90`

- `table_metatable_showcase.lua`
  关键输出应包含：
  `identity	table-a	table-b	fn-a	fn-b`
  `chain	from-chain	42`

- `generic_for_showcase.lua`
  关键输出应包含：
  `custom_iter	66`
  `gmatch_iter	a|a`

## 备注

- 这四份样例都以 `print` 输出人工可检查的结果，便于在不看单元测试的情况下快速验证主链路。
- 这四份样例当前也已接入 `cmd/aluavm` 的 CLI 进程级集成测试，关键输出会被自动回归校验。
- 当前 CLI 在执行脚本成功后保持安静，不再额外输出 `aluavm bootstrap ready`。

# Test README

## 当前可重复验证方式

- 运行 `go test ./...` 验证当前 Lua 5.1 子集运行时是否可编译并通过单元测试。
- 运行 `go run ./cmd/aluavm` 验证最小 demo 入口是否可启动。
- 运行 `go run ./cmd/aluavm <script.lua>` 验证文件加载、前端编译与当前运行时链路是否可执行。

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

## 备注

- 当前 lexer 已支持基础关键字、标识符、数字、字符串、短注释和常用运算符；长注释、长字符串、十六进制和指数数字仍待补齐。
- 当前 parser 已支持 `local`、赋值、`return`、`if`、`while`、`repeat-until`、数值 `for`、generic `for`、函数声明、table 构造、匿名函数与基础表达式；`break`、`do ... end`、泛型语法的更完整子集仍待补齐。
- 当前已支持 `local`、赋值、`return`、`if`、`while`、`repeat-until`、数值 `for`、generic `for`、函数调用、table 读写、闭包基础能力和基础一元/二元表达式的执行；metatable、完整多返回值语义和更多 Lua 5.1 细节仍待补齐。
- 当前已支持 Go 宿主向 Lua 注册基础函数，并内置最小 `print`；标准库仍远未完整。
- 当前已内置 `print`、`type`、`tostring`、`tonumber`、`assert`、`error`、`pcall`、`next`、`pairs`、`ipairs`；这仍只是较小的基础内建子集。
- 当前已支持 `error`、`pcall`，并支持最后一个函数调用在返回列表中的多返回值展开；更完整的 Lua 多返回值规则仍未全部覆盖。
- 当前已支持 `{}`、键值字段、`t[k]`、`t.name` 的最小读写；数组长度语义、metatable 和完整 table 行为仍未实现。
- 当前已支持 `local function`、匿名函数表达式和基础 upvalue 读写；闭包仍未覆盖完整 Lua 5.1 upvalue 语义。
- 当前 generic `for` 主要面向 `pairs` / `ipairs` / `next` 这一最小可用链路；更完整的迭代器兼容性仍待补齐。

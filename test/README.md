# Test README

## 当前可重复验证方式

- 运行 `go test ./...` 验证当前 M1 骨架是否可编译并通过基础单元测试。
- 运行 `go run ./cmd/aluavm` 验证最小 demo 入口是否可启动。
- 运行 `go run ./cmd/aluavm <script.lua>` 验证文件加载入口是否已接入当前 bootstrap 流程。

## 当前覆盖范围

- VM 实例创建
- 基础状态初始化
- 空 Lua 源码的 bootstrap 执行入口
- Lua 脚本文件读取与统一 source 入口
- Lua 5.1 子集的基础词法切分
- Lua 5.1 子集的基础 AST 解析
- Lua 5.1 子集的基础 IR 编译链路
- Lua 5.1 子集的最小 IR 执行能力

## 备注

- 非空 Lua 源码执行仍是占位实现，后续会在加载链路和执行核心完成后补齐对应测试。
- 当前 lexer 已支持基础关键字、标识符、数字、字符串、短注释和常用运算符；长注释、长字符串、十六进制和指数数字仍待补齐。
- 当前 parser 已支持 chunk、`local`、`return`、标识符、字面量、基础一元/二元表达式；`if`、`while`、函数声明、table 构造等仍待补齐。
- 当前已支持 `local`、`return`、标识符、字面量、基础一元/二元表达式的执行；`if`、`while`、函数声明、table 构造、函数调用等仍待补齐。

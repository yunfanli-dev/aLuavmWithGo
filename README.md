# aLuavmWithGo

`aLuavmWithGo` 是一个使用 Go 实现的 Lua 虚拟机项目。

项目目标是提供一套可嵌入 Go 工程的 Lua 运行时能力，使 Go 应用能够加载 Lua 代码、执行 Lua 脚本，并逐步支持以 `Lua 5.1` 为目标基线的运行时特性与宿主交互能力。

## 项目目标

- 使用 Go 实现一个可维护的 Lua 5.1 子集虚拟机。
- 支持加载并执行 Lua 代码。
- 为后续将 Lua 嵌入 Go 项目提供运行时基础。
- 逐步补齐 Lua 5.1 子集所需的执行模型、函数调用、基础标准库与测试验证。

## 规划范围

当前项目会按阶段推进：

- 先搭建 Go 项目骨架与最小执行入口。
- 再实现 Lua 代码加载链路。
- 然后补齐虚拟机执行核心，如栈、调用帧与指令分发。
- 最后逐步扩展函数、table、标准库子集与测试。

## 预期用途

该项目面向需要在 Go 项目中嵌入脚本能力的场景，例如：

- 配置驱动逻辑扩展
- 可热更新的业务规则
- 轻量脚本插件系统
- 宿主程序与 Lua 脚本协同执行

## 当前状态

当前项目已不再停留在骨架阶段，而是已经具备一个可运行的 Lua 5.1 子集执行链路，包括：

- Lua 源码加载、lexer、parser、IR 编译与执行
- 基础表达式、局部变量、条件与循环
- 基础表达式、局部变量、条件、循环、块控制、基础 `vararg` 与方法语法
- 命名函数、`local function`、匿名函数与基础 upvalue
- 最小 table 读写
- 最小 metatable 读写、保护、调用、算术与比较元方法能力
- Go 宿主函数注册
- 基础内建函数、最小 `_G` / `module(...)` / `require` / `package` / `package.preload` / `package.loaders` / `package.searchpath` 模块加载与最小 generic `for` 迭代能力
- 最小执行步数限制能力，可用于阻止明显的死循环脚本长期占用执行线程
- 基于 `context.Context` 的最小宿主取消入口
- Go 宿主侧最小 preload 模块注册入口
- Go 宿主侧最小自定义 module searcher 注册入口
- Go 宿主侧最小直接 loaded-module 注入入口
- CLI 级最小内联执行、超时控制、步数预算、help/usage、基础错误分类输出、最小成功输出 / 退出码约定，以及进程级集成测试覆盖

补充说明：

- `SetStepLimit(...)` 默认未开启
- 只有宿主显式设置正数预算后，执行步数限制才会生效
- `limit <= 0` 会按“不限制”处理

当前仍在持续补齐 Lua 5.1 子集的剩余差距，例如更完整的 metatable 语义、多返回值规则、更多标准库与更完整语法。

## 样例脚本

仓库当前提供可直接运行的手工验证样例，见 [examples/README.md](./examples/README.md)。

- `examples/runtime_showcase.lua`：覆盖运行时和标准库主链路
- `examples/multivalue_showcase.lua`：覆盖多返回值和 `pcall` 链路

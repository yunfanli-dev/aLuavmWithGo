# Execution Safety Plan

本文件用于整理当前 Lua 5.1 子集 VM 在“脚本执行安全”上的主要风险、候选方案和推荐推进路径。

## 当前问题

当前 VM 已可以执行循环、递归、闭包和宿主注册函数，但还没有真正的脚本级执行中断能力。

现状：

- 没有 step limit
- 没有 instruction budget
- 没有 `context.Context` 取消链路
- 没有超时 watchdog
- `go test` 只能从测试框架层面兜底超时，不能保护 CLI 或宿主嵌入场景

直接影响：

- `while true do end` 这类脚本理论上可能长期占用执行线程
- 宿主如果直接调用 `ExecString` / `ExecFile`，当前没有内建的执行预算保护

## 目标

目标不是一次性做完整沙箱，而是先提供一个最小、可测试、可嵌入的执行预算机制。

首要目标：

- 能限制脚本执行步数
- 能在预算耗尽时返回明确错误
- 单元测试可以稳定验证该行为
- 后续可以平滑扩展到超时和宿主取消

## 候选方案

### 方案 A：Step Limit

思路：

- 在执行语句、表达式、函数调用或循环迭代时递减一个计数器
- 计数器耗尽后立即返回错误

优点：

- 实现最直接
- 对当前 IR/执行器结构侵入较小
- 容易写单元测试

缺点：

- “一步”定义不够精确
- 不同脚本形态的预算消耗不完全均衡

结论：

- 最适合作为当前项目的第一阶段执行安全方案

### 方案 B：Instruction Budget

思路：

- 把预算扣减绑定到更接近 IR/指令层的执行节点

优点：

- 语义更稳定
- 后续更适合演进成真正的 VM 执行预算

缺点：

- 当前项目还不是字节码型 VM，落地成本比 step limit 更高

结论：

- 适合作为 step limit 之后的演进方向

### 方案 C：Context Cancellation

思路：

- 为 `ExecString` / `ExecSource` / `ExecFile` 增加上下文参数
- 执行过程中定期检查 `ctx.Done()`

优点：

- 对宿主嵌入场景最友好
- 可以和 HTTP 请求、任务调度、外部超时控制直接集成

缺点：

- 接口会变化
- 仍然需要“检查点”机制，通常还是要和 step limit 或循环检查配合

结论：

- 适合作为第二阶段，把宿主控制能力补齐

## 推荐推进路径

### 第一阶段

- 为 `State` 增加最小执行预算配置
- 在执行器主路径上增加统一 `consumeStep()` 检查
- 预算耗尽时返回明确错误，例如：`execution step limit exceeded`
- 增加死循环/深循环的单元测试

### 第二阶段

- 增加可选 `context.Context` 入口
- 在循环和函数调用边界增加取消检查

### 第三阶段

- 如果后续 IR/VM 结构继续下沉，再把 step limit 平滑演进为更稳定的 instruction budget

## 推荐的最小 API 方向

可考虑新增：

- `SetStepLimit(limit int)`
- `ExecSourceWithContext(ctx context.Context, source Source)`

其中：

- `limit <= 0` 可表示不限制，保持当前兼容行为
- 默认值建议先保持“不限制”，避免悄悄改变现有执行语义

## 测试建议

- 新增一个无限循环或大循环脚本，用受限 step budget 验证能否正确中断
- 验证正常短脚本在有限预算内仍可通过
- 验证预算耗尽时返回的错误信息稳定可断言
- 如果后续接入 `context.Context`，再补取消路径测试

## 当前结论

当前最值得优先落地的是：

1. 先做 step limit
2. 再做 context cancellation
3. 最后视执行模型演进再考虑更严格的 instruction budget

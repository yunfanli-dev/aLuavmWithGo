# Gap Checklist

本清单用于记录当前项目距离“更完整的 Lua 5.1 子集 + 更稳定的验证方式”还剩哪些明显空白。

## 语法与语义空白

- Lua 5.1 全量多返回值边界语义仍未完全收平
  说明：当前已覆盖高频场景，但赋值、参数列表和复杂嵌套表达式的全部角落规则仍未系统对齐。
  参考：[Lua51SyntaxSupport.md](../Lua51SyntaxSupport.md)

- `#` 对 table 的完整长度语义仍未实现
  说明：当前已支持字符串长度，也已支持“存在索引 `1` 时按当前表中的最大正整数整数 key 取边界长度”的最小 table 长度行为；但更完整的 Lua 5.1 table 长度语义仍未补齐。
  影响：部分依赖 table 长度的 Lua 5.1 代码仍不能按标准行为运行。

- generic `for` 仍主要围绕 `next` / `pairs` / `ipairs`
  说明：更广泛的迭代器兼容性和边界行为还没有系统验证。

## 运行时与标准库空白

- `math` 库仍只覆盖较小子集
  说明：当前已支持 `pi` / `huge`、基础取整、`modf`、`mod` / `fmod`、`frexp` / `ldexp`、角度/弧度转换、极值、随机、指数、自然/十进制对数，以及基础三角/反三角函数、`atan2`、`sinh`、`cosh` 和 `tanh`，但离完整 Lua 5.1 数学库仍有距离。

- `string` 库仍只覆盖较小子集
  说明：当前已支持长度、纯文本 `find` / `match` / `gfind` / `gmatch` / `gsub`、最小 `format`、截取、大小写、重复、反转、字节提取和按字节组装；其中 `gsub` 已支持字符串 / table / function 替换器，`format` 已支持 `%c`、`%o`、`%u`、`%x`、`%X`、`%e`、`%E`、`%g`、`%G` 等少量高频格式符，但仍缺少 Lua pattern 版 `find` / `match` / `gfind` / `gmatch` / `gsub`、capture 语义以及更完整格式化语义。

- `table` 库仍只覆盖较小子集
  说明：当前已支持 `getn`、`maxn`、`foreach`、`foreachi`、`insert`、`remove`、`concat`、`sort`，其中 `concat` 已可复用现有字符串化逻辑处理带 `__tostring` 的值，但仍未补齐更完整的序列表辅助能力。

- upvalue、闭包和 metatable 仍未完全对齐 Lua 5.1
  说明：当前已具备最小可用链路，也已补上 `__index` / `__newindex` / `__call` 的明显链式环基础报错路径，并把 `__eq`、`__lt`、`<=` / `>=` 的最小比较规则收口到更接近 Lua 5.1 的形态，但更多边界行为和完整兼容性仍未系统补齐。

## 执行安全空白

- 当前 VM 已有最小 `step limit` 和 `context cancellation`，但仍没有更严格的 instruction budget
  说明：宿主现在可以配置基础执行预算，也可以主动取消执行，但预算精度和 CLI 级控制仍有继续收口空间。
  影响：执行安全能力已具备前两阶段最小保护，但仍未达到更严格、更细粒度的预算模型。
  参考：[ExecutionSafetyPlan.md](./ExecutionSafetyPlan.md)

## 验证与整理空白

- 手工样例仍较少
  说明：当前已覆盖运行时主链路、多返回值主链路，以及 table / metatable、generic `for` 两份专项样例，但对更多标准库和 metatable 组合场景还不够系统。

- 回归检查清单已建立，但还不是完整矩阵
  说明：当前主要按关键能力点和关键输出片段判定，尚未细化到逐能力的更完整通过标准。

- 部分剩余空白还没有单独的专题文档
  说明：例如执行安全策略、未支持特性分级、未来收口策略还没有独立整理。

## 使用方式

- 每次决定下一阶段工作时，先看本文件，再结合 [RegressionChecklist.md](./RegressionChecklist.md) 判断是“补能力”还是“补验证”。
- 如果某个 gap 被解决，应同步更新本文件、相关功能文档和 `ProgressLog.md`。

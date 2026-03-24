# Lua 5.1 Syntax Support

本文件只记录当前项目对 `Lua 5.1` 语法层面的支持情况。

范围说明：

- 这里只描述“语法是否可被当前前端与执行链路接受”
- 不展开标准库、宿主 API、metatable 内建函数等非语法能力
- 如果某种语法“可解析但语义仍有限”，会单独标注

## 已支持语法

### 语句

- chunk 顶层语句序列
- 语句分隔符 `;`
- 赋值：`a = 1`、`a, b = 1, 2`
- 局部赋值：`local a = 1`、`local a, b = 1, 2`
- 函数声明：`function name(...) ... end`
- 方法定义：`function t:name(...) ... end`
- 局部函数声明：`local function name(...) ... end`
- 函数调用语句：`fn(...)`
- 方法调用语句：`obj:method(...)`
- 块语句：`do ... end`
- `break`
- 条件分支：`if ... then ... elseif ... then ... else ... end`
- `while ... do ... end`
- `repeat ... until ...`
- 数值 `for`：`for i = start, limit[, step] do ... end`
- generic `for`：`for k, v in exprlist do ... end`
- `return`

### 表达式

- 标识符
- `nil`
- `true` / `false`
- 十进制数字字面量
- 指数形式数字字面量：`1e3`、`2.5e-2`
- 十六进制数字字面量：`0xff`、`0X10`
- 短字符串字面量
- 长字符串字面量：`[[ ... ]]`、`[=[ ... ]=]`
- 圆括号表达式：`(expr)`
  - 当前会保留 Lua 5.1 的单值语义，例如 `(fn())` 不再继续展开多返回值
- 匿名函数表达式：`function(...) ... end`
- `vararg` 表达式：`...`
- table 构造器：
  - 数组式字段：`{1, 2, 3}`
  - 命名字段：`{ answer = 42 }`
  - 索引字段：`{ [key] = value }`
  - 字段分隔符：`,` 和 `;`
  - 最后一个数组式字段支持函数调用或 `...` 的多返回值展开
- 索引表达式：`t[k]`
- 点语法字段访问：`t.name`
- 调用表达式：`fn(...)`
- table call：`fn{ ... }`
- string call：`fn"literal"`
- 方法调用表达式：`obj:method(...)`
- 方法 table call：`obj:method{ ... }`
- 方法 string call：`obj:method"literal"`

### 运算符

- 一元运算：`-`、`not`、`#`
- 二元算术：`+`、`-`、`*`、`/`、`%`、`^`
- 字符串拼接：`..`
- 比较：`<`、`<=`、`>`、`>=`、`==`、`~=`
- 逻辑：`and`、`or`
- 当前已处理基础优先级和结合性

## 未支持语法

### 语句

- `goto`
- 标签：`::label::`
- `repeat` 之外的 `until` 独立误用校验增强

### 参数与返回

- Lua 5.1 全量多返回值边界语义
  - 当前尚未系统覆盖所有赋值、参数列表和复杂嵌套表达式的边界调整规则

### table 构造与访问

- 更完整的 table 构造器边界兼容性校验
- 更完整的 table 长度相关语义

## 语义限制

- `#` 和 `table.getn` 当前已支持字符串或从索引 `1` 开始的连续数组段长度；`table.maxn` 当前返回表中的最大数值 key；这些能力仍不代表 Lua 5.1 完整 table 长度语义
- `string.find` / `string.match` / `string.gmatch` / `string.gsub` / `string.format` 当前都只覆盖最小可用子集；其中 `find` / `match` / `gmatch` 支持 Lua 风格起始下标下的纯文本匹配，`gsub` 支持纯文本匹配下的字符串 / table / function 替换器与可选替换次数，`format` 只支持少量高频格式符，不支持 Lua 5.1 完整 pattern / capture / replacer / format 语义
- `...` 当前已支持函数参数与表达式展开，并会拒绝非法作用域中的使用，但仍未覆盖 Lua 5.1 全部边界行为
- 虽然 generic `for` 语法已支持，但当前主要围绕 `next` / `pairs` / `ipairs` 这一最小链路使用

## 维护规则

- 每次修改语法、前端、执行器或会影响 Lua 代码可写法的行为后，都要同步检查本文件
- 如果支持范围发生变化，必须在“已支持语法 / 未支持语法 / 语义限制”中更新对应条目

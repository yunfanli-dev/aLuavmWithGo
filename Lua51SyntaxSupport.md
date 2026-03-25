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

- `#` 和 `table.getn` 当前已支持字符串长度，以及“存在索引 `1` 时按当前表中的最大正整数整数 key 取边界长度”的最小 table 长度语义；`table.maxn` 当前返回表中的最大数值 key；这些能力仍不代表 Lua 5.1 完整 table 长度语义
- table / function 当前已按对象身份参与 table key 匹配，因此 `t[other_table]`、`t[fn]`、`rawget` / `rawset` 不会再因相同调试文本而意外撞 key；但更完整的 Lua 5.1 table 行为仍未全部收平
- `string.find` / `string.match` / `string.gfind` / `string.gmatch` / `string.gsub` / `string.format` 当前都只覆盖最小可用子集；其中 `find` / `match` / `gfind` / `gmatch` 支持 Lua 风格起始下标下的纯文本匹配，`gsub` 支持纯文本匹配下的字符串 / table / function 替换器与可选替换次数，`format` 支持 `%c` / `%o` / `%u` / `%x` / `%X` / `%e` / `%E` / `%g` / `%G` 在内的少量高频格式符，不支持 Lua 5.1 完整 pattern / capture / replacer / format 语义
- `require` / `package` / `module(...)` 当前已支持最小文件模块加载、`package.loaded` 缓存、`package.preload` 内存 loader、`package.path` 搜索模板、`package.loaders` 顺序搜索、最小 `package.searchpath`、`package.seeall`、最小 `module(...)` 表注册和循环检测，会按 loader 顺序解析模块，并优先相对当前源码目录解析相对模板；其中 `package.loaded[name] = false` 当前不会被视为“已加载命中”，而会继续重新加载；`module(...)` 当前会把调用它的 Lua 帧切到模块环境，并把点分模块路径挂到调用点当前可见环境上；`package.seeall` 和 `require` 执行链也会继续沿用调用点当前可见环境，且后置 `package.seeall(_M)` 不会再错误地回退到模块表自身；当前 `seeall` 的内部基环境记录也已隐藏，不再污染 Lua 可见 metatable 字段；但仍不覆盖完整 Lua 5.1 环境切换全量语义
- `__index` / `__newindex` 当前已支持 table / function 两种最小回退形态，并会拒绝明显的链式自引用或环；但更完整的 Lua 5.1 元方法链式语义仍未全部收平
- `__call` 当前已支持最小 table callable 语义，并会拒绝明显的链式自引用或环；但更完整的 Lua 5.1 callable / 元方法兼容性仍未全部收平
- `__eq` 当前要求两侧 table 共享同一个元方法值才会触发，这和 Lua 5.1 的基础规则一致；但更完整的元方法兼容性仍未全部收平
- `<` / `>` 当前已要求两侧 table 共享同一个 `__lt` 元方法值才会触发，这和 Lua 5.1 的基础规则更接近；但更完整的比较元方法兼容性仍未全部收平
- `<=` / `>=` 当前已要求两侧 table 共享同一个 `__le` 元方法值才会触发，并会在 `__le` 缺失时通过共享 `__lt` 的反向比较做最小回退；但更完整的 Lua 5.1 比较元方法兼容性仍未全部收平
- 算术、一元负号、数值 `for` 和当前复用数值 helper 的 builtin 现在会对可解析数字字符串做最小强转；但关系比较仍保留“字符串只能和字符串直接比较”的基础规则，更完整的 Lua 5.1 数值兼容性仍未全部收平
- `..` 当前已把原生快速路径收口到“双方都必须是字符串或数字”，否则会继续尝试 `__concat` 或直接报错；但更完整的 Lua 5.1 拼接兼容性仍未全部收平
- 同类型字符串当前已支持直接参与 `<` / `<=` / `>` / `>=` 的字典序比较；更完整的跨类型比较兼容性仍未实现
- `...` 当前已支持函数参数与表达式展开，并会拒绝非法作用域中的使用，但仍未覆盖 Lua 5.1 全部边界行为
- 未声明名称的普通赋值当前会回落到当前函数绑定的最小环境表；默认环境仍是 `_G`，因此 `name`、`_G.name` 和 `rawget/rawset(_G, ...)` 这几条基础访问路径可以互通；同时当前已支持最小 `getfenv` / `setfenv`，可改写函数值、当前活跃调用栈环境以及 `getfenv(0)` / `setfenv(0, ...)` 线程级环境；栈级 `setfenv(level, env)` 现在会同步改回对应函数对象本身，`setfenv(fn, env)` 命中当前正在执行的函数时也会立即刷新活跃调用帧，未显式绑环境的 native 函数当前也会按调用者线程环境返回 `getfenv(fn)`；但完整 Lua 5.1 调试栈级别环境切换语义仍未实现
- 虽然 generic `for` 语法已支持，且最后一个迭代表达式已覆盖普通调用、`...`、builtin 与 protected call 的最小多返回值调整，但整体仍主要围绕 `next` / `pairs` / `ipairs` 这一最小链路使用

## 维护规则

- 每次修改语法、前端、执行器或会影响 Lua 代码可写法的行为后，都要同步检查本文件
- 如果支持范围发生变化，必须在“已支持语法 / 未支持语法 / 语义限制”中更新对应条目

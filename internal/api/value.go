package api

import "github.com/yunfanli-dev/aLuavmWithGo/internal/vm"

// ValueType 是公开 API 层直接复用的运行时值类型枚举。
// 这里做类型别名而不是重新定义，避免 API 层和 VM 内部出现不必要的转换。
type ValueType = vm.ValueType

// Value 是公开 API 层暴露给宿主的统一运行时值容器。
// 它直接复用内部 VM 的表示形式，便于宿主函数和运行时之间传递参数与返回值。
type Value = vm.Value

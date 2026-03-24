package vm

import "fmt"

type table struct {
	entries   map[string]Value
	keys      map[string]Value
	order     []string
	metatable *table
}

// newTable 创建当前 Lua table 子集使用的底层存储结构。
// 它会初始化值映射、原始 key 映射和插入顺序记录。
func newTable() *table {
	return &table{
		entries: make(map[string]Value),
		keys:    make(map[string]Value),
	}
}

// get 根据归一化后的运行时 key 读取一个 table 字段。
// 如果 key 不存在，会返回 `exists=false`，而不是把缺失值偷偷转成 nil。
func (t *table) get(key Value) (Value, bool, error) {
	if t == nil {
		return NilValue(), false, fmt.Errorf("index nil table")
	}

	storageKey, err := tableKey(key)
	if err != nil {
		return NilValue(), false, err
	}

	value, ok := t.entries[storageKey]
	return value, ok, nil
}

// set 根据归一化后的运行时 key 写入一个 table 字段。
// 当 value 为 nil 类型时会删除该 key；否则会更新值并维护插入顺序。
func (t *table) set(key Value, value Value) error {
	if t == nil {
		return fmt.Errorf("assign into nil table")
	}

	storageKey, err := tableKey(key)
	if err != nil {
		return err
	}

	if value.Type == ValueTypeNil {
		t.deleteKey(storageKey)
		return nil
	}

	if _, exists := t.entries[storageKey]; !exists {
		t.order = append(t.order, storageKey)
	}

	t.entries[storageKey] = value
	t.keys[storageKey] = key
	return nil
}

// getMetatable 返回当前 table 已绑定的 metatable。
// 如果还没有设置 metatable，则返回 nil。
func (t *table) getMetatable() *table {
	if t == nil {
		return nil
	}

	return t.metatable
}

// getProtectedMetatable 返回对外暴露的受保护 metatable 视图。
// 当 metatable 设置了 `__metatable` 时，`getmetatable` 应该看到的是这个保护值。
func (t *table) getProtectedMetatable() (Value, bool, error) {
	if t == nil || t.metatable == nil {
		return NilValue(), false, nil
	}

	return t.metatable.get(Value{Type: ValueTypeString, Data: "__metatable"})
}

// setMetatable 替换当前 table 绑定的 metatable。
// 这里不做保护判断，保护逻辑由更高层 builtin 控制。
func (t *table) setMetatable(metatable *table) {
	if t == nil {
		return
	}

	t.metatable = metatable
}

// next 按当前实现使用的插入顺序返回给定 key 之后的下一个键值对。
// 这为 `next` / `pairs` 和 generic for 提供最小可用迭代基础。
func (t *table) next(key Value) (Value, Value, bool, error) {
	if t == nil {
		return NilValue(), NilValue(), false, fmt.Errorf("iterate nil table")
	}

	if key.Type == ValueTypeNil {
		return t.firstEntry()
	}

	storageKey, err := tableKey(key)
	if err != nil {
		return NilValue(), NilValue(), false, err
	}

	for index, current := range t.order {
		if current != storageKey {
			continue
		}

		if index+1 >= len(t.order) {
			return NilValue(), NilValue(), false, nil
		}

		nextStorageKey := t.order[index+1]
		return t.keys[nextStorageKey], t.entries[nextStorageKey], true, nil
	}

	return NilValue(), NilValue(), false, fmt.Errorf("invalid key to 'next'")
}

// firstEntry 返回当前插入顺序中的第一个键值对。
// 当 table 为空时，会返回 `ok=false`。
func (t *table) firstEntry() (Value, Value, bool, error) {
	if len(t.order) == 0 {
		return NilValue(), NilValue(), false, nil
	}

	first := t.order[0]
	return t.keys[first], t.entries[first], true, nil
}

// sequenceLength 返回从索引 1 开始的连续数组段长度。
// 这对应当前实现里 `#table` 和 `table.getn` 共享的最小 sequence 语义。
func (t *table) sequenceLength() (int, error) {
	if t == nil {
		return 0, fmt.Errorf("measure nil table")
	}

	end, err := tableSequenceEnd(t, 1)
	if err != nil {
		return 0, err
	}

	return end, nil
}

// maxNumericKey 返回当前 table 中存在的最大数值 key。
// 这对应 `table.maxn` 的最小实现语义，与连续数组段长度不是同一个概念。
func (t *table) maxNumericKey() (float64, error) {
	if t == nil {
		return 0, fmt.Errorf("measure nil table")
	}

	maximum := float64(0)
	for _, key := range t.keys {
		if key.Type != ValueTypeNumber {
			continue
		}

		number, ok := key.Data.(float64)
		if !ok {
			return 0, fmt.Errorf("invalid numeric table key payload %T", key.Data)
		}

		if number > maximum {
			maximum = number
		}
	}

	return maximum, nil
}

// deleteKey 从 table 底层存储中移除一个已经归一化的 key。
// 它会同时清理值映射、原始 key 映射以及插入顺序记录。
func (t *table) deleteKey(storageKey string) {
	delete(t.entries, storageKey)
	delete(t.keys, storageKey)

	for index, current := range t.order {
		if current != storageKey {
			continue
		}

		t.order = append(t.order[:index], t.order[index+1:]...)
		return
	}
}

func tableKey(key Value) (string, error) {
	if key.Type == ValueTypeNil {
		return "", fmt.Errorf("table key cannot be nil")
	}

	return fmt.Sprintf("%s:%s", key.Type, valueToString(key)), nil
}

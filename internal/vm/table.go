package vm

import "fmt"

type table struct {
	entries   map[string]Value
	keys      map[string]Value
	order     []string
	metatable *table
}

// newTable creates the runtime storage used by the current Lua table subset.
func newTable() *table {
	return &table{
		entries: make(map[string]Value),
		keys:    make(map[string]Value),
	}
}

// get reads one table field by its normalized runtime key.
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

// set stores or removes one table field by its normalized runtime key.
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

// getMetatable returns the currently attached metatable, if any.
func (t *table) getMetatable() *table {
	if t == nil {
		return nil
	}

	return t.metatable
}

// getProtectedMetatable returns the protected metatable view exposed by getmetatable, if configured.
func (t *table) getProtectedMetatable() (Value, bool, error) {
	if t == nil || t.metatable == nil {
		return NilValue(), false, nil
	}

	return t.metatable.get(Value{Type: ValueTypeString, Data: "__metatable"})
}

// setMetatable replaces the current table metatable.
func (t *table) setMetatable(metatable *table) {
	if t == nil {
		return
	}

	t.metatable = metatable
}

// next returns the next key/value pair after the provided key using insertion order.
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

// firstEntry returns the first key/value pair in iteration order.
func (t *table) firstEntry() (Value, Value, bool, error) {
	if len(t.order) == 0 {
		return NilValue(), NilValue(), false, nil
	}

	first := t.order[0]
	return t.keys[first], t.entries[first], true, nil
}

// deleteKey removes one normalized key from the table storage.
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

package vm

import "fmt"

type table struct {
	entries map[string]Value
}

func newTable() *table {
	return &table{
		entries: make(map[string]Value),
	}
}

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

func (t *table) set(key Value, value Value) error {
	if t == nil {
		return fmt.Errorf("assign into nil table")
	}

	storageKey, err := tableKey(key)
	if err != nil {
		return err
	}

	t.entries[storageKey] = value
	return nil
}

func tableKey(key Value) (string, error) {
	if key.Type == ValueTypeNil {
		return "", fmt.Errorf("table key cannot be nil")
	}

	return fmt.Sprintf("%s:%s", key.Type, valueToString(key)), nil
}

package executor

import (
	"bytes"
	"encoding/json"
)

type OrderedMap struct {
	m     map[string]interface{}
	order []string
}

func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		m: map[string]interface{}{},
	}
}

func (m *OrderedMap) Set(key string, value interface{}) {
	if _, ok := m.m[key]; !ok {
		m.order = append(m.order, key)
	}
	m.m[key] = value
}

func (m *OrderedMap) Get(key string) (interface{}, bool) {
	v, ok := m.m[key]
	return v, ok
}

func (m *OrderedMap) Len() int {
	return len(m.m)
}

func (m *OrderedMap) Keys() []string {
	return m.order
}

func (m *OrderedMap) MarshalJSON() ([]byte, error) {
	pairs := make([][]byte, len(m.order))
	for i, key := range m.order {
		keyJSON, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		valueJSON, err := json.Marshal(m.m[key])
		if err != nil {
			return nil, err
		}
		pairs[i] = bytes.Join([][]byte{keyJSON, valueJSON}, []byte{':'})
	}
	return append(append([]byte{'{'}, bytes.Join(pairs, []byte{','})...), '}'), nil
}

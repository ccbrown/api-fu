package executor

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"
)

type OrderedMapItem struct {
	Key   string
	Value interface{}
}

type OrderedMap struct {
	m     map[string]int
	items []OrderedMapItem
}

func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		m: map[string]int{},
	}
}

func NewOrderedMapWithCapacity(n int) *OrderedMap {
	return &OrderedMap{
		m:     make(map[string]int, n),
		items: make([]OrderedMapItem, 0, n),
	}
}

// Sets the value for a given key. The order of the key-value pairs is based on the first time a key
// is set for the map. Overwriting an existing value does not change the order.
func (m *OrderedMap) Set(key string, value interface{}) {
	if idx, ok := m.m[key]; !ok {
		m.m[key] = len(m.items)
		m.items = append(m.items, OrderedMapItem{
			Key:   key,
			Value: value,
		})
	} else {
		m.items[idx].Value = value
	}
}

func (m *OrderedMap) Get(key string) (interface{}, bool) {
	if idx, ok := m.m[key]; ok {
		return m.items[idx].Value, true
	}
	return nil, false
}

func (m *OrderedMap) Len() int {
	return len(m.m)
}

func (m *OrderedMap) Items() []OrderedMapItem {
	return m.items
}

func (m *OrderedMap) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(m)
}

type orderedMapEncoder struct{}

func (e *orderedMapEncoder) IsEmpty(ptr unsafe.Pointer) bool {
	m := *((*OrderedMap)(ptr))
	return m.Len() == 0
}
func (e *orderedMapEncoder) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	m := *((*OrderedMap)(ptr))
	stream.WriteObjectStart()
	for i, kv := range m.items {
		if i != 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField(kv.Key)
		stream.WriteVal(kv.Value)
	}
	stream.WriteObjectEnd()
}

func init() {
	jsoniter.RegisterTypeEncoder("executor.OrderedMap", &orderedMapEncoder{})
}

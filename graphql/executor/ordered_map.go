package executor

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"
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
	for i, key := range m.order {
		if i != 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField(key)
		stream.WriteVal(m.m[key])
	}
	stream.WriteObjectEnd()
}

func init() {
	jsoniter.RegisterTypeEncoder("executor.OrderedMap", &orderedMapEncoder{})
}

package executor

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"
)

// OrderedMapItem is a key-value pair for an item in an OrderedMap.
type OrderedMapItem struct {
	Key   string
	Value interface{}
}

// OrderedMap represents a map that maintains the order of its key-value pairs. It's more or less
// just a list that serializes to a JSON map.
type OrderedMap struct {
	items []OrderedMapItem
}

// NewOrderedMap creates a new ordered map.
func NewOrderedMap() *OrderedMap {
	return &OrderedMap{}
}

// NewOrderedMapWithCapacity creates a new ordered map with n elements pre-allocated and
// zero-initialized.
func NewOrderedMapWithLength(n int) *OrderedMap {
	return &OrderedMap{
		items: make([]OrderedMapItem, n),
	}
}

// Set writes a key-value pair to the map at the given index.
func (m *OrderedMap) Set(index int, key string, value interface{}) {
	m.items[index] = OrderedMapItem{
		Key:   key,
		Value: value,
	}
}

// Append appends a key-value pair to the map. It is the caller's responsibility to make sure the
// key doesn't already exist in the map.
func (m *OrderedMap) Append(key string, value interface{}) {
	m.items = append(m.items, OrderedMapItem{
		Key:   key,
		Value: value,
	})
}

// Len returns the length of the map.
func (m *OrderedMap) Len() int {
	return len(m.items)
}

// Items provides the items in the map, in the order they were added.
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

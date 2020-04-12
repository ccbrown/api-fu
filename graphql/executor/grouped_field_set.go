package executor

import (
	"github.com/ccbrown/api-fu/graphql/ast"
)

// GroupedFieldSetItem contains a key and field list pair in a GroupedFieldSet.
type GroupedFieldSetItem struct {
	Key    string
	Fields []*ast.Field
}

// GroupedFieldSet holds the results of the GraphQL CollectFields algorithm.
type GroupedFieldSet struct {
	m     map[string]int
	items []GroupedFieldSetItem
}

// NewGroupedFieldSetWithCapacity allocates a GroupedFieldSet with capacity for n elements.
func NewGroupedFieldSetWithCapacity(n int) *GroupedFieldSet {
	return &GroupedFieldSet{
		m:     make(map[string]int, n),
		items: make([]GroupedFieldSetItem, 0, n),
	}
}

// Append appends a field to the list for the given key.
func (m *GroupedFieldSet) Append(key string, field *ast.Field) {
	if idx, ok := m.m[key]; !ok {
		idx = len(m.items)
		m.m[key] = idx
		m.items = append(m.items, GroupedFieldSetItem{
			Key:    key,
			Fields: []*ast.Field{field},
		})
	} else {
		m.items[idx].Fields = append(m.items[idx].Fields, field)
	}
}

// Len returns the length of the GroupedFieldSet
func (m *GroupedFieldSet) Len() int {
	return len(m.items)
}

// Items returns the items in the GroupedFieldSet, in the order they were added.
func (m *GroupedFieldSet) Items() []GroupedFieldSetItem {
	return m.items
}

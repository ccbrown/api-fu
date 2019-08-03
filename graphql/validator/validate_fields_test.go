package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFields_SelectionsOnObjectsInterfacesAndUnions(t *testing.T) {
	assert.Empty(t, validateSource(t, `{object{scalar}}`))
	assert.Len(t, validateSource(t, `{object{asd}}`), 1)

	assert.Empty(t, validateSource(t, `{interface{scalar}}`))
	assert.Len(t, validateSource(t, `{interface{asd}}`), 1)

	assert.Empty(t, validateSource(t, `{union{__typename}}`))
	assert.Len(t, validateSource(t, `{union{a}}`), 1)
}

func TestFields_LeafFieldSelections(t *testing.T) {
	assert.Empty(t, validateSource(t, `{scalar}`))
	assert.Len(t, validateSource(t, `{scalar{asd}}`), 1)

	assert.Empty(t, validateSource(t, `{interface{scalar}}`))
	assert.Len(t, validateSource(t, `{interface{}}`), 1)
	assert.Len(t, validateSource(t, `{interface}`), 1)

	assert.Empty(t, validateSource(t, `{__typename}`))
	assert.Len(t, validateSource(t, `{__typename{}}`), 1)
}

func TestFields_FieldSelectionMerging(t *testing.T) {
	assert.Empty(t, validateSource(t, `{int int}`))
	assert.Empty(t, validateSource(t, `{a: int a: int}`))
	assert.Len(t, validateSource(t, `{int: int2 int}`), 1)

	assert.Empty(t, validateSource(t, `query($id: ID!){node(id: $id){id} node(id: $id){id}}`))
	assert.Len(t, validateSource(t, `query($id: ID!, $id2: ID!){node(id: $id){id} node(id: $id2){id}}`), 1)

	assert.Empty(t, validateSource(t, `{node(id: "1"){id} node(id: "1"){id}}`))
	assert.Len(t, validateSource(t, `{node(id: "1"){id} node(id: "2"){id}}`), 1)

	assert.Empty(t, validateSource(t, `{pet{... on Dog{volume: barkVolume} ... on Cat{volume: meowVolume}}}`))
	assert.Len(t, validateSource(t, `{pet{... on Dog{someValue: nickname} ... on Cat{someValue: meowVolume}}}`), 1)
}

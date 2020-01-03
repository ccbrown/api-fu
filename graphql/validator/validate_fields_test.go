package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ccbrown/api-fu/graphql/ast"
)

func TestFields_SelectionsOnObjectsInterfacesAndUnions(t *testing.T) {
	assert.Empty(t, validateSource(t, `{object{scalar}}`))
	assert.Len(t, validateSource(t, `{object{asd}}`), 1)

	assert.Empty(t, validateSource(t, `{interface{scalar}}`))
	assert.Len(t, validateSource(t, `{interface{asd}}`), 1)

	assert.Empty(t, validateSource(t, `{union{__typename}}`))
	assert.Len(t, validateSource(t, `{union{a}}`), 1)

	assert.Empty(t, validateSource(t, `{union{__typename, ... on UnionObjectA {__typename}}}`))

	assert.Empty(t, validateSource(t, `{__schema{__typename}}`))
	assert.Empty(t, validateSource(t, `{__type(name:"foo"){__typename}}`))
	assert.Len(t, validateSource(t, `{__type(name:"foo"){asdf}}`), 1)
}

func TestFields_LeafFieldSelections(t *testing.T) {
	assert.Empty(t, validateSource(t, `{scalar}`))
	assert.Len(t, validateSource(t, `{scalar{asd}}`), 1)

	assert.Empty(t, validateSource(t, `{interface{scalar}}`))
	assert.Len(t, validateSource(t, `{interface{x}}`), 1)
	assert.Len(t, validateSource(t, `{interface}`), 1)

	assert.Empty(t, validateSource(t, `{__typename}`))
	assert.Len(t, validateSource(t, `{__typename{x}}`), 1)
}

func TestFields_FieldSelectionMerging(t *testing.T) {
	assert.Empty(t, validateSource(t, `{int int}`))
	assert.Empty(t, validateSource(t, `{a: int a: int}`))
	assert.Len(t, validateSource(t, `{int: int2 int}`), 1)

	assert.Empty(t, validateSource(t, `query($id: ID!){node(id: $id){id} node(id: $id){id}}`))
	assert.Len(t, validateSource(t, `query($id: ID!, $id2: ID!){node(id: $id){id} node(id: $id2){id}}`), 1)

	assert.Empty(t, validateSource(t, `{node(id: "1"){id} node(id: "1"){id}}`))
	assert.Len(t, validateSource(t, `{node(id: "1"){id} node(id: "2"){id}}`), 1)
	assert.Len(t, validateSource(t, `{object{int} object(object:{requiredString:""}){int}}`), 1)

	assert.Empty(t, validateSource(t, `{pet{... on Dog{volume: barkVolume} ... on Cat{volume: meowVolume}}}`))
	assert.Len(t, validateSource(t, `{pet{... on Dog{someValue: nickname} ... on Cat{someValue: meowVolume}}}`), 1)

	assert.Empty(t, validateSource(t, `{int int}`))
	assert.Empty(t, validateSource(t, `{nonNullInt nonNullInt}`))
	assert.Len(t, validateSource(t, `{int int:nonNullInt}`), 1)
	assert.Len(t, validateSource(t, `{int:nonNullInt int}`), 1)

	assert.Empty(t, validateSource(t, `{objects{int} objects{int}}`))
	assert.Empty(t, validateSource(t, `{objects:object{int} objects:object{int}}`))
	assert.Len(t, validateSource(t, `{objects{int} objects:object{int}}`), 1)
	assert.Len(t, validateSource(t, `{objects:object{int} objects{int}}`), 1)
}

func TestValuesAreIdentical(t *testing.T) {
	for name, tc := range map[string]struct {
		A1 ast.Value
		A2 ast.Value
		B1 ast.Value
		B2 ast.Value
	}{
		"Boolean": {
			A1: &ast.BooleanValue{Value: true},
			A2: &ast.BooleanValue{Value: true},
			B1: &ast.BooleanValue{Value: false},
		},
		"Float": {
			A1: &ast.FloatValue{Value: "0.1"},
			A2: &ast.FloatValue{Value: "0.1"},
			B1: &ast.FloatValue{Value: "0.2"},
		},
		"Int": {
			A1: &ast.IntValue{Value: "0"},
			A2: &ast.IntValue{Value: "0"},
			B1: &ast.IntValue{Value: "1"},
		},
		"Enum": {
			A1: &ast.EnumValue{Value: "A"},
			A2: &ast.EnumValue{Value: "A"},
			B1: &ast.EnumValue{Value: "B"},
		},
		"Null": {
			A1: &ast.NullValue{},
			A2: &ast.NullValue{},
			B1: &ast.IntValue{Value: "B"},
		},
		"List": {
			A1: &ast.ListValue{Values: []ast.Value{&ast.IntValue{Value: "0"}}},
			A2: &ast.ListValue{Values: []ast.Value{&ast.IntValue{Value: "0"}}},
			B1: &ast.ListValue{Values: []ast.Value{&ast.IntValue{Value: "1"}}},
			B2: &ast.ListValue{Values: []ast.Value{}},
		},
		"Object": {
			A1: &ast.ObjectValue{Fields: []*ast.ObjectField{
				{
					Name:  &ast.Name{Name: "foo"},
					Value: &ast.IntValue{Value: "0"},
				},
			}},
			A2: &ast.ObjectValue{Fields: []*ast.ObjectField{
				{
					Name:  &ast.Name{Name: "foo"},
					Value: &ast.IntValue{Value: "0"},
				},
			}},
			B1: &ast.ObjectValue{Fields: []*ast.ObjectField{
				{
					Name:  &ast.Name{Name: "foo2"},
					Value: &ast.IntValue{Value: "0"},
				},
			}},
			B2: &ast.ObjectValue{Fields: []*ast.ObjectField{}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.True(t, valuesAreIdentical(tc.A1, tc.A2))
			assert.False(t, valuesAreIdentical(tc.A1, tc.B1))
			assert.False(t, valuesAreIdentical(tc.A2, tc.B1))
			if tc.B2 != nil {
				assert.False(t, valuesAreIdentical(tc.A1, tc.B2))
				assert.False(t, valuesAreIdentical(tc.A2, tc.B2))
			}
		})
	}
}

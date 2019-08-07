package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql/parser"
)

func TestListType_Coercion(t *testing.T) {
	for name, tc := range map[string]struct {
		Type          *ListType
		Literal       string
		VariableValue interface{}
		Okay          bool
		Expected      interface{}
	}{
		"IntList":           {NewListType(IntType), `[1, 2, 3]`, []interface{}{1, 2, 3}, true, []interface{}{1, 2, 3}},
		"MixedList":         {NewListType(IntType), `[1, "b", true]`, []interface{}{1, "b", true}, false, nil},
		"Int":               {NewListType(IntType), `1`, 1, true, []interface{}{1}},
		"Null":              {NewListType(IntType), `null`, nil, true, nil},
		"NestedIntListList": {NewListType(NewListType(IntType)), `[[1], [2, 3]]`, []interface{}{[]interface{}{1}, []interface{}{2, 3}}, true, []interface{}{[]interface{}{1}, []interface{}{2, 3}}},
		"NestedIntList":     {NewListType(NewListType(IntType)), `[1, 2, 3]`, []interface{}{1, 2, 3}, false, nil},
		"NestedInt":         {NewListType(NewListType(IntType)), `1`, 1, true, []interface{}{[]interface{}{1}}},
		"NestedNull":        {NewListType(NewListType(IntType)), `null`, nil, true, nil},
	} {
		t.Run(name, func(t *testing.T) {
			t.Run("Literal", func(t *testing.T) {
				value, errs := parser.ParseValue([]byte(tc.Literal))
				require.Empty(t, errs)
				coerced, err := CoerceLiteral(value, tc.Type, nil)
				if tc.Okay {
					assert.NoError(t, err)
					assert.Equal(t, tc.Expected, coerced)
				} else {
					assert.Error(t, err)
				}
			})

			t.Run("VariableValue", func(t *testing.T) {
				coerced, err := CoerceVariableValue(tc.VariableValue, tc.Type)
				if tc.Okay {
					assert.NoError(t, err)
					assert.Equal(t, tc.Expected, coerced)
				} else {
					assert.Error(t, err)
				}
			})
		})
	}
}

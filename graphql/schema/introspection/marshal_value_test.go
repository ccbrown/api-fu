package introspection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql/schema"
)

func TestMarshalValue(t *testing.T) {
	for name, tc := range map[string]struct {
		Type     schema.Type
		Value    interface{}
		Expected string
	}{
		"InputObject": {
			Type: &schema.InputObjectType{
				Fields: map[string]*schema.InputValueDefinition{
					"foo": {
						Type: schema.IntType,
					},
				},
				ResultCoercion: func(v interface{}) (map[string]interface{}, error) {
					return map[string]interface{}{
						"foo": v,
					}, nil
				},
			},
			Value:    1,
			Expected: "{foo: 1}",
		},
		"List": {
			Type:     schema.NewListType(schema.IntType),
			Value:    []int{1, 2},
			Expected: "[1, 2]",
		},
		"Enum": {
			Type: &schema.EnumType{
				Name: "FooBarEnum",
				Values: map[string]*schema.EnumValueDefinition{
					"FOO": {Value: 1},
					"BAR": {Value: 2},
				},
			},
			Value:    1,
			Expected: "FOO",
		},
	} {
		t.Run(name, func(t *testing.T) {
			s, err := marshalValue(tc.Type, tc.Value)
			require.NoError(t, err)
			assert.Equal(t, tc.Expected, s)
		})
	}
}

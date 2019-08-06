package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql/parser"
)

func TestSchema(t *testing.T) {
	def := &SchemaDefinition{
		Query: &ObjectType{
			Name: "Query",
			Fields: map[string]*FieldDefinition{
				"foo": &FieldDefinition{
					Type: IntType,
				},
			},
		},
	}
	schema, err := New(def)
	assert.NotNil(t, schema)
	assert.NoError(t, err)

	assert.NotNil(t, schema.NamedType("Query"))
	assert.NotNil(t, schema.NamedType("Int"))
}

func TestCoercion(t *testing.T) {
	for name, tc := range map[string]struct {
		JSONInput      string
		LiteralInput   string
		Expected       interface{}
		Type           Type
		VariableValues map[string]interface{}
	}{
		"Complex": {
			JSONInput:    `{"string": null, "stringList": ["a", "b", null], "nonNullInt": 1, "enum": "FOO"}`,
			LiteralInput: `{string: null, stringList: ["a", "b", null], nonNullInt: 1, enum: FOO}`,
			Expected: map[string]interface{}{
				"enum":       "foo",
				"stringList": []interface{}{"a", "b", nil},
				"string":     nil,
				"nonNullInt": 1,
			},
			Type: &InputObjectType{
				Name: "InputObject",
				Fields: map[string]*InputValueDefinition{
					"string": &InputValueDefinition{
						Type:         StringType,
						DefaultValue: "default",
					},
					"stringList": &InputValueDefinition{
						Type: NewListType(StringType),
					},
					"nonNullInt": &InputValueDefinition{
						Type: NewNonNullType(IntType),
					},
					"enum": &InputValueDefinition{
						Type: &EnumType{
							Values: map[string]*EnumValueDefinition{
								"FOO": &EnumValueDefinition{
									Value: "foo",
								},
							},
						},
					},
				},
			},
		},
		"Default": {
			JSONInput:    `{}`,
			LiteralInput: `{}`,
			Expected: map[string]interface{}{
				"string": "default",
			},
			Type: &InputObjectType{
				Name: "InputObject",
				Fields: map[string]*InputValueDefinition{
					"string": &InputValueDefinition{
						Type:         StringType,
						DefaultValue: "default",
					},
				},
			},
		},
		"DefaultNull": {
			JSONInput:    `{}`,
			LiteralInput: `{}`,
			Expected: map[string]interface{}{
				"string": nil,
			},
			Type: &InputObjectType{
				Name: "InputObject",
				Fields: map[string]*InputValueDefinition{
					"string": &InputValueDefinition{
						Type:         StringType,
						DefaultValue: Null,
					},
				},
			},
		},
		"Variable": {
			LiteralInput: `{string: $foo}`,
			VariableValues: map[string]interface{}{
				"foo": "foo",
			},
			Expected: map[string]interface{}{
				"string": "foo",
			},
			Type: &InputObjectType{
				Name: "InputObject",
				Fields: map[string]*InputValueDefinition{
					"string": &InputValueDefinition{
						Type: StringType,
					},
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if tc.JSONInput != "" {
				t.Run("Variable", func(t *testing.T) {
					var variable interface{}
					require.NoError(t, json.Unmarshal([]byte(tc.JSONInput), &variable))
					v, err := CoerceVariableValue(variable, tc.Type)
					require.NoError(t, err)
					assert.Equal(t, tc.Expected, v)
				})
			}
			if tc.LiteralInput != "" {
				t.Run("Literal", func(t *testing.T) {
					literal, errs := parser.ParseValue([]byte(tc.LiteralInput))
					assert.Empty(t, errs)
					v, err := CoerceLiteral(literal, tc.Type, tc.VariableValues)
					require.NoError(t, err)
					assert.Equal(t, tc.Expected, v)
				})
			}
		})
	}
}

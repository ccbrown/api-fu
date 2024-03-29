package introspection_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql/executor"
	"github.com/ccbrown/api-fu/graphql/parser"
	"github.com/ccbrown/api-fu/graphql/schema"
	"github.com/ccbrown/api-fu/graphql/schema/introspection"
)

var petType = &schema.InterfaceType{
	Name: "Pet",
	Fields: map[string]*schema.FieldDefinition{
		"nickname": {
			Type: schema.StringType,
		},
		"age": {
			Type:             schema.IntType,
			RequiredFeatures: schema.NewFeatureSet("petage"),
		},
	},
}

var dogType = &schema.ObjectType{
	Name: "Dog",
	Fields: map[string]*schema.FieldDefinition{
		"nickname": {
			Type: schema.StringType,
		},
		"barkVolume": {
			Type: schema.IntType,
		},
		"age": {
			Type:             schema.IntType,
			RequiredFeatures: schema.NewFeatureSet("petage"),
		},
	},
	ImplementedInterfaces: []*schema.InterfaceType{petType},
	IsTypeOf:              func(interface{}) bool { return false },
}

var fooBarEnumType = &schema.EnumType{
	Name: "FooBarEnum",
	Values: map[string]*schema.EnumValueDefinition{
		"FOO": {},
		"BAR": {},
	},
}

var objectType = &schema.ObjectType{
	Name: "Object",
	Fields: map[string]*schema.FieldDefinition{
		"pet": {
			Type: petType,
			Arguments: map[string]*schema.InputValueDefinition{
				"booleanArg": {
					Type: schema.BooleanType,
				},
			},
		},
		"union": {
			Type: &schema.UnionType{
				Name: "Union",
				MemberTypes: []*schema.ObjectType{
					{
						Name: "UnionObjectA",
						Fields: map[string]*schema.FieldDefinition{
							"a": {
								Type: schema.StringType,
							},
							"scalar": {
								Type: schema.StringType,
							},
						},
						IsTypeOf: func(interface{}) bool { return false },
					},
					{
						Name: "UnionObjectB",
						Fields: map[string]*schema.FieldDefinition{
							"b": {
								Type: schema.StringType,
							},
							"scalar": {
								Type: schema.StringType,
							},
						},
						IsTypeOf: func(interface{}) bool { return false },
					},
				},
			},
		},
		"int": {
			Type: schema.IntType,
		},
		"nonNullInt": {
			Type: schema.NewNonNullType(schema.IntType),
		},
		"enum": {
			Type: fooBarEnumType,
		},
	},
}

func TestIntrospection(t *testing.T) {
	s, err := schema.New(&schema.SchemaDefinition{
		Query: objectType,
		Directives: map[string]*schema.DirectiveDefinition{
			"directive": {
				Locations: []schema.DirectiveLocation{schema.DirectiveLocationField, schema.DirectiveLocationFragmentSpread, schema.DirectiveLocationInlineFragment},
			},
		},
		AdditionalTypes: []schema.NamedType{dogType},
	})
	require.NoError(t, err)
	require.NoError(t, err)
	doc, parseErrs := parser.ParseDocument(introspection.Query)
	require.Empty(t, parseErrs)

	t.Run("Features", func(t *testing.T) {
		data, errs := executor.ExecuteRequest(context.Background(), &executor.Request{
			Document: doc,
			Schema:   s,
			Features: schema.NewFeatureSet("petage"),
		})
		require.Empty(t, errs)
		buf, err := json.Marshal(data)
		require.NoError(t, err)
		assert.Contains(t, string(buf), `"name":"age"`)
	})

	t.Run("NoFeatures", func(t *testing.T) {
		data, errs := executor.ExecuteRequest(context.Background(), &executor.Request{
			Document: doc,
			Schema:   s,
		})
		require.Empty(t, errs)
		buf, err := json.Marshal(data)
		require.NoError(t, err)
		assert.NotContains(t, string(buf), `"name":"age"`)
	})
}

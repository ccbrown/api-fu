package introspection_test

import (
	"context"
	"testing"

	"github.com/ccbrown/api-fu/graphql/executor"
	"github.com/ccbrown/api-fu/graphql/parser"
	"github.com/ccbrown/api-fu/graphql/schema"
	"github.com/ccbrown/api-fu/graphql/schema/introspection"
	"github.com/stretchr/testify/require"
)

var petType = &schema.InterfaceType{
	Name: "Pet",
	Fields: map[string]*schema.FieldDefinition{
		"nickname": &schema.FieldDefinition{
			Type: schema.StringType,
		},
	},
}

var dogType = &schema.ObjectType{
	Name: "Dog",
	Fields: map[string]*schema.FieldDefinition{
		"nickname": &schema.FieldDefinition{
			Type: schema.StringType,
		},
		"barkVolume": &schema.FieldDefinition{
			Type: schema.IntType,
		},
	},
	ImplementedInterfaces: []*schema.InterfaceType{petType},
	IsTypeOf:              func(interface{}) bool { return false },
}

var fooBarEnumType = &schema.EnumType{
	Name: "FooBarEnum",
	Values: map[string]*schema.EnumValueDefinition{
		"FOO": &schema.EnumValueDefinition{},
		"BAR": &schema.EnumValueDefinition{},
	},
}

var objectType = &schema.ObjectType{
	Name: "Object",
	Fields: map[string]*schema.FieldDefinition{
		"pet": &schema.FieldDefinition{
			Type: petType,
			Arguments: map[string]*schema.InputValueDefinition{
				"booleanArg": &schema.InputValueDefinition{
					Type: schema.BooleanType,
				},
			},
		},
		"union": &schema.FieldDefinition{
			Type: &schema.UnionType{
				Name: "Union",
				MemberTypes: []*schema.ObjectType{
					&schema.ObjectType{
						Name: "UnionObjectA",
						Fields: map[string]*schema.FieldDefinition{
							"a": &schema.FieldDefinition{
								Type: schema.StringType,
							},
							"scalar": &schema.FieldDefinition{
								Type: schema.StringType,
							},
						},
						IsTypeOf: func(interface{}) bool { return false },
					},
					&schema.ObjectType{
						Name: "UnionObjectB",
						Fields: map[string]*schema.FieldDefinition{
							"b": &schema.FieldDefinition{
								Type: schema.StringType,
							},
							"scalar": &schema.FieldDefinition{
								Type: schema.StringType,
							},
						},
						IsTypeOf: func(interface{}) bool { return false },
					},
				},
			},
		},
		"int": &schema.FieldDefinition{
			Type: schema.IntType,
		},
		"nonNullInt": &schema.FieldDefinition{
			Type: schema.NewNonNullType(schema.IntType),
		},
	},
}

func TestIntrospection(t *testing.T) {
	s, err := schema.New(&schema.SchemaDefinition{
		Query: objectType,
		Directives: map[string]*schema.DirectiveDefinition{
			"directive": &schema.DirectiveDefinition{
				Locations: []schema.DirectiveLocation{schema.DirectiveLocationField, schema.DirectiveLocationFragmentSpread, schema.DirectiveLocationInlineFragment},
			},
		},
	})
	require.NoError(t, err)
	require.NoError(t, err)
	doc, parseErrs := parser.ParseDocument(introspection.Query)
	require.Empty(t, parseErrs)
	_, errs := executor.ExecuteRequest(context.Background(), &executor.Request{
		Document: doc,
		Schema:   s,
	})
	require.Empty(t, errs)
}

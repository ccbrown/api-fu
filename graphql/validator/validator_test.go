package validator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

var nodeType = &schema.InterfaceType{
	Name: "Node",
	Fields: map[string]*schema.FieldDefinition{
		"id": {
			Type: schema.IDType,
		},
	},
}

var unionMemberType = &schema.InterfaceType{
	Name: "UnionMember",
	Fields: map[string]*schema.FieldDefinition{
		"scalar": {
			Type: schema.StringType,
		},
	},
}

var objectType = &schema.ObjectType{
	Name: "Object",
}

var experimentalObjectType = &schema.ObjectType{
	Name:             "ExperimentalObject",
	RequiredFeatures: schema.NewFeatureSet("experimentalobject"),
	Fields: map[string]*schema.FieldDefinition{
		"foo": {
			Type: schema.BooleanType,
		},
	},
}

var complexInputType = &schema.InputObjectType{
	Name: "ComplexInput",
	Fields: map[string]*schema.InputValueDefinition{
		"name": {
			Type: schema.StringType,
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

type costContextKeyType int

var costContextKey costContextKeyType

func init() {
	objectType.Fields = map[string]*schema.FieldDefinition{
		"freeBoolean": {
			Type: schema.BooleanType,
			Cost: schema.FieldResolverCost(0),
		},
		"experimentalObject": {
			Type:             experimentalObjectType,
			RequiredFeatures: schema.NewFeatureSet("experimentalobject"),
		},
		"booleanArgField": {
			Type: schema.BooleanType,
			Arguments: map[string]*schema.InputValueDefinition{
				"booleanArg": {
					Type: schema.BooleanType,
				},
			},
		},
		"intArgField": {
			Type: schema.IntType,
			Arguments: map[string]*schema.InputValueDefinition{
				"intArg": {
					Type: schema.IntType,
				},
			},
		},
		"enumArgField": {
			Type: fooBarEnumType,
			Arguments: map[string]*schema.InputValueDefinition{
				"enumArg": {
					Type: fooBarEnumType,
				},
			},
		},
		"floatArgField": {
			Type: schema.FloatType,
			Arguments: map[string]*schema.InputValueDefinition{
				"floatArg": {
					Type: schema.FloatType,
				},
			},
		},
		"intListListArgField": {
			Type: schema.NewListType(schema.NewListType(schema.IntType)),
			Arguments: map[string]*schema.InputValueDefinition{
				"intListListArg": {
					Type: schema.NewListType(schema.NewListType(schema.IntType)),
				},
			},
		},
		"nonNullIntListArgField": {
			Type: schema.NewListType(schema.IntType),
			Arguments: map[string]*schema.InputValueDefinition{
				"intListArg": {
					Type: schema.NewNonNullType(schema.NewListType(schema.IntType)),
				},
			},
		},
		"intListArgField": {
			Type: schema.NewListType(schema.IntType),
			Arguments: map[string]*schema.InputValueDefinition{
				"intListArg": {
					Type: schema.NewListType(schema.IntType),
				},
			},
		},
		"findDog": {
			Type: dogType,
			Arguments: map[string]*schema.InputValueDefinition{
				"complex": {
					Type: complexInputType,
				},
			},
		},
		"pet": {
			Type: petType,
		},
		"dog": {
			Type: dogType,
		},
		"cat": {
			Type: &schema.ObjectType{
				Name: "Cat",
				Fields: map[string]*schema.FieldDefinition{
					"nickname": {
						Type: schema.StringType,
					},
					"meowVolume": {
						Type: schema.IntType,
					},
					"age": {
						Type: schema.IntType,
					},
				},
				ImplementedInterfaces: []*schema.InterfaceType{petType},
				IsTypeOf:              func(interface{}) bool { return false },
			},
		},
		"node": {
			Type: nodeType,
			Arguments: map[string]*schema.InputValueDefinition{
				"id": {
					Type: schema.NewNonNullType(schema.IDType),
				},
			},
		},
		"resource": {
			Type: &schema.ObjectType{
				Name: "Resource",
				Fields: map[string]*schema.FieldDefinition{
					"id": {
						Type: schema.IDType,
					},
				},
				ImplementedInterfaces: []*schema.InterfaceType{nodeType},
				IsTypeOf:              func(interface{}) bool { return false },
			},
		},
		"costFromArg": {
			Type: schema.IntType,
			Arguments: map[string]*schema.InputValueDefinition{
				"cost": {
					Type:         schema.IntType,
					DefaultValue: 10,
				},
			},
			Cost: func(ctx schema.FieldCostContext) schema.FieldCost {
				cost, _ := ctx.Arguments["cost"].(int)
				return schema.FieldCost{
					Resolver: cost,
				}
			},
		},
		"objectWithCostContext": {
			Type: objectType,
			Arguments: map[string]*schema.InputValueDefinition{
				"cost": {
					Type: schema.IntType,
				},
			},
			Cost: func(ctx schema.FieldCostContext) schema.FieldCost {
				cost, _ := ctx.Arguments["cost"].(int)
				return schema.FieldCost{
					Context: context.WithValue(ctx.Context, costContextKey, cost),
				}
			},
		},
		"costFromContext": {
			Type: schema.IntType,
			Cost: func(ctx schema.FieldCostContext) schema.FieldCost {
				return schema.FieldCost{
					Resolver: ctx.Context.Value(costContextKey).(int),
				}
			},
		},
		"objects": {
			Type: schema.NewListType(objectType),
			Arguments: map[string]*schema.InputValueDefinition{
				"first": {
					Type: schema.IntType,
				},
			},
			Cost: func(ctx schema.FieldCostContext) schema.FieldCost {
				multiplier, _ := ctx.Arguments["first"].(int)
				return schema.FieldCost{
					Resolver:   1,
					Multiplier: multiplier,
				}
			},
		},
		"object": {
			Type: objectType,
			Arguments: map[string]*schema.InputValueDefinition{
				"object": {
					Type: &schema.InputObjectType{
						Name: "ObjectInput",
						Fields: map[string]*schema.InputValueDefinition{
							"defaultedString": {
								Type:         schema.NewNonNullType(schema.StringType),
								DefaultValue: "foo",
							},
							"requiredString": {
								Type: schema.NewNonNullType(schema.StringType),
							},
						},
					},
				},
			},
		},
		"object2": {
			Type: &schema.ObjectType{
				Name: "Object2",
				Fields: map[string]*schema.FieldDefinition{
					"scalar": {
						Type: schema.StringType,
					},
				},
			},
		},
		"interface": {
			Type: &schema.InterfaceType{
				Name: "Interface",
				Fields: map[string]*schema.FieldDefinition{
					"scalar": {
						Type: schema.StringType,
					},
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
						ImplementedInterfaces: []*schema.InterfaceType{unionMemberType},
						IsTypeOf:              func(interface{}) bool { return false },
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
						ImplementedInterfaces: []*schema.InterfaceType{unionMemberType},
						IsTypeOf:              func(interface{}) bool { return false },
					},
				},
			},
		},
		"scalar": {
			Type: schema.StringType,
		},
		"int": {
			Type: schema.IntType,
		},
		"nonNullInt": {
			Type: schema.NewNonNullType(schema.IntType),
		},
		"int2": {
			Type: schema.IntType,
		},
	}
}

func validateSource(t *testing.T, src string, features ...string) []*Error {
	s, err := schema.New(&schema.SchemaDefinition{
		Query:        objectType,
		Subscription: objectType,
		Directives: map[string]*schema.DirectiveDefinition{
			"include": schema.IncludeDirective,
			"skip":    schema.SkipDirective,
		},
	})
	require.NoError(t, err)
	return validateSourceWithSchema(t, s, src, features...)
}

func validateSourceWithSchema(t *testing.T, s *schema.Schema, src string, features ...string) []*Error {
	doc, parseErrs := parser.ParseDocument([]byte(src))
	require.Empty(t, parseErrs)
	require.NotNil(t, doc)

	errs := ValidateDocument(doc, s, schema.NewFeatureSet(features...))
	for _, err := range errs {
		assert.NotEmpty(t, err.Message)
		assert.NotEmpty(t, err.Locations)
		assert.False(t, err.isSecondary)
	}
	return errs
}

func TestIntrospectionQuery(t *testing.T) {
	assert.Empty(t, validateSource(t, string(introspection.Query)))
}

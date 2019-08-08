package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql/parser"
	"github.com/ccbrown/api-fu/graphql/schema"
)

var petType = &schema.InterfaceType{
	Name: "Pet",
	Fields: map[string]*schema.FieldDefinition{
		"nickname": &schema.FieldDefinition{
			Type: schema.StringType,
		},
	},
}

var nodeType = &schema.InterfaceType{
	Name: "Node",
	Fields: map[string]*schema.FieldDefinition{
		"id": &schema.FieldDefinition{
			Type: schema.IDType,
		},
	},
}

var unionMemberType = &schema.InterfaceType{
	Name: "UnionMember",
	Fields: map[string]*schema.FieldDefinition{
		"scalar": &schema.FieldDefinition{
			Type: schema.StringType,
		},
	},
}

var objectType = &schema.ObjectType{
	Name: "Object",
}

var complexInputType = &schema.InputObjectType{
	Name: "ComplexInput",
	Fields: map[string]*schema.InputValueDefinition{
		"name": &schema.InputValueDefinition{
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

func init() {
	objectType.Fields = map[string]*schema.FieldDefinition{
		"booleanArgField": &schema.FieldDefinition{
			Type: schema.BooleanType,
			Arguments: map[string]*schema.InputValueDefinition{
				"booleanArg": &schema.InputValueDefinition{
					Type: schema.BooleanType,
				},
			},
		},
		"intArgField": &schema.FieldDefinition{
			Type: schema.IntType,
			Arguments: map[string]*schema.InputValueDefinition{
				"intArg": &schema.InputValueDefinition{
					Type: schema.IntType,
				},
			},
		},
		"enumArgField": &schema.FieldDefinition{
			Type: fooBarEnumType,
			Arguments: map[string]*schema.InputValueDefinition{
				"enumArg": &schema.InputValueDefinition{
					Type: fooBarEnumType,
				},
			},
		},
		"floatArgField": &schema.FieldDefinition{
			Type: schema.FloatType,
			Arguments: map[string]*schema.InputValueDefinition{
				"floatArg": &schema.InputValueDefinition{
					Type: schema.FloatType,
				},
			},
		},
		"intListListArgField": &schema.FieldDefinition{
			Type: schema.NewListType(schema.NewListType(schema.IntType)),
			Arguments: map[string]*schema.InputValueDefinition{
				"intListListArg": &schema.InputValueDefinition{
					Type: schema.NewListType(schema.NewListType(schema.IntType)),
				},
			},
		},
		"intListArgField": &schema.FieldDefinition{
			Type: schema.NewListType(schema.IntType),
			Arguments: map[string]*schema.InputValueDefinition{
				"intListArg": &schema.InputValueDefinition{
					Type: schema.NewListType(schema.IntType),
				},
			},
		},
		"findDog": &schema.FieldDefinition{
			Type: dogType,
			Arguments: map[string]*schema.InputValueDefinition{
				"complex": &schema.InputValueDefinition{
					Type: complexInputType,
				},
			},
		},
		"pet": &schema.FieldDefinition{
			Type: petType,
		},
		"dog": &schema.FieldDefinition{
			Type: dogType,
		},
		"cat": &schema.FieldDefinition{
			Type: &schema.ObjectType{
				Name: "Cat",
				Fields: map[string]*schema.FieldDefinition{
					"nickname": &schema.FieldDefinition{
						Type: schema.StringType,
					},
					"meowVolume": &schema.FieldDefinition{
						Type: schema.IntType,
					},
				},
				ImplementedInterfaces: []*schema.InterfaceType{petType},
				IsTypeOf:              func(interface{}) bool { return false },
			},
		},
		"node": &schema.FieldDefinition{
			Type: nodeType,
			Arguments: map[string]*schema.InputValueDefinition{
				"id": &schema.InputValueDefinition{
					Type: schema.NewNonNullType(schema.IDType),
				},
			},
		},
		"resource": &schema.FieldDefinition{
			Type: &schema.ObjectType{
				Name: "Resource",
				Fields: map[string]*schema.FieldDefinition{
					"id": &schema.FieldDefinition{
						Type: schema.IDType,
					},
				},
				ImplementedInterfaces: []*schema.InterfaceType{nodeType},
				IsTypeOf:              func(interface{}) bool { return false },
			},
		},
		"objects": &schema.FieldDefinition{
			Type: schema.NewListType(objectType),
		},
		"object": &schema.FieldDefinition{
			Type: objectType,
			Arguments: map[string]*schema.InputValueDefinition{
				"object": &schema.InputValueDefinition{
					Type: &schema.InputObjectType{
						Name: "ObjectInput",
						Fields: map[string]*schema.InputValueDefinition{
							"defaultedString": &schema.InputValueDefinition{
								Type:         schema.NewNonNullType(schema.StringType),
								DefaultValue: "foo",
							},
							"requiredString": &schema.InputValueDefinition{
								Type: schema.NewNonNullType(schema.StringType),
							},
						},
					},
				},
			},
		},
		"object2": &schema.FieldDefinition{
			Type: &schema.ObjectType{
				Name: "Object2",
				Fields: map[string]*schema.FieldDefinition{
					"scalar": &schema.FieldDefinition{
						Type: schema.StringType,
					},
				},
			},
		},
		"interface": &schema.FieldDefinition{
			Type: &schema.InterfaceType{
				Name: "Interface",
				Fields: map[string]*schema.FieldDefinition{
					"scalar": &schema.FieldDefinition{
						Type: schema.StringType,
					},
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
						ImplementedInterfaces: []*schema.InterfaceType{unionMemberType},
						IsTypeOf:              func(interface{}) bool { return false },
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
						ImplementedInterfaces: []*schema.InterfaceType{unionMemberType},
						IsTypeOf:              func(interface{}) bool { return false },
					},
				},
			},
		},
		"scalar": &schema.FieldDefinition{
			Type: schema.StringType,
		},
		"int": &schema.FieldDefinition{
			Type: schema.IntType,
		},
		"nonNullInt": &schema.FieldDefinition{
			Type: schema.NewNonNullType(schema.IntType),
		},
		"int2": &schema.FieldDefinition{
			Type: schema.IntType,
		},
	}
}

func validateSource(t *testing.T, src string) []*Error {
	s, err := schema.New(&schema.SchemaDefinition{
		Query:        objectType,
		Subscription: objectType,
		DirectiveDefinitions: map[string]*schema.DirectiveDefinition{
			"include": schema.IncludeDirective,
			"skip":    schema.SkipDirective,
		},
	})
	require.NoError(t, err)
	return validateSourceWithSchema(t, s, src)
}

func validateSourceWithSchema(t *testing.T, s *schema.Schema, src string) []*Error {
	doc, parseErrs := parser.ParseDocument([]byte(src))
	require.Empty(t, parseErrs)
	require.NotNil(t, doc)

	errs := ValidateDocument(doc, s)
	for _, err := range errs {
		assert.NotEmpty(t, err.Message)
		assert.NotEmpty(t, err.Locations)
		assert.False(t, err.isSecondary)
	}
	return errs
}

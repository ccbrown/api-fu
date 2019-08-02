package validator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ccbrown/apifu/graphql/parser"
	"github.com/ccbrown/apifu/graphql/schema"
)

func validateSource(t *testing.T, src string) []*Error {
	s, err := schema.New(&schema.SchemaDefinition{
		Query: &schema.ObjectType{
			Name: "Query",
			Fields: map[string]*schema.FieldDefinition{
				"node": &schema.FieldDefinition{
					Type: &schema.InterfaceType{
						Name: "Node",
						Fields: map[string]*schema.FieldDefinition{
							"id": &schema.FieldDefinition{
								Type: schema.IDType,
							},
						},
					},
					Arguments: map[string]*schema.InputValueDefinition{
						"id": &schema.InputValueDefinition{
							Type: schema.NewNonNullType(schema.IDType),
						},
					},
				},
				"object": &schema.FieldDefinition{
					Type: &schema.ObjectType{
						Name: "Object",
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
						MemberTypes: []schema.NamedType{
							&schema.ObjectType{
								Name: "UnionObjectA",
								Fields: map[string]*schema.FieldDefinition{
									"a": &schema.FieldDefinition{
										Type: schema.StringType,
									},
								},
							},
							&schema.ObjectType{
								Name: "UnionObjectB",
								Fields: map[string]*schema.FieldDefinition{
									"b": &schema.FieldDefinition{
										Type: schema.StringType,
									},
								},
							},
						},
					},
				},
				"scalar": &schema.FieldDefinition{
					Type: schema.StringType,
				},
			},
		},
	})
	require.NoError(t, err)

	doc, errs := parser.ParseDocument([]byte(src))
	require.Empty(t, errs)
	require.NotNil(t, doc)
	return ValidateDocument(doc, s)
}

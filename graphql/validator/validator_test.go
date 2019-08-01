package validator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ccbrown/apifu/graphql/parser"
	"github.com/ccbrown/apifu/graphql/schema"
)

func validateSource(t *testing.T, src string) []*Error {
	s, err := schema.New(&schema.SchemaDefinition{
		Query: &schema.ObjectDefinition{
			Name: "Query",
			Fields: map[string]*schema.FieldDefinition{
				"node": &schema.FieldDefinition{
					Type: &schema.InterfaceDefinition{
						Name: "Node",
						Fields: map[string]*schema.FieldDefinition{
							"id": &schema.FieldDefinition{
								Type: schema.IDType,
							},
						},
					},
					Arguments: map[string]*schema.InputValueDefinition{
						"id": &schema.InputValueDefinition{
							Type: schema.NonNull(schema.IDType),
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

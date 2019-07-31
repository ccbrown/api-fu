package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchema(t *testing.T) {
	def := &SchemaDefinition{
		Query: &ObjectDefinition{
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
}

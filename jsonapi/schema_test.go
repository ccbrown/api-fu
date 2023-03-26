package jsonapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemaValidation(t *testing.T) {
	for name, tc := range map[string]struct {
		In   *SchemaDefinition
		Okay bool
	}{
		"InvalidResourceName": {
			In: &SchemaDefinition{
				ResourceTypes: map[string]AnyResourceType{
					"arti!cles": ResourceType[struct{}]{},
				},
			},
			Okay: false,
		},
		"InvalidResourceDefinition": {
			In: &SchemaDefinition{
				ResourceTypes: map[string]AnyResourceType{
					"articles": ResourceType[struct{}]{
						Attributes: map[string]*AttributeDefinition[struct{}]{
							"tit!le": {},
						},
					},
				},
			},
			Okay: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewSchema(tc.In)
			if tc.Okay {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

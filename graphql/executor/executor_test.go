package executor

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql/parser"
	"github.com/ccbrown/api-fu/graphql/schema"
	"github.com/ccbrown/api-fu/graphql/validator"
)

var objectType = &schema.ObjectType{
	Name: "Object",
}

func init() {
	objectType.Fields = map[string]*schema.FieldDefinition{
		"intOne": &schema.FieldDefinition{
			Type: schema.IntType,
			Resolve: func(*schema.FieldContext) (interface{}, error) {
				return 1, nil
			},
		},
		"stringFoo": &schema.FieldDefinition{
			Type: schema.StringType,
			Resolve: func(*schema.FieldContext) (interface{}, error) {
				return "foo", nil
			},
		},
		"object": &schema.FieldDefinition{
			Type: objectType,
			Resolve: func(*schema.FieldContext) (interface{}, error) {
				return struct{}{}, nil
			},
		},
	}
}

func TestExecuteRequest(t *testing.T) {
	s, err := schema.New(&schema.SchemaDefinition{
		Query:        objectType,
		Subscription: objectType,
		DirectiveDefinitions: map[string]*schema.DirectiveDefinition{
			"include": schema.IncludeDirective,
			"skip":    schema.SkipDirective,
		},
	})
	require.NoError(t, err)

	for name, tc := range map[string]struct {
		Document     string
		ExpectedData string
	}{
		"Query": {
			Document:     `{intOne stringFoo object {intOne}}`,
			ExpectedData: `{"intOne":1,"stringFoo":"foo","object":{"intOne":1}}`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			parsed, errs := parser.ParseDocument([]byte(tc.Document))
			require.Empty(t, errs)
			require.Empty(t, validator.ValidateDocument(parsed, s))
			response := ExecuteRequest(&Request{
				Document: parsed,
				Schema:   s,
			})
			serializedData, err := json.Marshal(response.Data)
			require.NoError(t, err)
			assert.Equal(t, tc.ExpectedData, string(serializedData))
		})
	}
}

func TestGetOperation(t *testing.T) {
	doc, errs := parser.ParseDocument([]byte(`{x} {x} query q {x} mutation m {x} mutation m {x}`))
	assert.Empty(t, errs)

	op, err := getOperation(doc, "")
	assert.NotEmpty(t, err)

	op, err = getOperation(doc, "m")
	assert.NotEmpty(t, err)

	op, err = getOperation(doc, "q")
	assert.NotNil(t, op)
	assert.Empty(t, err)
}

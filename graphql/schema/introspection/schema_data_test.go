package introspection_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql"
	"github.com/ccbrown/api-fu/graphql/schema"
	"github.com/ccbrown/api-fu/graphql/schema/introspection"
)

func TestSchemaData(t *testing.T) {
	f, err := os.Open("testdata/github-schema.json")
	require.NoError(t, err)
	defer f.Close()

	var result struct {
		Data struct {
			Schema introspection.SchemaData `json:"__schema"`
		}
	}
	require.NoError(t, json.NewDecoder(f).Decode(&result))

	def, err := result.Data.Schema.GetSchemaDefinition()
	require.NoError(t, err)

	schema, err := schema.New(def)
	require.NoError(t, err)

	t.Run("GoodQuery", func(t *testing.T) {
		query := `query FindIssueID {
			  repository(owner:"octocat", name:"Hello-World") {
				issue(number:349) {
				  id
				}
			  }
			}
		`

		doc, errs := graphql.ParseAndValidate(query, schema, nil)
		require.Empty(t, errs)
		assert.NotNil(t, doc)
	})

	t.Run("BadQuery", func(t *testing.T) {
		query := `query FindIssueID {
			  repository(owner:"octocat", name:"Hello-World") {
				isue(number:349) {
				  id
				}
			  }
			}
		`

		_, errs := graphql.ParseAndValidate(query, schema, nil)
		assert.NotEmpty(t, errs)
	})

	t.Run("UnreferencedInterface", func(t *testing.T) {
		query := `{
				node(id: "foo") {
					... on RepositoryInvitation {
						id
					}
				}
			}
		`

		_, errs := graphql.ParseAndValidate(query, schema, nil)
		assert.Empty(t, errs)
	})
}

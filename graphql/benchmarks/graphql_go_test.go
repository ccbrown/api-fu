package benchmarks

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"

	"github.com/graphql-go/graphql"
)

func BenchmarkGraphQLGo(b *testing.B) {
	var objectFields graphql.Fields
	var objectType = graphql.NewObject(graphql.ObjectConfig{
		Name: "Object",
		Fields: graphql.FieldsThunk(func() graphql.Fields {
			return objectFields
		}),
	})
	objectFields = graphql.Fields{
		"string": {
			Type: graphql.String,
			Resolve: graphql.FieldResolveFn(func(graphql.ResolveParams) (interface{}, error) {
				return "foo", nil
			}),
		},
		"objects": {
			Type: graphql.NewList(objectType),
			Args: graphql.FieldConfigArgument{
				"count": {
					Type: graphql.NewNonNull(graphql.Int),
				},
			},
			Resolve: graphql.FieldResolveFn(func(params graphql.ResolveParams) (interface{}, error) {
				return make([]struct{}, params.Args["count"].(int)), nil
			}),
		},
	}

	s, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: objectType,
	})
	require.NoError(b, err)

	p := graphql.Params{
		RequestString: `{
			string
			objects(count: 20) {
				string
				objects(count: 100) {
					string
				}
			}
		}`,
		Schema: s,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sink, _ = jsoniter.Marshal(graphql.Do(p))
	}
}

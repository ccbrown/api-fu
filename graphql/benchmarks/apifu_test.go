package benchmarks

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql"
)

var sink interface{}

func BenchmarkAPIFu(b *testing.B) {
	var objectType = &graphql.ObjectType{
		Name: "Object",
	}

	objectType.Fields = map[string]*graphql.FieldDefinition{
		"string": {
			Type: graphql.StringType,
			Resolve: func(*graphql.FieldContext) (interface{}, error) {
				return "foo", nil
			},
		},
		"objects": {
			Type: graphql.NewListType(objectType),
			Arguments: map[string]*graphql.InputValueDefinition{
				"count": {
					Type: graphql.NewNonNullType(graphql.IntType),
				},
			},
			Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
				return make([]struct{}, ctx.Arguments["count"].(int)), nil
			},
		},
	}

	s, err := graphql.NewSchema(&graphql.SchemaDefinition{
		Query: objectType,
	})
	require.NoError(b, err)
	r := &graphql.Request{
		Query: `{
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
		sink, _ = jsoniter.Marshal(graphql.Execute(r))
	}
}

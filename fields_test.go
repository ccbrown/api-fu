package apifu

import (
	"context"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/ccbrown/api-fu/graphql"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFields(t *testing.T) {
	type node struct {
		Id string
	}

	testCfg := Config{
		ResolveNodesByGlobalIds: func(ctx context.Context, ids []string) ([]interface{}, error) {
			var ret []interface{}
			for _, id := range ids {
				if id == "a" || id == "b" {
					ret = append(ret, &node{Id: id})
				}
			}
			return ret, nil
		},
	}

	nodeType := &graphql.ObjectType{
		Name: "TestNode",
		Fields: map[string]*graphql.FieldDefinition{
			"id": {
				Type: graphql.NewNonNullType(graphql.IDType),
				Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
					return ctx.Object.(*node).Id, nil
				},
			},
		},
		ImplementedInterfaces: []*graphql.InterfaceType{testCfg.NodeInterface()},
		IsTypeOf: func(value interface{}) bool {
			_, ok := value.(*node)
			return ok
		},
	}
	testCfg.AddNamedType(nodeType)

	// If this is not executed asynchronously alongside a matching asyncReceiver, it will deadlock.
	testCfg.AddQueryField("obj", &graphql.FieldDefinition{
		Type: &graphql.ObjectType{
			Name: "Object",
			Fields: map[string]*graphql.FieldDefinition{
				"int": NonNull(graphql.IntType, "Int"),
				"s0":  NonEmptyString("S0"),
				"s1":  NonEmptyString("S1"),
			},
		},
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			return struct {
				Int    int
				S0     string
				S1     string
				NodeId string
			}{
				S1: "foo",
			}, nil
		},
	})

	api, err := NewAPI(&testCfg)
	require.NoError(t, err)

	resp := executeGraphQL(t, api, `{
		obj {
			int
			s0
			s1
		}
	}`)

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"data":{"obj":{"int":0,"s0":null,"s1":"foo"}}}`, string(body))
}

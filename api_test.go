package apifu

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql"
)

func executeGraphQL(t *testing.T, api *API, query string) *http.Response {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("POST", "", strings.NewReader(query))
	r.Header.Set("Content-Type", "application/graphql")
	require.NoError(t, err)
	api.ServeGraphQL(w, r)
	return w.Result()
}

func TestGo(t *testing.T) {
	var asyncChannel = make(chan struct{})

	var testCfg Config

	// If this is not executed asynchronously alongside a matching asyncReceiver, it will deadlock.
	testCfg.AddQueryField("asyncSender", &graphql.FieldDefinition{
		Type: graphql.BooleanType,
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			return Go(ctx.Context, func() (interface{}, error) {
				asyncChannel <- struct{}{}
				return true, nil
			}), nil
		},
	})

	// If this is not executed asynchronously alongside a matching asyncSender, it will deadlock.
	testCfg.AddQueryField("asyncReceiver", &graphql.FieldDefinition{
		Type: graphql.BooleanType,
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			return Go(ctx.Context, func() (interface{}, error) {
				<-asyncChannel
				return true, nil
			}), nil
		},
	})

	api, err := NewAPI(&testCfg)
	require.NoError(t, err)

	resp := executeGraphQL(t, api, `{
		s: asyncSender
		r: asyncReceiver
	}`)

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"data":{"s":true,"r":true}}`, string(body))
}

func TestBatch(t *testing.T) {
	var testCfg Config

	testCfg.AddQueryField("batched1", &graphql.FieldDefinition{
		Type: graphql.IntType,
		Resolve: Batch(func(ctx []graphql.FieldContext) []graphql.ResolveResult {
			assert.Len(t, ctx, 1)
			return []graphql.ResolveResult{
				{Value: 1, Error: nil},
			}
		}),
	})

	testCfg.AddQueryField("batched2", &graphql.FieldDefinition{
		Type: graphql.IntType,
		Resolve: Batch(func(ctx []graphql.FieldContext) []graphql.ResolveResult {
			assert.Len(t, ctx, 2)
			return []graphql.ResolveResult{
				{Value: 1, Error: nil},
				{Value: 2, Error: nil},
			}
		}),
	})

	api, err := NewAPI(&testCfg)
	require.NoError(t, err)

	resp := executeGraphQL(t, api, `{
		b11: batched1
		b21: batched2
		b22: batched2
	}`)

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"data":{"b11":1,"b21":1,"b22":2}}`, string(body))
}

func TestNodes(t *testing.T) {
	const nodeTypeId = 10

	testCfg := Config{
		SerializeNodeId: func(typeId int, id interface{}) string {
			assert.Equal(t, nodeTypeId, typeId)
			return id.(string)
		},
		DeserializeNodeId: func(id string) (int, interface{}) {
			return nodeTypeId, id
		},
	}

	type node struct {
		Id string
	}

	testCfg.AddNodeType(&NodeType{
		Id:    nodeTypeId,
		Name:  "TestNode",
		Model: reflect.TypeOf(node{}),
		GetByIds: func(ctx context.Context, ids interface{}) (interface{}, error) {
			var ret []*node
			for _, id := range ids.([]string) {
				if id == "a" || id == "b" {
					ret = append(ret, &node{
						Id: id,
					})
				}
			}
			return ret, nil
		},
		Fields: map[string]*graphql.FieldDefinition{
			"id": OwnID("Id"),
		},
	})

	api, err := NewAPI(&testCfg)
	require.NoError(t, err)

	t.Run("Single", func(t *testing.T) {
		resp := executeGraphQL(t, api, `{
			a: node(id: "a") {
				id
			}
			c: node(id: "c") {
				id
			}
		}`)

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data struct {
				A *node
				C *node
			}
			Errors []struct{}
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Empty(t, result.Errors)

		assert.NotNil(t, result.Data.A)
		assert.Nil(t, result.Data.C)
	})

	t.Run("Multiple", func(t *testing.T) {
		resp := executeGraphQL(t, api, `{
			nodes(ids: ["a", "b", "c", "d"]) {
				id
			}
		}`)

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data struct {
				Nodes []node
			}
			Errors []struct{}
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Empty(t, result.Errors)

		assert.ElementsMatch(t, []node{{Id: "a"}, {Id: "b"}}, result.Data.Nodes)
	})
}

func TestMutation(t *testing.T) {
	var testCfg Config

	testCfg.AddMutation("mut", &graphql.FieldDefinition{
		Type: graphql.BooleanType,
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			return true, nil
		},
	})

	api, err := NewAPI(&testCfg)
	require.NoError(t, err)

	resp := executeGraphQL(t, api, `mutation {
		mut
	}`)

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"data":{"mut":true}}`, string(body))
}

package apifu

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql"
)

func executeGraphQL(t *testing.T, api *API, query string) *http.Response {
	return executeGraphQLWithFeatures(t, api, query, nil)
}

const featuresContextKey = "features"

func featuresFromContext(ctx context.Context) graphql.FeatureSet {
	features, _ := ctx.Value(featuresContextKey).(graphql.FeatureSet)
	return features
}

func executeGraphQLWithFeatures(t *testing.T, api *API, query string, features []string) *http.Response {
	w := httptest.NewRecorder()
	ctx := context.WithValue(context.Background(), featuresContextKey, graphql.NewFeatureSet(features...))
	r, err := http.NewRequestWithContext(ctx, "POST", "", strings.NewReader(query))
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

	testCfg.AddNamedType(&graphql.ObjectType{
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
			one: node(id: 1) {
				id
			}
		}`)

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result struct {
			Data struct {
				A   *node
				C   *node
				One *node
			}
			Errors []struct{}
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Empty(t, result.Errors)

		assert.NotNil(t, result.Data.A)
		assert.Nil(t, result.Data.C)
		assert.Nil(t, result.Data.One)
	})

	t.Run("Multiple", func(t *testing.T) {
		resp := executeGraphQL(t, api, `{
			nodes(ids: ["a", "b", "c", "d", 1]) {
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

func TestFeatures(t *testing.T) {
	var testCfg Config
	testCfg.Features = featuresFromContext

	testCfg.AddQueryField("foo", &graphql.FieldDefinition{
		Type: graphql.BooleanType,
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			return true, nil
		},
	})

	testCfg.AddQueryField("bar", &graphql.FieldDefinition{
		Type:             graphql.BooleanType,
		RequiredFeatures: graphql.NewFeatureSet("bar"),
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			return true, nil
		},
	})

	api, err := NewAPI(&testCfg)
	require.NoError(t, err)

	t.Run("NoFeatures", func(t *testing.T) {
		resp := executeGraphQL(t, api, `{
			foo
		}`)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"data":{"foo":true}}`, string(body))
	})

	t.Run("NoFeatures_Error", func(t *testing.T) {
		resp := executeGraphQL(t, api, `{
			foo
			bar
		}`)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"errors":[{"locations":[{"column":4,"line":3}],"message":"Validation error: field bar does not exist on Query"}]}`, string(body))
	})

	t.Run("BarFeature", func(t *testing.T) {
		resp := executeGraphQLWithFeatures(t, api, `{
			foo
			bar
		}`, []string{"bar"})
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"data":{"foo":true,"bar":true}}`, string(body))
	})
}

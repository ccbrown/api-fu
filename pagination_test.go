package apifu

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ccbrown/api-fu/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnection(t *testing.T) {
	config := &Config{}
	config.AddQueryField("connection", Connection(&ConnectionConfig{
		NamePrefix: "Test",
		ResolveEdges: func(ctx *graphql.FieldContext, after, before interface{}, limit int) (edgeSlice interface{}, cursorLess func(a, b interface{}) bool, err error) {
			ret := make([]int, limit)
			for i := range ret {
				ret[i] = i
			}
			return ret, func(a, b interface{}) bool {
				return false
			}, nil
		},
		ResolveTotalCount: func(ctx *graphql.FieldContext) (interface{}, error) {
			return 1000, nil
		},
		CursorType: reflect.TypeOf(""),
		EdgeCursor: func(edge interface{}) interface{} {
			return strconv.Itoa(edge.(int))
		},
		EdgeFields: map[string]*graphql.FieldDefinition{
			"node": &graphql.FieldDefinition{
				Type: graphql.IntType,
				Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
					return ctx.Object, nil
				},
			},
		},
	}))

	api, err := NewAPI(config)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{
		connection(first: 10) {
			edges {
				node
				cursor
			}
			pageInfo {
				hasPreviousPage
				hasNextPage
				startCursor
				endCursor
			}
			totalCount
		}
	}`))
	req.Header.Set("Content-Type", "application/graphql")
	w := httptest.NewRecorder()

	api.ServeGraphQL(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	assert.JSONEq(t, `{
		"data": {
			"connection": {
				"edges": [
					{
						"cursor": "oTA",
						"node": 0
					},
					{
						"cursor": "oTE",
						"node": 1
					},
					{
						"cursor": "oTI",
						"node": 2
					},
					{
						"cursor": "oTM",
						"node": 3
					},
					{
						"cursor": "oTQ",
						"node": 4
					},
					{
						"cursor": "oTU",
						"node": 5
					},
					{
						"cursor": "oTY",
						"node": 6
					},
					{
						"cursor": "oTc",
						"node": 7
					},
					{
						"cursor": "oTg",
						"node": 8
					},
					{
						"cursor": "oTk",
						"node": 9
					}
				],
				"pageInfo": {
					"endCursor": "oTk",
					"hasNextPage": true,
					"hasPreviousPage": false,
					"startCursor": "oTA"
				},
				"totalCount": 1000
			}
		}
	}`, string(body))
}

func TestConnection_ZeroArg_WithoutPageInfo(t *testing.T) {
	config := &Config{}
	config.AddQueryField("connection", Connection(&ConnectionConfig{
		NamePrefix: "Test",
		ResolveEdges: func(ctx *graphql.FieldContext, after, before interface{}, limit int) (edgeSlice interface{}, cursorLess func(a, b interface{}) bool, err error) {
			return nil, nil, fmt.Errorf("the edge resolver should not be invoked")
		},
		ResolveTotalCount: func(ctx *graphql.FieldContext) (interface{}, error) {
			return 1000, nil
		},
		CursorType: reflect.TypeOf(""),
		EdgeCursor: func(edge interface{}) interface{} {
			return strconv.Itoa(edge.(int))
		},
		EdgeFields: map[string]*graphql.FieldDefinition{
			"node": &graphql.FieldDefinition{
				Type: graphql.IntType,
				Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
					return ctx.Object, nil
				},
			},
		},
	}))

	api, err := NewAPI(config)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{
		connection(first: 0) {
			edges {
				node
			}
			totalCount
		}
	}`))
	req.Header.Set("Content-Type", "application/graphql")
	w := httptest.NewRecorder()

	api.ServeGraphQL(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	assert.JSONEq(t, `{
		"data": {
			"connection": {
				"edges": [],
				"totalCount": 1000
			}
		}
	}`, string(body))
}

func TestConnection_ZeroArg_WithPageInfo(t *testing.T) {
	config := &Config{}
	config.AddQueryField("connection", Connection(&ConnectionConfig{
		NamePrefix: "Test",
		ResolveEdges: func(ctx *graphql.FieldContext, after, before interface{}, limit int) (edgeSlice interface{}, cursorLess func(a, b interface{}) bool, err error) {
			return Go(ctx.Context, func() (interface{}, error) {
					return make([]int, limit), nil
				}), func(a, b interface{}) bool {
					return false
				}, nil
		},
		ResolveTotalCount: func(ctx *graphql.FieldContext) (interface{}, error) {
			return 1000, nil
		},
		CursorType: reflect.TypeOf(""),
		EdgeCursor: func(edge interface{}) interface{} {
			return strconv.Itoa(edge.(int))
		},
		EdgeFields: map[string]*graphql.FieldDefinition{
			"node": &graphql.FieldDefinition{
				Type: graphql.IntType,
				Resolve: func(ctx *graphql.FieldContext) (interface{}, error) {
					return ctx.Object, nil
				},
			},
		},
	}))

	api, err := NewAPI(config)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{
		connection(first: 0) {
			edges {
				node
			}
			totalCount
			pageInfo {
				hasNextPage
				startCursor
				endCursor
			}
		}
	}`))
	req.Header.Set("Content-Type", "application/graphql")
	w := httptest.NewRecorder()

	api.ServeGraphQL(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	assert.JSONEq(t, `{
		"data": {
			"connection": {
				"edges": [],
				"pageInfo": {
					"endCursor": "",
					"hasNextPage": true,
					"startCursor": ""
				},
				"totalCount": 1000
			}
		}
	}`, string(body))
}

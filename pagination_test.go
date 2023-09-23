package apifu

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql"
)

func TestConnection(t *testing.T) {
	config := &Config{}

	connectionInterface := ConnectionInterface(&ConnectionInterfaceConfig{
		NamePrefix: "TestInterface",
		EdgeFields: map[string]*graphql.FieldDefinition{
			"node": {
				Type: graphql.IntType,
				Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
					return ctx.Object, nil
				},
			},
		},
		HasTotalCount: true,
	})

	config.AddQueryField("connection", Connection(&ConnectionConfig{
		NamePrefix: "Test",
		ResolveEdges: func(ctx graphql.FieldContext, after, before interface{}, limit int) (edgeSlice interface{}, cursorLess func(a, b interface{}) bool, err error) {
			ret := make([]int, limit)
			for i := range ret {
				ret[i] = i
			}
			return ret, func(a, b interface{}) bool {
				return false
			}, nil
		},
		ResolveTotalCount: func(ctx graphql.FieldContext) (interface{}, error) {
			return 1000, nil
		},
		CursorType: reflect.TypeOf(""),
		EdgeCursor: func(edge interface{}) interface{} {
			return strconv.Itoa(edge.(int))
		},
		EdgeFields: map[string]*graphql.FieldDefinition{
			"node": {
				Type: graphql.IntType,
				Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
					return ctx.Object, nil
				},
			},
		},
		ImplementedInterfaces: []*graphql.InterfaceType{connectionInterface},
	}))

	api, err := NewAPI(config)
	require.NoError(t, err)

	t.Run("Cost", func(t *testing.T) {
		var cost int
		_, errs := graphql.ParseAndValidate(`
		{
			connection(first: 10) {
				...connectionFields
			}
		}

		fragment connectionFields on TestInterfaceConnection {
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
	`, api.schema, graphql.ValidateCost("", nil, -1, &cost, graphql.FieldCost{Resolver: 1}))
		require.Empty(t, errs)
		assert.Equal(t, (1 /*connection*/)+(10 /* edges */)*(1 /* node */)+(1 /*totalCount*/), cost)
	})

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
		ResolveEdges: func(ctx graphql.FieldContext, after, before interface{}, limit int) (edgeSlice interface{}, cursorLess func(a, b interface{}) bool, err error) {
			return nil, nil, fmt.Errorf("the edge resolver should not be invoked")
		},
		ResolveTotalCount: func(ctx graphql.FieldContext) (interface{}, error) {
			return 1000, nil
		},
		CursorType: reflect.TypeOf(""),
		EdgeCursor: func(edge interface{}) interface{} {
			return strconv.Itoa(edge.(int))
		},
		EdgeFields: map[string]*graphql.FieldDefinition{
			"node": {
				Type: graphql.IntType,
				Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
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
		ResolveEdges: func(ctx graphql.FieldContext, after, before interface{}, limit int) (edgeSlice interface{}, cursorLess func(a, b interface{}) bool, err error) {
			return Go(ctx.Context, func() (interface{}, error) {
					return make([]int, limit), nil
				}), func(a, b interface{}) bool {
					return false
				}, nil
		},
		ResolveTotalCount: func(ctx graphql.FieldContext) (interface{}, error) {
			return 1000, nil
		},
		CursorType: reflect.TypeOf(""),
		EdgeCursor: func(edge interface{}) interface{} {
			return strconv.Itoa(edge.(int))
		},
		EdgeFields: map[string]*graphql.FieldDefinition{
			"node": {
				Type: graphql.IntType,
				Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
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

func TestTimeBasedConnection(t *testing.T) {
	edges := make([]time.Time, 10)
	for i := range edges {
		edges[i] = time.Date(2020, time.January, 01, 0, 0, i, 0, time.UTC)
	}

	config := &Config{}
	config.AddQueryField("connection", TimeBasedConnection(&TimeBasedConnectionConfig{
		NamePrefix:  "Test",
		Description: "Test",
		Arguments: map[string]*graphql.InputValueDefinition{
			"async": &graphql.InputValueDefinition{
				Type: graphql.BooleanType,
			},
		},
		EdgeGetter: func(ctx graphql.FieldContext, minTime time.Time, maxTime time.Time, limit int) (interface{}, error) {
			if limit == 0 {
				return nil, nil
			}
			var ret []time.Time
			for _, edge := range edges {
				if !edge.Before(minTime) && !edge.After(maxTime) {
					ret = append(ret, edge)
				}
			}
			if async, ok := ctx.Arguments["async"].(bool); ok && async {
				return Go(ctx.Context, func() (interface{}, error) {
					return ret, nil
				}), nil
			}
			return ret, nil
		},
		EdgeCursor: func(edge interface{}) TimeBasedCursor {
			return NewTimeBasedCursor(edge.(time.Time), "")
		},
		EdgeFields: map[string]*graphql.FieldDefinition{
			"node": {
				Type: DateTimeType,
				Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
					return ctx.Object, nil
				},
			},
		},
	}))

	api, err := NewAPI(config)
	require.NoError(t, err)

	for name, tc := range map[string]struct {
		Query        string
		ExpectedJSON string
	}{
		"All": {
			Query: `{
				connection(first: 100) {
					edges {
						node
					}
				}
			}`,
			ExpectedJSON: `{
				"data":{
					"connection":{
						"edges":[
							{"node":"2020-01-01T00:00:00Z"},
							{"node":"2020-01-01T00:00:01Z"},
							{"node":"2020-01-01T00:00:02Z"},
							{"node":"2020-01-01T00:00:03Z"},
							{"node":"2020-01-01T00:00:04Z"},
							{"node":"2020-01-01T00:00:05Z"},
							{"node":"2020-01-01T00:00:06Z"},
							{"node":"2020-01-01T00:00:07Z"},
							{"node":"2020-01-01T00:00:08Z"},
							{"node":"2020-01-01T00:00:09Z"}
						]
					}
				}
			}`,
		},
		"AllAsync": {
			Query: `{
				connection(first: 100, async: true) {
					edges {
						node
					}
				}
			}`,
			ExpectedJSON: `{
				"data":{
					"connection":{
						"edges":[
							{"node":"2020-01-01T00:00:00Z"},
							{"node":"2020-01-01T00:00:01Z"},
							{"node":"2020-01-01T00:00:02Z"},
							{"node":"2020-01-01T00:00:03Z"},
							{"node":"2020-01-01T00:00:04Z"},
							{"node":"2020-01-01T00:00:05Z"},
							{"node":"2020-01-01T00:00:06Z"},
							{"node":"2020-01-01T00:00:07Z"},
							{"node":"2020-01-01T00:00:08Z"},
							{"node":"2020-01-01T00:00:09Z"}
						]
					}
				}
			}`,
		},
		"AtOrAfterTime": {
			Query: `{
				connection(first: 100, atOrAfterTime: "2020-01-01T00:00:05Z") {
					edges {
						node
					}
				}
			}`,
			ExpectedJSON: `{
				"data":{
					"connection":{
						"edges":[
							{"node":"2020-01-01T00:00:05Z"},
							{"node":"2020-01-01T00:00:06Z"},
							{"node":"2020-01-01T00:00:07Z"},
							{"node":"2020-01-01T00:00:08Z"},
							{"node":"2020-01-01T00:00:09Z"}
						]
					}
				}
			}`,
		},
		"BeforeTime": {
			Query: `{
				connection(first: 100, beforeTime: "2020-01-01T00:00:05Z") {
					edges {
						node
					}
				}
			}`,
			ExpectedJSON: `{
				"data":{
					"connection":{
						"edges":[
							{"node":"2020-01-01T00:00:00Z"},
							{"node":"2020-01-01T00:00:01Z"},
							{"node":"2020-01-01T00:00:02Z"},
							{"node":"2020-01-01T00:00:03Z"},
							{"node":"2020-01-01T00:00:04Z"}
						]
					}
				}
			}`,
		},
		"After": {
			Query: `{
				connection(first: 2, after: "gqROYW5v0xXlmjan9SgAoklkoA") {
					edges {
						node
					}
				}
			}`,
			ExpectedJSON: `{
				"data":{
					"connection":{
						"edges":[
							{"node":"2020-01-01T00:00:05Z"},
							{"node":"2020-01-01T00:00:06Z"}
						]
					}
				}
			}`,
		},
		"Before": {
			Query: `{
				connection(last: 2, before: "gqROYW5v0xXlmjan9SgAoklkoA") {
					edges {
						node
					}
				}
			}`,
			ExpectedJSON: `{
				"data":{
					"connection":{
						"edges":[
							{"node":"2020-01-01T00:00:02Z"},
							{"node":"2020-01-01T00:00:03Z"}
						]
					}
				}
			}`,
		},
		"Empty": {
			Query: `{
				connection(last: 0, before: "") {
					edges {
						node
					}
				}
			}`,
			ExpectedJSON: `{
				"data":{
					"connection":{
						"edges":[]
					}
				}
			}`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tc.Query))
			req.Header.Set("Content-Type", "application/graphql")
			w := httptest.NewRecorder()

			api.ServeGraphQL(w, req)

			resp := w.Result()
			body, _ := ioutil.ReadAll(resp.Body)

			assert.JSONEq(t, tc.ExpectedJSON, string(body))
		})
	}
}

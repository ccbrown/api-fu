package jsonapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/jsonapi/types"
)

var testSchema *Schema

func init() {
	if s, err := NewSchema(&SchemaDefinition{
		ResourceTypes: map[string]AnyResourceType{
			"articles": ResourceType[struct{}]{
				Attributes: map[string]*AttributeDefinition[struct{}]{
					"title": {
						Resolver: ConstantString[struct{}]("JSON:API paints my bikeshed!"),
					},
				},
				Relationships: map[string]*RelationshipDefinition[struct{}]{
					"author": {
						Resolver: ToOneRelationshipResolver[struct{}]{
							ResolveByDefault: true,
							Resolve: func(ctx context.Context, resource struct{}) (*types.ResourceId, *types.Error) {
								return &types.ResourceId{
									Type: "people",
									Id:   "9",
								}, nil
							},
						},
					},
					"comments": {
						Resolver: ToManyRelationshipResolver[struct{}]{
							Resolve: func(ctx context.Context, resource struct{}) ([]types.ResourceId, *types.Error) {
								return []types.ResourceId{
									{
										Type: "comments",
										Id:   "5",
									},
									{
										Type: "comments",
										Id:   "12",
									},
								}, nil
							},
						},
					},
				},
				Get: func(ctx context.Context, id string) (struct{}, *types.Error) {
					if id == "make-error" {
						return struct{}{}, &types.Error{
							Title:  "Error!",
							Status: "400",
						}
					}
					return struct{}{}, nil
				},
			},
			"comments": ResourceType[struct{}]{
				Get: func(ctx context.Context, id string) (struct{}, *types.Error) {
					return struct{}{}, nil
				},
				Delete: func(ctx context.Context, id string) *types.Error {
					return nil
				},
			},
			"people": ResourceType[struct{}]{
				Attributes: map[string]*AttributeDefinition[struct{}]{
					"firstName": {
						Resolver: ConstantString[struct{}]("Dan"),
					},
					"lastName": {
						Resolver: ConstantString[struct{}]("Gebhardt"),
					},
					"twitter": {
						Resolver: ConstantString[struct{}]("dgeb"),
					},
				},
				Get: func(ctx context.Context, id string) (struct{}, *types.Error) {
					return struct{}{}, nil
				},
				Patch: func(ctx context.Context, id string, attributes map[string]json.RawMessage, relationships map[string]any) (struct{}, *types.Error) {
					return struct{}{}, nil
				},
			},
		},
	}); err != nil {
		panic(err)
	} else {
		testSchema = s
	}
}

func TestNoAcceptHeader(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/articles/1", nil)
	require.NoError(t, err)
	API{Schema: testSchema}.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusNotAcceptable, resp.StatusCode)
}

func TestUnsupportedExtension(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/articles/1", nil)
	require.NoError(t, err)
	r.Header.Set("Accept", `application/vnd.api+json; ext="https://jsonapi.org/ext/version"`)
	API{Schema: testSchema}.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusNotAcceptable, resp.StatusCode)
}

func TestMultipleAcceptHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/articles/1", nil)
	require.NoError(t, err)
	r.Header.Add("Accept", `application/vnd.api+json; ext="https://jsonapi.org/ext/version"`)
	r.Header.Add("Accept", `application/foo`)
	r.Header.Add("Accept", `application/vnd.api+json`)
	API{Schema: testSchema}.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestNonsensePath(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/asdlkjqweqwe/asdoijqweoi/qwe", nil)
	require.NoError(t, err)
	r.Header.Set("Accept", "application/vnd.api+json")
	API{Schema: testSchema}.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestQueryParameters(t *testing.T) {
	for name, tc := range map[string]struct {
		Query string
		Okay  bool
	}{
		"Reserved":              {Query: "foo=bar"},
		"Extension":             {Query: "foo:bar=bar"},
		"InvalidChars":          {Query: "aa123@!@#$=bar"},
		"UnbalancedBracket":     {Query: "foo[asd=bar"},
		"Pagination":            {Query: "page[size]=foo", Okay: true},
		"ImplementationDefined": {Query: "Foo=bar&Foo[bar]=baz", Okay: true},
	} {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r, err := http.NewRequest("GET", "/articles/1?"+tc.Query, nil)
			require.NoError(t, err)
			r.Header.Set("Accept", "application/vnd.api+json")
			API{Schema: testSchema}.ServeHTTP(w, r)
			resp := w.Result()
			if tc.Okay {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			} else {
				assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
			}
		})
	}
}

func TestGetResource_Error(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/articles/make-error", nil)
	require.NoError(t, err)
	r.Header.Set("Accept", "application/vnd.api+json")
	API{Schema: testSchema}.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.JSONEq(t, `{
	  "errors": [{
		"title": "Error!",
		"status": "400"
	  }],
	  "jsonapi": {
	  	"version": "1.1"
	  }
	}`, string(body))
}

func TestGetResource(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/articles/1", nil)
	require.NoError(t, err)
	r.Header.Set("Accept", "application/vnd.api+json")
	API{Schema: testSchema}.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.JSONEq(t, `{
	  "links": {
		"self": "/articles/1"
	  },
	  "data": {
		"type": "articles",
		"id": "1",
		"attributes": {
		  "title": "JSON:API paints my bikeshed!"
		},
		"relationships": {
		  "author": {
			"links": {
			  "self": "/articles/1/relationships/author",
			  "related": "/articles/1/author"
			},
			"data": { "type": "people", "id": "9" }
		  },
		  "comments": {
			"links": {
			  "self": "/articles/1/relationships/comments",
			  "related": "/articles/1/comments"
			}
		  }
		}
	  },
	  "jsonapi": {
	  	"version": "1.1"
	  }
	}`, string(body))
}

func TestGetResourceRelationship(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/articles/1/relationships/author", nil)
	require.NoError(t, err)
	r.Header.Set("Accept", "application/vnd.api+json")
	API{Schema: testSchema}.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.JSONEq(t, `{
	  "links": {
		"self": "/articles/1/relationships/author",
		"related": "/articles/1/author"
	  },
	  "data": { "type": "people", "id": "9" },
	  "jsonapi": {
	  	"version": "1.1"
	  }
	}`, string(body))
}

func TestGetRelatedResource(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/articles/1/author", nil)
	require.NoError(t, err)
	r.Header.Set("Accept", "application/vnd.api+json")
	API{Schema: testSchema}.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.JSONEq(t, `{
	  "links": {
		"self": "/articles/1/author"
	  },
	  "data": {
		"type": "people",
		"id": "9",
		"attributes": {
		  "firstName": "Dan",
		  "lastName": "Gebhardt",
		  "twitter": "dgeb"
		}
	  },
	  "jsonapi": {
	  	"version": "1.1"
	  }
	}`, string(body))
}

func TestDelete(t *testing.T) {
	for name, tc := range map[string]struct {
		Path           string
		ExpectedStatus int
	}{
		"Okay": {
			Path:           "/comments/1",
			ExpectedStatus: http.StatusOK,
		},
		"Unsupported": {
			Path:           "/people/1",
			ExpectedStatus: http.StatusMethodNotAllowed,
		},
	} {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r, err := http.NewRequest("DELETE", tc.Path, nil)
			require.NoError(t, err)
			r.Header.Set("Accept", "application/vnd.api+json")
			API{Schema: testSchema}.ServeHTTP(w, r)
			resp := w.Result()
			assert.Equal(t, tc.ExpectedStatus, resp.StatusCode)
			if tc.ExpectedStatus == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				assert.JSONEq(t, `{
				  "jsonapi": {
					"version": "1.1"
				  }
				}`, string(body))
			}
		})
	}
}

func TestPatch(t *testing.T) {
	for name, tc := range map[string]struct {
		Path           string
		ExpectedStatus int
	}{
		"Okay": {
			Path:           "/people/9",
			ExpectedStatus: http.StatusOK,
		},
		"OkayRelatedResource": {
			Path:           "/articles/1/author",
			ExpectedStatus: http.StatusOK,
		},
		"Mismatch": {
			Path:           "/comments/1",
			ExpectedStatus: http.StatusConflict,
		},
	} {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r, err := http.NewRequest("PATCH", tc.Path, strings.NewReader(`{
				"data": {
					"type": "people",
					"id": "9",
					"attributes": {
						"firstName": "Dan"
					}
				}
			}`))
			require.NoError(t, err)
			r.Header.Set("Accept", "application/vnd.api+json")
			API{Schema: testSchema}.ServeHTTP(w, r)
			resp := w.Result()
			assert.Equal(t, tc.ExpectedStatus, resp.StatusCode)
			if tc.ExpectedStatus == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				assert.JSONEq(t, `{
				  "links": {
					"self": "`+r.URL.Path+`"
				  },
				  "data": {
					"type": "people",
					"id": "9",
					"attributes": {
					  "firstName": "Dan",
					  "lastName": "Gebhardt",
					  "twitter": "dgeb"
					}
				  },
				  "jsonapi": {
					"version": "1.1"
				  }
				}`, string(body))
			}
		})
	}
}

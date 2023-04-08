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

type Article struct {
	Author   *types.ResourceId
	Comments []types.ResourceId
}

func init() {
	if s, err := NewSchema(&SchemaDefinition{
		ResourceTypes: map[string]AnyResourceType{
			"articles": ResourceType[Article]{
				Attributes: map[string]*AttributeDefinition[Article]{
					"title": {
						Resolver: ConstantString[Article]("JSON:API paints my bikeshed!"),
					},
				},
				Relationships: map[string]*RelationshipDefinition[Article]{
					"author": {
						Resolver: ToOneRelationshipResolver[Article]{
							ResolveByDefault: true,
							Resolve: func(ctx context.Context, resource Article) (*types.ResourceId, *types.Error) {
								return resource.Author, nil
							},
						},
					},
					"comments": {
						Resolver: ToManyRelationshipResolver[Article]{
							Resolve: func(ctx context.Context, resource Article) ([]types.ResourceId, *types.Error) {
								return resource.Comments, nil
							},
						},
					},
				},
				Get: func(ctx context.Context, id string) (Article, *types.Error) {
					if id == "make-error" {
						return Article{}, &types.Error{
							Title:  "Error!",
							Status: "400",
						}
					}

					return Article{
						Author: &types.ResourceId{Type: "people", Id: "9"},
						Comments: []types.ResourceId{
							{
								Type: "comments",
								Id:   "5",
							},
							{
								Type: "comments",
								Id:   "12",
							},
						},
					}, nil
				},
				Patch: func(ctx context.Context, id string, attributes map[string]json.RawMessage, relationships map[string]any) (Article, *types.Error) {
					ret, err := testSchema.resourceTypes["articles"].(ResourceType[Article]).Get(ctx, id)
					if err != nil {
						return Article{}, err
					}

					if _, ok := relationships["author"]; ok {
						switch author := relationships["author"].(type) {
						case types.ResourceId:
							ret.Author = &author
						case nil:
							ret.Author = nil
						}
					}

					if _, ok := relationships["comments"]; ok {
						switch comments := relationships["comments"].(type) {
						case []types.ResourceId:
							ret.Comments = comments
						case nil:
							ret.Comments = nil
						}
					}

					return ret, nil
				},
				AddRelationshipMembers: func(ctx context.Context, id string, relationshipName string, members []types.ResourceId) (Article, *types.Error) {
					ret, err := testSchema.resourceTypes["articles"].(ResourceType[Article]).Get(ctx, id)
					if err != nil {
						return Article{}, err
					}

					switch relationshipName {
					case "comments":
						existing := map[types.ResourceId]struct{}{}
						for _, comment := range ret.Comments {
							existing[comment] = struct{}{}
						}
						for _, member := range members {
							if _, ok := existing[member]; !ok {
								ret.Comments = append(ret.Comments, member)
								existing[member] = struct{}{}
							}
						}
					default:
						return Article{}, &types.Error{
							Title:  "Invalid relationship",
							Status: "400",
						}
					}

					return ret, nil
				},
				RemoveRelationshipMembers: func(ctx context.Context, id string, relationshipName string, members []types.ResourceId) (Article, *types.Error) {
					ret, err := testSchema.resourceTypes["articles"].(ResourceType[Article]).Get(ctx, id)
					if err != nil {
						return Article{}, err
					}

					switch relationshipName {
					case "comments":
						toRemove := map[types.ResourceId]struct{}{}
						for _, member := range members {
							toRemove[member] = struct{}{}
						}
						var newComments []types.ResourceId
						for _, comment := range ret.Comments {
							if _, ok := toRemove[comment]; !ok {
								newComments = append(newComments, comment)
							}
						}
						ret.Comments = newComments
					default:
						return Article{}, &types.Error{
							Title:  "Invalid relationship",
							Status: "400",
						}
					}

					return ret, nil
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

func TestPatchRelationship(t *testing.T) {
	for name, tc := range map[string]struct {
		Path             string
		Body             string
		ExpectedStatus   int
		ExpectedResponse string
	}{
		"NewAuthor": {
			Path:           "/articles/1/relationships/author",
			Body:           `{"data": {"type": "people", "id": "12"}}`,
			ExpectedStatus: http.StatusOK,
			ExpectedResponse: `{
			  "links": {
				"self": "/articles/1/relationships/author",
			  	"related": "/articles/1/author"
			  },
			  "data": {
				"type": "people",
				"id": "12"
			  },
			  "jsonapi": {
				"version": "1.1"
			  }
			}`,
		},
		"ClearAuthor": {
			Path:           "/articles/1/relationships/author",
			Body:           `{"data": null}`,
			ExpectedStatus: http.StatusOK,
			ExpectedResponse: `{
			  "links": {
				"self": "/articles/1/relationships/author",
			  	"related": "/articles/1/author"
			  },
			  "data": null,
			  "jsonapi": {
				"version": "1.1"
			  }
			}`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r, err := http.NewRequest("PATCH", tc.Path, strings.NewReader(tc.Body))
			require.NoError(t, err)
			r.Header.Set("Accept", "application/vnd.api+json")
			API{Schema: testSchema}.ServeHTTP(w, r)
			resp := w.Result()
			assert.Equal(t, tc.ExpectedStatus, resp.StatusCode)
			if tc.ExpectedStatus == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				assert.JSONEq(t, tc.ExpectedResponse, string(body))
			}
		})
	}
}

func TestPostRelationship(t *testing.T) {
	for name, tc := range map[string]struct {
		Path             string
		Body             string
		ExpectedStatus   int
		ExpectedResponse string
	}{
		"AddComments": {
			Path: "/articles/1/relationships/comments",
			Body: `{
			  "data": [
				{ "type": "comments", "id": "12" },
				{ "type": "comments", "id": "13" }
			  ]
			}`,
			ExpectedStatus: http.StatusOK,
			ExpectedResponse: `{
			  "links": {
				"self": "/articles/1/relationships/comments",
			  	"related": "/articles/1/comments"
			  },
			  "data": [
				{ "type": "comments", "id": "5" },
				{ "type": "comments", "id": "12" },
				{ "type": "comments", "id": "13" }
			  ],
			  "jsonapi": {
				"version": "1.1"
			  }
			}`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r, err := http.NewRequest("POST", tc.Path, strings.NewReader(tc.Body))
			require.NoError(t, err)
			r.Header.Set("Accept", "application/vnd.api+json")
			API{Schema: testSchema}.ServeHTTP(w, r)
			resp := w.Result()
			assert.Equal(t, tc.ExpectedStatus, resp.StatusCode)
			if tc.ExpectedStatus == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				assert.JSONEq(t, tc.ExpectedResponse, string(body))
			}
		})
	}
}

func TestDeleteRelationship(t *testing.T) {
	for name, tc := range map[string]struct {
		Path             string
		Body             string
		ExpectedStatus   int
		ExpectedResponse string
	}{
		"AddComments": {
			Path: "/articles/1/relationships/comments",
			Body: `{
			  "data": [
				{ "type": "comments", "id": "12" },
				{ "type": "comments", "id": "13" }
			  ]
			}`,
			ExpectedStatus: http.StatusOK,
			ExpectedResponse: `{
			  "links": {
				"self": "/articles/1/relationships/comments",
			  	"related": "/articles/1/comments"
			  },
			  "data": [
				{ "type": "comments", "id": "5" }
			  ],
			  "jsonapi": {
				"version": "1.1"
			  }
			}`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r, err := http.NewRequest("DELETE", tc.Path, strings.NewReader(tc.Body))
			require.NoError(t, err)
			r.Header.Set("Accept", "application/vnd.api+json")
			API{Schema: testSchema}.ServeHTTP(w, r)
			resp := w.Result()
			assert.Equal(t, tc.ExpectedStatus, resp.StatusCode)
			if tc.ExpectedStatus == http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				assert.JSONEq(t, tc.ExpectedResponse, string(body))
			}
		})
	}
}

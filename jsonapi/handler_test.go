package jsonapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
							Resolve: func(ctx context.Context, resource struct{}) (*ResourceId, *Error) {
								return &ResourceId{
									Type: "people",
									Id:   "9",
								}, nil
							},
						},
					},
					"comments": {
						Resolver: ToManyRelationshipResolver[struct{}]{
							Resolve: func(ctx context.Context, resource struct{}) ([]ResourceId, *Error) {
								return []ResourceId{
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
				Getter: func(ctx context.Context, id string) (struct{}, *Error) {
					if id == "make-error" {
						return struct{}{}, &Error{
							Title:  "Error!",
							Status: "400",
						}
					}
					return struct{}{}, nil
				},
			},
			"comments": ResourceType[struct{}]{
				Getter: func(ctx context.Context, id string) (struct{}, *Error) {
					return struct{}{}, nil
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
				Getter: func(ctx context.Context, id string) (struct{}, *Error) {
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

func TestUnsupportedQueryParameter(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/articles/1?foo=bar", nil)
	require.NoError(t, err)
	r.Header.Set("Accept", "application/vnd.api+json")
	API{Schema: testSchema}.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
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
			},
			"data": [
			  { "type": "comments", "id": "5" },
			  { "type": "comments", "id": "12" }
			]
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

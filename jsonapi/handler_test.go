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

func TestGetResource_Error(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "http://example.com/articles/make-error", nil)
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
	  }]
	}`, string(body))
}

func TestGetResource(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "http://example.com/articles/1", nil)
	require.NoError(t, err)
	r.Header.Set("Accept", "application/vnd.api+json")
	API{Schema: testSchema}.ServeHTTP(w, r)
	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.JSONEq(t, `{
	  "links": {
		"self": "http://example.com/articles/1"
	  },
	  "data": {
		"type": "articles",
		"id": "1",
		"attributes": {
		  "title": "JSON:API paints my bikeshed!"
		}
	  }
	}`, string(body))
}

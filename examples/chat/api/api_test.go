package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ccbrown/keyvaluestore/memorystore"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/examples/chat/app"
	"github.com/ccbrown/api-fu/examples/chat/store"
	"github.com/ccbrown/api-fu/graphql"
)

func NewTestAPI() *API {
	return &API{
		App: &app.App{
			Store: &store.Store{
				Backend: memorystore.NewBackend(),
			},
		},
	}
}

func (api *API) execGraphQL(t *testing.T, query string, variables map[string]interface{}) *graphql.Response {
	body, err := json.Marshal(struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}{
		Query:     query,
		Variables: variables,
	})
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("POST", "http://example.com/graphql", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)
	api.ServeGraphQL(w, r)

	require.Equal(t, http.StatusOK, w.Result().StatusCode)

	var resp graphql.Response
	require.NoError(t, json.NewDecoder(w.Result().Body).Decode(&resp))
	return &resp
}

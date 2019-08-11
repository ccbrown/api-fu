package apifu

import (
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/pkg/errors"

	"github.com/ccbrown/api-fu/graphql"
)

type API struct {
	schema *graphql.Schema
}

func normalizeModel(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func NewAPI(cfg *Config) (*API, error) {
	schema, err := cfg.graphqlSchema()
	if err != nil {
		return nil, errors.Wrap(err, "error building graphql schema")
	}
	return &API{
		schema: schema,
	}, nil
}

func (api *API) ServeGraphQL(w http.ResponseWriter, r *http.Request) {
	req := &graphql.Request{
		Schema: api.schema,
	}

	switch r.Method {
	case http.MethodGet:
		query := r.URL.Query().Get("query")
		if query == "" {
			http.Error(w, "a query is required", http.StatusBadRequest)
			return
		}
		req.Query = query
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(graphql.Execute(req))
}

package apifu

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"

	"github.com/pkg/errors"

	"github.com/ccbrown/api-fu/graphql"
)

type API struct {
	schema *graphql.Schema
	config *Config
}

func normalizeModelType(t reflect.Type) reflect.Type {
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
		config: cfg,
		schema: schema,
	}, nil
}

type apiContextKeyType int

var apiContextKey apiContextKeyType

func ctxAPI(ctx context.Context) *API {
	return ctx.Value(apiContextKey).(*API)
}

func (api *API) ServeGraphQL(w http.ResponseWriter, r *http.Request) {
	r = r.WithContext(context.WithValue(r.Context(), apiContextKey, api))

	req, err, code := graphql.NewRequestFromHTTP(r)
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}
	req.Schema = api.schema

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(graphql.Execute(req))
}

func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface) && rv.IsNil()
}

func (api *API) resolveNodeByGlobalId(ctx context.Context, id string) (interface{}, error) {
	typeId, modelId := api.config.DeserializeNodeId(id)
	nodeType, ok := api.config.nodeTypesById[typeId]
	if !ok {
		return nil, nil
	}
	return api.resolveNodeById(ctx, nodeType, modelId)
}

func (api *API) resolveNodeById(ctx context.Context, nodeType *NodeType, modelId interface{}) (interface{}, error) {
	// TODO: batching and concurrency

	ids := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(modelId)), 1, 1)
	ids.Index(0).Set(reflect.ValueOf(modelId))
	nodes, err := nodeType.GetByIds(ctx, ids.Interface())
	if !isNil(err) {
		return nil, err
	}
	nodesValue := reflect.ValueOf(nodes)
	if nodesValue.Len() < 1 {
		return nil, nil
	}
	return nodesValue.Index(0).Interface(), nil
}

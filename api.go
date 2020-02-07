package apifu

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ccbrown/api-fu/graphql"
	"github.com/ccbrown/api-fu/graphqlws"
)

type API struct {
	schema *graphql.Schema
	config *Config
	logger logrus.FieldLogger

	graphqlWSConnectionsMutex sync.Mutex
	graphqlWSConnections      map[*graphqlws.Connection]struct{}
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
	logger := cfg.Logger
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &API{
		config:               cfg,
		schema:               schema,
		logger:               logger,
		graphqlWSConnections: map[*graphqlws.Connection]struct{}{},
	}, nil
}

type apiContextKeyType int

var apiContextKey apiContextKeyType

func ctxAPI(ctx context.Context) *API {
	return ctx.Value(apiContextKey).(*API)
}

type asyncResolution struct {
	Result graphql.ResolveResult
	Dest   graphql.ResolvePromise
}

type apiRequest struct {
	asyncResolutions        chan asyncResolution
	chainedAsyncResolutions map[graphql.ResolvePromise]struct{}
}

func (r *apiRequest) IdleHandler() {
	for {
		resolution := <-r.asyncResolutions
		resolution.Dest <- resolution.Result
		if _, ok := r.chainedAsyncResolutions[resolution.Dest]; ok {
			delete(r.chainedAsyncResolutions, resolution.Dest)
			continue
		}
		for {
			select {
			case resolution := <-r.asyncResolutions:
				resolution.Dest <- resolution.Result
			default:
				return
			}
		}
	}
}

type apiRequestContextKeyType int

var apiRequestContextKey apiRequestContextKeyType

func ctxAPIRequest(ctx context.Context) *apiRequest {
	return ctx.Value(apiRequestContextKey).(*apiRequest)
}

func chain(ctx context.Context, p graphql.ResolvePromise, f func(interface{}) (interface{}, error)) graphql.ResolvePromise {
	apiRequest := ctxAPIRequest(ctx)
	if apiRequest.chainedAsyncResolutions == nil {
		apiRequest.chainedAsyncResolutions = map[graphql.ResolvePromise]struct{}{}
	}
	apiRequest.chainedAsyncResolutions[p] = struct{}{}
	return Go(ctx, func() (interface{}, error) {
		result := <-p
		if !isNil(result.Error) {
			return nil, result.Error
		}
		return f(result.Value)
	})
}

// When used within the context of a resolve function, completes resolution asynchronously and
// concurrently with any other asynchronous resolutions.
func Go(ctx context.Context, f func() (interface{}, error)) graphql.ResolvePromise {
	apiRequest := ctxAPIRequest(ctx)
	if apiRequest.asyncResolutions == nil {
		apiRequest.asyncResolutions = make(chan asyncResolution)
	}
	ch := make(graphql.ResolvePromise, 1)
	go func() {
		v, err := f()
		apiRequest.asyncResolutions <- asyncResolution{
			Result: graphql.ResolveResult{
				Value: v,
				Error: err,
			},
			Dest: ch,
		}
	}()
	return ch
}

func (api *API) ServeGraphQL(w http.ResponseWriter, r *http.Request) {
	ctx := context.WithValue(r.Context(), apiContextKey, api)
	apiRequest := &apiRequest{}
	ctx = context.WithValue(ctx, apiRequestContextKey, apiRequest)
	r = r.WithContext(ctx)

	req, code, err := graphql.NewRequestFromHTTP(r)
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}
	req.Schema = api.schema
	req.IdleHandler = apiRequest.IdleHandler

	body, err := json.Marshal(graphql.Execute(req))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.Write(body)
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

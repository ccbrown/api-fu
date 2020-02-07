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
	"github.com/ccbrown/api-fu/graphql/executor"
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
	Result executor.ResolveResult
	Dest   executor.ResolvePromise
}

type apiRequest struct {
	asyncResolutions chan asyncResolution
}

func (r *apiRequest) IdleHandler() {
	resolution := <-r.asyncResolutions
	resolution.Dest <- resolution.Result
	for {
		select {
		case resolution := <-r.asyncResolutions:
			resolution.Dest <- resolution.Result
		default:
			return
		}
	}
}

type apiRequestContextKeyType int

var apiRequestContextKey apiRequestContextKeyType

func ctxAPIRequest(ctx context.Context) *apiRequest {
	return ctx.Value(apiRequestContextKey).(*apiRequest)
}

// Async causes the given resolver to be executed within a new goroutine. It will be executed
// concurrently with other asynchronous resolvers if possible.
func Async(resolve func(ctx *graphql.FieldContext) (interface{}, error)) func(ctx *graphql.FieldContext) (interface{}, error) {
	return func(ctx *graphql.FieldContext) (interface{}, error) {
		apiRequest := ctxAPIRequest(ctx.Context)
		if apiRequest.asyncResolutions == nil {
			apiRequest.asyncResolutions = make(chan asyncResolution)
		}
		ch := make(executor.ResolvePromise, 1)
		go func() {
			v, err := resolve(ctx)
			result := executor.ResolveResult{
				Value: v,
				Error: err,
			}
			apiRequest.asyncResolutions <- asyncResolution{
				Result: result,
				Dest:   ch,
			}
		}()
		return ch, nil
	}
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

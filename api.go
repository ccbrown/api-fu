package apifu

import (
	"context"
	"net/http"
	"reflect"
	"strconv"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ccbrown/api-fu/graphql"
	"github.com/ccbrown/api-fu/graphqlws"
)

// API is responsible for serving your API traffic. Construct an API by creating a Config, then
// calling NewAPI.
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
	batches                 map[*int]*batch
}

func (r *apiRequest) IdleHandler() {
	for {
		if len(r.batches) > 0 {
			// Go ahead and resolve all the batches.
			var wg sync.WaitGroup
			for _, b := range r.batches {
				wg.Add(1)
				b := b
				go func() {
					defer wg.Done()
					for i, result := range b.resolver(b.items) {
						b.dests[i] <- result
					}
				}()
			}
			wg.Wait()
			r.batches = map[*int]*batch{}
		} else {
			// Block until we've fully resolved something.
			resolution := <-r.asyncResolutions
			resolution.Dest <- resolution.Result
			if _, ok := r.chainedAsyncResolutions[resolution.Dest]; ok {
				delete(r.chainedAsyncResolutions, resolution.Dest)
				continue
			}
		}

		// Move along anything else that we happen to also be done resolving.
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

// Go completes resolution asynchronously and concurrently with any other asynchronous resolutions.
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

type batch struct {
	resolver func([]*graphql.FieldContext) []graphql.ResolveResult
	items    []*graphql.FieldContext
	dests    []chan graphql.ResolveResult
}

// Batch batches up the resolver invocations into a single call. As queries are executed, whenever
// resolution gets "stuck", all pending batch resolvers will be triggered concurrently. Batch
// resolvers must return one result for every field context it receives.
func Batch(f func([]*graphql.FieldContext) []graphql.ResolveResult) func(*graphql.FieldContext) (interface{}, error) {
	var x int
	key := &x
	return func(ctx *graphql.FieldContext) (interface{}, error) {
		apiRequest := ctxAPIRequest(ctx.Context)
		b, ok := apiRequest.batches[key]
		if !ok {
			b = &batch{
				resolver: f,
			}
			if apiRequest.batches == nil {
				apiRequest.batches = map[*int]*batch{}
			}
			apiRequest.batches[key] = b
		}
		ch := make(graphql.ResolvePromise, 1)
		b.items = append(b.items, ctx)
		b.dests = append(b.dests, ch)
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

	body, err := jsoniter.Marshal(graphql.Execute(req))
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

func (api *API) resolveNodesByGlobalIds(ctx context.Context, ids []string) ([]interface{}, error) {
	modelIds := map[int][]interface{}{}
	for _, id := range ids {
		typeId, modelId := api.config.DeserializeNodeId(id)
		modelIds[typeId] = append(modelIds[typeId], modelId)
	}
	var ret []interface{}
	for typeId, modelIds := range modelIds {
		nodeType, ok := api.config.nodeTypesById[typeId]
		if !ok {
			continue
		}
		ids := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(modelIds[0])), len(modelIds), len(modelIds))
		for i, modelId := range modelIds {
			ids.Index(i).Set(reflect.ValueOf(modelId))
		}
		nodes, err := nodeType.GetByIds(ctx, ids.Interface())
		if !isNil(err) {
			return nil, err
		}
		nodesValue := reflect.ValueOf(nodes)
		for i := 0; i < nodesValue.Len(); i++ {
			ret = append(ret, nodesValue.Index(i).Interface())
		}
	}
	return ret, nil
}

func (api *API) resolveNodeById(ctx context.Context, nodeType *NodeType, modelId interface{}) (interface{}, error) {
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

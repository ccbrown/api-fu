package apifu

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/ccbrown/api-fu/graphql"
	"github.com/ccbrown/api-fu/graphql/transport/graphqltransportws"
	"github.com/ccbrown/api-fu/graphql/transport/graphqlws"
)

type graphqlWSConnection interface {
	SendData(ctx context.Context, id string, response *graphql.Response) error
	SendComplete(ctx context.Context, id string) error
	Serve(conn *websocket.Conn)
	io.Closer
}

type graphqlWSHandler struct {
	API        *API
	Connection graphqlWSConnection
	Context    context.Context
	Logger     logrus.FieldLogger

	cancelContext func()
	subscriptions map[string]*SubscriptionSourceStream
	features      graphql.FeatureSet
}

func (h *graphqlWSHandler) HandleInit(parameters json.RawMessage) error {
	if f := h.API.config.HandleGraphQLWSInit; f != nil {
		if ctx, err := f(h.Context, parameters); err != nil {
			return err
		} else {
			h.Context = ctx
		}
	}
	if h.API.config.Features != nil {
		h.features = h.API.config.Features(h.Context)
	}
	return nil
}

func (h *graphqlWSHandler) HandleStart(id string, query string, variables map[string]interface{}, operationName string) {
	ctx := context.WithValue(h.Context, apiContextKey, h.API)

	apiRequest := &apiRequest{}
	ctx = context.WithValue(ctx, apiRequestContextKey, apiRequest)

	req := &graphql.Request{
		Context:        ctx,
		Query:          query,
		Schema:         h.API.schema,
		IdleHandler:    apiRequest.IdleHandler,
		Features:       h.features,
		OperationName:  operationName,
		VariableValues: variables,
	}

	var info RequestInfo
	var resp *graphql.Response
	if doc, errs := graphql.ParseAndValidate(req.Query, req.Schema, req.Features, req.ValidateCost(-1, &info.Cost, h.API.config.DefaultFieldCost)); len(errs) > 0 {
		resp = &graphql.Response{
			Errors: errs,
		}
	} else {
		req.Document = doc

		if graphql.IsSubscription(doc, operationName) {
			if _, ok := h.subscriptions[id]; ok {
				// if the subscription already exists, ignore this message. should we do something
				// else though?
				return
			}
			if sourceStream, errs := graphql.Subscribe(req); len(errs) > 0 {
				resp = &graphql.Response{
					Errors: errs,
				}
			} else {
				if h.subscriptions == nil {
					h.subscriptions = map[string]*SubscriptionSourceStream{}
				}
				sourceStream := sourceStream.(*SubscriptionSourceStream)
				h.subscriptions[id] = sourceStream
				go func() {
					// Note we can't use ctx here, because the Go http package closes it after a
					// hijacked connection's handler returns.
					if err := sourceStream.Run(context.Background(), func(event interface{}) {
						req := *req
						req.InitialValue = event
						if err := h.Connection.SendData(context.Background(), id, h.API.execute(&req, &info)); err != nil {
							h.Logger.Warn(errors.Wrap(err, "error sending graphql-ws data"))
						}
					}); err != nil && err != context.Canceled {
						h.Logger.Error(errors.Wrap(err, "error running source stream"))
					}
					if err := h.Connection.SendComplete(context.Background(), id); err != nil {
						h.Logger.Warn(errors.Wrap(err, "error sending graphql-ws complete"))
					}
				}()
			}
		} else {
			resp = h.API.execute(req, &info)
		}
	}

	if resp != nil {
		if err := h.Connection.SendData(context.Background(), id, resp); err != nil {
			h.Logger.Warn(errors.Wrap(err, "error sending graphql-ws data"))
		}
		if err := h.Connection.SendComplete(context.Background(), id); err != nil {
			h.Logger.Warn(errors.Wrap(err, "error sending graphql-ws complete"))
		}
	}
}

func (h *graphqlWSHandler) HandleStop(id string) {
	if stream, ok := h.subscriptions[id]; ok {
		stream.Stop()
		delete(h.subscriptions, id)
	}
}

func (h *graphqlWSHandler) LogError(err error) {
	h.Logger.Error(err)
}

func (h *graphqlWSHandler) Cancel() {
	h.cancelContext()
}

func (h *graphqlWSHandler) HandleClose() {
	for _, stream := range h.subscriptions {
		stream.Stop()
	}
	h.subscriptions = nil

	h.API.graphqlWSConnectionsMutex.Lock()
	defer h.API.graphqlWSConnectionsMutex.Unlock()
	delete(h.API.graphqlWSConnections, h.Connection)
}

// This type is a context which gets values from another context (e.g. a canceled http.Request
// context after a hijacking).
type hijackedContext struct {
	newContext   context.Context
	valueContext context.Context
}

func (ctx hijackedContext) Deadline() (time.Time, bool) {
	return ctx.newContext.Deadline()
}

func (ctx hijackedContext) Done() <-chan struct{} {
	return ctx.newContext.Done()
}

func (ctx hijackedContext) Err() error {
	return ctx.newContext.Err()
}

func (ctx hijackedContext) Value(key any) any {
	return ctx.valueContext.Value(key)
}

// ServeGraphQLWS serves a GraphQL WebSocket connection. It will serve connections for both the
// deprecated graphql-ws subprotocol and the newer graphql-transport-ws subprotocol.
//
// This method hijacks connections. To gracefully close them, use CloseHijackedConnections.
func (api *API) ServeGraphQLWS(w http.ResponseWriter, r *http.Request) {
	if !websocket.IsWebSocketUpgrade(r) {
		http.Error(w, "not a websocket upgrade", http.StatusBadRequest)
		return
	}

	var upgrader = websocket.Upgrader{
		CheckOrigin:       api.config.WebSocketOriginCheck,
		EnableCompression: true,
		Subprotocols:      []string{graphqlws.WebSocketSubprotocol, graphqltransportws.WebSocketSubprotocol},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// We've hijacked the request and can't use its context as it'll be cancelled once this handler
	// returns. We create a new context and cancel it if we detect that the connection is closed.
	ctx, cancel := context.WithCancel(context.Background())

	handler := &graphqlWSHandler{
		API: api,
		Context: hijackedContext{
			newContext:   ctx,
			valueContext: r.Context(),
		},
		Logger:        api.logger,
		cancelContext: cancel,
	}

	var connection graphqlWSConnection
	if conn.Subprotocol() == graphqltransportws.WebSocketSubprotocol {
		connection = &graphqltransportws.Connection{
			Handler: handler,
		}
	} else {
		connection = &graphqlws.Connection{
			Handler: handler,
		}
	}

	handler.Connection = connection

	api.graphqlWSConnectionsMutex.Lock()
	api.graphqlWSConnections[connection] = struct{}{}
	api.graphqlWSConnectionsMutex.Unlock()

	connection.Serve(conn)
}

// CloseHijackedConnections closes connections hijacked by ServeGraphQLWS.
func (api *API) CloseHijackedConnections() error {
	api.graphqlWSConnectionsMutex.Lock()
	connections := make([]graphqlWSConnection, len(api.graphqlWSConnections))
	i := 0
	for connection := range api.graphqlWSConnections {
		connections[i] = connection
		i++
	}
	api.graphqlWSConnections = map[graphqlWSConnection]struct{}{}
	api.graphqlWSConnectionsMutex.Unlock()

	var ret error
	for _, connection := range connections {
		if err := connection.Close(); err != nil {
			ret = multierror.Append(ret, errors.Wrap(err, "error closing connection"))
		}
	}
	return ret
}

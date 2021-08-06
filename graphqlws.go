package apifu

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"github.com/ccbrown/api-fu/graphql"
	"github.com/ccbrown/api-fu/graphqlws"
)

type graphqlWSHandler struct {
	API        *API
	Connection *graphqlws.Connection
	Context    context.Context

	subscriptions map[string]*SubscriptionSourceStream
}

func (h *graphqlWSHandler) HandleInit(parameters json.RawMessage) error {
	if f := h.API.config.HandleGraphQLWSInit; f != nil {
		if ctx, err := f(h.Context, parameters); err != nil {
			return err
		} else {
			h.Context = ctx
		}
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
		OperationName:  operationName,
		VariableValues: variables,
	}

	var info RequestInfo
	var resp *graphql.Response
	if doc, errs := graphql.ParseAndValidate(req.Query, req.Schema, req.ValidateCost(-1, &info.Cost)); len(errs) > 0 {
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
							h.Connection.Logger.Warn(errors.Wrap(err, "error sending graphql-ws data"))
						}
					}); err != nil && err != context.Canceled {
						h.Connection.Logger.Error(errors.Wrap(err, "error running source stream"))
					}
				}()
			}
		} else {
			resp = h.API.execute(req, &info)
		}
	}

	if resp != nil {
		if err := h.Connection.SendData(context.Background(), id, resp); err != nil {
			h.Connection.Logger.Warn(errors.Wrap(err, "error sending graphql-ws data"))
		}
		if err := h.Connection.SendComplete(context.Background(), id); err != nil {
			h.Connection.Logger.Warn(errors.Wrap(err, "error sending graphql-ws complete"))
		}
	}
}

func (h *graphqlWSHandler) HandleStop(id string) {
	if stream, ok := h.subscriptions[id]; ok {
		stream.Stop()
		delete(h.subscriptions, id)
	}
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

// ServeGraphQLWS serves a graphql-ws WebSocket connection. This method hijacks connections. To
// gracefully close them, use CloseHijackedConnections.
func (api *API) ServeGraphQLWS(w http.ResponseWriter, r *http.Request) {
	if !websocket.IsWebSocketUpgrade(r) {
		http.Error(w, "not a websocket upgrade", http.StatusBadRequest)
		return
	}

	var upgrader = websocket.Upgrader{
		CheckOrigin:       api.config.WebSocketOriginCheck,
		EnableCompression: true,
		Subprotocols:      []string{"graphql-ws"},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	connection := &graphqlws.Connection{
		Logger: api.logger,
	}

	api.graphqlWSConnectionsMutex.Lock()
	api.graphqlWSConnections[connection] = struct{}{}
	api.graphqlWSConnectionsMutex.Unlock()

	connection.Handler = &graphqlWSHandler{
		API:        api,
		Connection: connection,
		Context:    r.Context(),
	}
	connection.Serve(conn)
}

// CloseHijackedConnections closes connections hijacked by ServeGraphQLWS.
func (api *API) CloseHijackedConnections() {
	api.graphqlWSConnectionsMutex.Lock()
	connections := make([]*graphqlws.Connection, len(api.graphqlWSConnections))
	i := 0
	for connection := range api.graphqlWSConnections {
		connections[i] = connection
		i++
	}
	api.graphqlWSConnections = map[*graphqlws.Connection]struct{}{}
	api.graphqlWSConnectionsMutex.Unlock()

	for _, connection := range connections {
		if err := connection.Close(); err != nil {
			connection.Logger.Error(errors.Wrap(err, "error closing connection"))
		}
	}
}

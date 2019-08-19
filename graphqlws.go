package apifu

import (
	"context"
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
}

func (h *graphqlWSHandler) HandleStart(id string, query string, variables map[string]interface{}, operationName string) {
	response := graphql.Execute(&graphql.Request{
		Context:        context.WithValue(h.Context, apiContextKey, h.API),
		Query:          query,
		Schema:         h.API.schema,
		OperationName:  operationName,
		VariableValues: variables,
	})
	if err := h.Connection.SendData(id, response); err != nil {
		h.Connection.Logger.Warn(errors.Wrap(err, "error sending graphql-ws data"))
	}
	if err := h.Connection.SendComplete(id); err != nil {
		h.Connection.Logger.Warn(errors.Wrap(err, "error sending graphql-ws complete"))
	}
}

func (h *graphqlWSHandler) HandleStop(id string) {
	// TODO: subscription support
}

func (h *graphqlWSHandler) HandleClose() {
	h.API.graphqlWSConnectionsMutex.Lock()
	defer h.API.graphqlWSConnectionsMutex.Unlock()
	delete(h.API.graphqlWSConnections, h.Connection)
}

// Serves a graphql-ws WebSocket connection. This method hijacks connections. To gracefully close
// them, use CloseHijackedConnections.
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
	connection.Handler = &graphqlWSHandler{
		API:        api,
		Connection: connection,
		Context:    r.Context(),
	}
	connection.Serve(conn)
}

// Closes connections hijacked by ServeGraphQLWS.
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

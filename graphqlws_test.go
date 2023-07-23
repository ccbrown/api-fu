package apifu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ccbrown/api-fu/graphql"
	"github.com/ccbrown/api-fu/graphql/transport/graphqltransportws"
	"github.com/ccbrown/api-fu/graphql/transport/graphqlws"
)

func TestGraphQLWS(t *testing.T) {
	var testCfg Config

	testCfg.AddQueryField("foo", &graphql.FieldDefinition{
		Type: graphql.BooleanType,
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			return true, nil
		},
	})

	testCfg.AddSubscription("time", &graphql.FieldDefinition{
		Type: graphql.NewNonNullType(DateTimeType),
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			if ctx.IsSubscribe {
				ticker := time.NewTicker(time.Second)
				return &SubscriptionSourceStream{
					EventChannel: ticker.C,
					Stop:         ticker.Stop,
				}, nil
			} else if ctx.Object != nil {
				return ctx.Object, nil
			} else {
				return nil, fmt.Errorf("subscriptions are not supported using this protocol")
			}
		},
	})

	api, err := NewAPI(&testCfg)
	require.NoError(t, err)
	defer api.CloseHijackedConnections()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.ServeGraphQLWS(w, r)
	}))
	defer ts.Close()

	dialer := &websocket.Dialer{
		HandshakeTimeout: time.Second,
		Subprotocols:     []string{graphqlws.WebSocketSubprotocol},
	}

	var conn *websocket.Conn
	for attempts := 0; attempts < 100; attempts++ {
		clientConn, _, err := dialer.Dial("ws"+strings.TrimPrefix(ts.URL, "http"), nil)
		if err != nil {
			time.Sleep(time.Millisecond * 10)
		} else {
			conn = clientConn
			break
		}
	}
	require.NotNil(t, conn)
	defer func() {
		assert.NoError(t, conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "closing")))
		conn.Close()
	}()

	require.NoError(t, conn.WriteJSON(map[string]string{
		"id":   "init",
		"type": "connection_init",
	}))

	var msg graphqlws.Message

	require.NoError(t, conn.ReadJSON(&msg))
	assert.Equal(t, graphqlws.MessageTypeConnectionAck, msg.Type)

	require.NoError(t, conn.ReadJSON(&msg))
	assert.Equal(t, graphqlws.MessageTypeConnectionKeepAlive, msg.Type)

	t.Run("Query", func(t *testing.T) {
		require.NoError(t, conn.WriteJSON(map[string]interface{}{
			"id":   "query",
			"type": "start",
			"payload": map[string]interface{}{
				"query": `
					{
						foo
					}
				`,
			},
		}))

		require.NoError(t, conn.ReadJSON(&msg))
		assert.Equal(t, "query", msg.Id)
		assert.Equal(t, graphqlws.MessageTypeData, msg.Type)

		require.NoError(t, conn.ReadJSON(&msg))
		assert.Equal(t, "query", msg.Id)
		assert.Equal(t, graphqlws.MessageTypeComplete, msg.Type)
	})

	t.Run("Subscription", func(t *testing.T) {
		require.NoError(t, conn.WriteJSON(map[string]interface{}{
			"id":   "sub",
			"type": "start",
			"payload": map[string]interface{}{
				"query": `
					subscription {
						time
					}
				`,
			},
		}))

		require.NoError(t, conn.ReadJSON(&msg))
		assert.Equal(t, "sub", msg.Id)
		assert.Equal(t, graphqlws.MessageTypeData, msg.Type)

		require.NoError(t, conn.WriteJSON(map[string]interface{}{
			"id":   "sub",
			"type": "stop",
		}))

		require.NoError(t, conn.ReadJSON(&msg))
		assert.Equal(t, "sub", msg.Id)
		assert.Equal(t, graphqlws.MessageTypeComplete, msg.Type)
	})
}

func TestGraphQLWS_InitParameters(t *testing.T) {
	var testCfg Config

	testCfg.AddQueryField("whoami", &graphql.FieldDefinition{
		Type: graphql.StringType,
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			return ctx.Context.Value("name"), nil
		},
	})

	testCfg.HandleGraphQLWSInit = func(ctx context.Context, parameters json.RawMessage) (context.Context, error) {
		var params struct {
			Name string
		}
		if err := json.Unmarshal(parameters, &params); err != nil {
			return ctx, err
		} else if params.Name == "" {
			return ctx, fmt.Errorf("no name")
		}
		ctx = context.WithValue(ctx, "name", params.Name)
		return ctx, nil
	}

	api, err := NewAPI(&testCfg)
	require.NoError(t, err)
	defer api.CloseHijackedConnections()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.ServeGraphQLWS(w, r)
	}))
	defer ts.Close()

	dialer := &websocket.Dialer{
		HandshakeTimeout: time.Second,
		Subprotocols:     []string{graphqlws.WebSocketSubprotocol},
	}

	for name, tc := range map[string]struct {
		Parameters    json.RawMessage
		ExpectedName  string
		ExpectedError string
	}{
		"Ok": {
			ExpectedName: "alice",
			Parameters:   json.RawMessage(`{"name": "alice"}`),
		},
		"NoName": {
			ExpectedError: "no name",
			Parameters:    json.RawMessage(`{"foo": "bar"}`),
		},
	} {
		t.Run(name, func(t *testing.T) {
			var conn *websocket.Conn
			for attempts := 0; attempts < 100; attempts++ {
				clientConn, _, err := dialer.Dial("ws"+strings.TrimPrefix(ts.URL, "http"), nil)
				if err != nil {
					time.Sleep(time.Millisecond * 10)
				} else {
					conn = clientConn
					break
				}
			}
			require.NotNil(t, conn)
			defer func() {
				assert.NoError(t, conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "closing")))
				conn.Close()
			}()

			require.NoError(t, conn.WriteJSON(map[string]interface{}{
				"id":      "init",
				"type":    "connection_init",
				"payload": tc.Parameters,
			}))

			var msg graphqlws.Message

			if tc.ExpectedError != "" {
				require.NoError(t, conn.ReadJSON(&msg))
				assert.Equal(t, graphqlws.MessageTypeConnectionError, msg.Type)
				assert.JSONEq(t, fmt.Sprintf(`{"message": %#v}`, tc.ExpectedError), string(msg.Payload))
			} else {
				require.NoError(t, conn.ReadJSON(&msg))
				assert.Equal(t, graphqlws.MessageTypeConnectionAck, msg.Type)

				require.NoError(t, conn.ReadJSON(&msg))
				assert.Equal(t, graphqlws.MessageTypeConnectionKeepAlive, msg.Type)

				require.NoError(t, conn.WriteJSON(map[string]interface{}{
					"id":   "query",
					"type": "start",
					"payload": map[string]interface{}{
						"query": `
					{
						whoami
					}
				`,
					},
				}))

				require.NoError(t, conn.ReadJSON(&msg))
				assert.Equal(t, "query", msg.Id)
				assert.Equal(t, graphqlws.MessageTypeData, msg.Type)
				assert.JSONEq(t, fmt.Sprintf(`{"data": {"whoami": %#v}}`, tc.ExpectedName), string(msg.Payload))

				require.NoError(t, conn.ReadJSON(&msg))
				assert.Equal(t, "query", msg.Id)
				assert.Equal(t, graphqlws.MessageTypeComplete, msg.Type)
			}
		})
	}
}

func TestGraphQLWSTransport(t *testing.T) {
	var testCfg Config

	testCfg.AddQueryField("foo", &graphql.FieldDefinition{
		Type: graphql.BooleanType,
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			return true, nil
		},
	})

	testCfg.AddSubscription("time", &graphql.FieldDefinition{
		Type: graphql.NewNonNullType(DateTimeType),
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			if ctx.IsSubscribe {
				ticker := time.NewTicker(time.Second)
				return &SubscriptionSourceStream{
					EventChannel: ticker.C,
					Stop:         ticker.Stop,
				}, nil
			} else if ctx.Object != nil {
				return ctx.Object, nil
			} else {
				return nil, fmt.Errorf("subscriptions are not supported using this protocol")
			}
		},
	})

	api, err := NewAPI(&testCfg)
	require.NoError(t, err)
	defer api.CloseHijackedConnections()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.ServeGraphQLWS(w, r)
	}))
	defer ts.Close()

	dialer := &websocket.Dialer{
		HandshakeTimeout: time.Second,
		Subprotocols:     []string{graphqltransportws.WebSocketSubprotocol},
	}

	var conn *websocket.Conn
	for attempts := 0; attempts < 100; attempts++ {
		clientConn, _, err := dialer.Dial("ws"+strings.TrimPrefix(ts.URL, "http"), nil)
		if err != nil {
			time.Sleep(time.Millisecond * 10)
		} else {
			conn = clientConn
			break
		}
	}
	require.NotNil(t, conn)
	defer func() {
		assert.NoError(t, conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "closing")))
		conn.Close()
	}()

	require.NoError(t, conn.WriteJSON(map[string]string{
		"id":   "init",
		"type": "connection_init",
	}))

	var msg graphqltransportws.Message

	require.NoError(t, conn.ReadJSON(&msg))
	assert.Equal(t, graphqltransportws.MessageTypeConnectionAck, msg.Type)

	t.Run("Query", func(t *testing.T) {
		require.NoError(t, conn.WriteJSON(map[string]interface{}{
			"id":   "query",
			"type": "subscribe",
			"payload": map[string]interface{}{
				"query": `
					{
						foo
					}
				`,
			},
		}))

		require.NoError(t, conn.ReadJSON(&msg))
		assert.Equal(t, "query", msg.Id)
		assert.Equal(t, graphqltransportws.MessageTypeNext, msg.Type)

		require.NoError(t, conn.ReadJSON(&msg))
		assert.Equal(t, "query", msg.Id)
		assert.Equal(t, graphqltransportws.MessageTypeComplete, msg.Type)
	})

	t.Run("Subscription", func(t *testing.T) {
		require.NoError(t, conn.WriteJSON(map[string]interface{}{
			"id":   "sub",
			"type": "subscribe",
			"payload": map[string]interface{}{
				"query": `
					subscription {
						time
					}
				`,
			},
		}))

		require.NoError(t, conn.ReadJSON(&msg))
		assert.Equal(t, "sub", msg.Id)
		assert.Equal(t, graphqltransportws.MessageTypeNext, msg.Type)

		require.NoError(t, conn.WriteJSON(map[string]interface{}{
			"id":   "sub",
			"type": "complete",
		}))

		require.NoError(t, conn.ReadJSON(&msg))
		assert.Equal(t, "sub", msg.Id)
		assert.Equal(t, graphqltransportws.MessageTypeComplete, msg.Type)
	})
}

func TestGraphQLTransportWS_InitParameters(t *testing.T) {
	var testCfg Config

	testCfg.AddQueryField("whoami", &graphql.FieldDefinition{
		Type: graphql.StringType,
		Resolve: func(ctx graphql.FieldContext) (interface{}, error) {
			return ctx.Context.Value("name"), nil
		},
	})

	testCfg.HandleGraphQLWSInit = func(ctx context.Context, parameters json.RawMessage) (context.Context, error) {
		var params struct {
			Name string
		}
		if err := json.Unmarshal(parameters, &params); err != nil {
			return ctx, err
		} else if params.Name == "" {
			return ctx, fmt.Errorf("no name")
		}
		ctx = context.WithValue(ctx, "name", params.Name)
		return ctx, nil
	}

	api, err := NewAPI(&testCfg)
	require.NoError(t, err)
	defer api.CloseHijackedConnections()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.ServeGraphQLWS(w, r)
	}))
	defer ts.Close()

	dialer := &websocket.Dialer{
		HandshakeTimeout: time.Second,
		Subprotocols:     []string{graphqltransportws.WebSocketSubprotocol},
	}

	for name, tc := range map[string]struct {
		Parameters    json.RawMessage
		ExpectedName  string
		ExpectedError string
	}{
		"Ok": {
			ExpectedName: "alice",
			Parameters:   json.RawMessage(`{"name": "alice"}`),
		},
		"NoName": {
			ExpectedError: "websocket: close 4403: no name",
			Parameters:    json.RawMessage(`{"foo": "bar"}`),
		},
	} {
		t.Run(name, func(t *testing.T) {
			var conn *websocket.Conn
			for attempts := 0; attempts < 100; attempts++ {
				clientConn, _, err := dialer.Dial("ws"+strings.TrimPrefix(ts.URL, "http"), nil)
				if err != nil {
					time.Sleep(time.Millisecond * 10)
				} else {
					conn = clientConn
					break
				}
			}
			require.NotNil(t, conn)
			defer func() {
				conn.Close()
			}()

			require.NoError(t, conn.WriteJSON(map[string]interface{}{
				"type":    "connection_init",
				"payload": tc.Parameters,
			}))

			var msg graphqltransportws.Message

			if tc.ExpectedError != "" {
				err := conn.ReadJSON(&msg)
				require.Error(t, err)
				assert.Equal(t, err.Error(), tc.ExpectedError)
			} else {
				require.NoError(t, conn.ReadJSON(&msg))
				assert.Equal(t, graphqltransportws.MessageTypeConnectionAck, msg.Type)

				require.NoError(t, conn.WriteJSON(map[string]interface{}{
					"id":   "query",
					"type": "subscribe",
					"payload": map[string]interface{}{
						"query": `
					{
						whoami
					}
				`,
					},
				}))

				require.NoError(t, conn.ReadJSON(&msg))
				assert.Equal(t, "query", msg.Id)
				assert.Equal(t, graphqltransportws.MessageTypeNext, msg.Type)
				assert.JSONEq(t, fmt.Sprintf(`{"data": {"whoami": %#v}}`, tc.ExpectedName), string(msg.Payload))

				require.NoError(t, conn.ReadJSON(&msg))
				assert.Equal(t, "query", msg.Id)
				assert.Equal(t, graphqltransportws.MessageTypeComplete, msg.Type)
			}
		})
	}
}

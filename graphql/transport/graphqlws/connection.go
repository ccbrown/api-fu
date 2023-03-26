package graphqlws

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/ccbrown/api-fu/graphql"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

// Connection represents a server-side GraphQL-WS connection.
type Connection struct {
	Handler ConnectionHandler

	conn              *websocket.Conn
	readLoopDone      chan struct{}
	writeLoopDone     chan struct{}
	outgoing          chan *websocket.PreparedMessage
	close             chan struct{}
	closeReceived     chan struct{}
	closeMessage      chan []byte
	beginClosingOnce  sync.Once
	finishClosingOnce sync.Once
	didInit           bool
}

// ConnectionHandler methods may be invoked on a separate goroutine, but invocations will never be
// made concurrently.
type ConnectionHandler interface {
	// Called when the server receives the init message. If an error is returned, it will be sent to
	// the client and the connection will be closed.
	HandleInit(parameters json.RawMessage) error

	// Called when the client wants to start an operation. If the operation is a query or mutation,
	// the handler should immediately call SendData followed by SendComplete. If the operation is a
	// subscription, the handler should call SendData to send events and SendComplete if/when the
	// event stream ends.
	HandleStart(id string, query string, variables map[string]interface{}, operationName string)

	// Called when the client wants to stop an operation. The handler should unsubscribe them from
	// the corresponding subscription.
	HandleStop(id string)

	// Called when an unexpected error occurs. The connection will perform the appropriate response,
	// but you may want to log it.
	LogError(err error)

	// Called when the connection begins closing and all in-flight operations should be canceled.
	Cancel()

	// Called when the connection is closed.
	HandleClose()
}

const connectionSendBufferSize = 100

// Serve takes ownership of the given connection and begins reading / writing to it.
func (c *Connection) Serve(conn *websocket.Conn) {
	c.conn = conn
	c.readLoopDone = make(chan struct{})
	c.writeLoopDone = make(chan struct{})
	c.outgoing = make(chan *websocket.PreparedMessage, connectionSendBufferSize)
	c.close = make(chan struct{})
	c.closeReceived = make(chan struct{})
	c.closeMessage = make(chan []byte, 1)
	conn.SetCloseHandler(func(code int, text string) error {
		select {
		case <-c.closeReceived:
		default:
			close(c.closeReceived)
		}
		return nil
	})
	go c.readLoop()
	go c.writeLoop()
}

// SendData sends the given GraphQL response to the client.
func (c *Connection) SendData(ctx context.Context, id string, response *graphql.Response) error {
	buf, err := jsoniter.Marshal(response)
	if err != nil {
		return errors.Wrap(err, "unable to marshal graphql response")
	}
	return c.sendMessage(ctx, &Message{
		Id:      id,
		Type:    MessageTypeData,
		Payload: json.RawMessage(buf),
	})
}

// SendComplete sends the "complete" message to the client. This should be done after queries are
// executed or subscriptions are stopped.
func (c *Connection) SendComplete(ctx context.Context, id string) error {
	return c.sendMessage(ctx, &Message{
		Id:   id,
		Type: MessageTypeComplete,
	})
}

// Close closes the connection. This must not be called from handler functions.
func (c *Connection) Close() error {
	c.beginClosing(websocket.CloseNormalClosure, "close requested by application")
	c.finishClosing()
	return nil
}

func (c *Connection) sendMessage(ctx context.Context, msg *Message) error {
	data, err := jsoniter.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "error marshaling message")
	}
	prepared, err := websocket.NewPreparedMessage(websocket.TextMessage, data)
	if err != nil {
		return errors.Wrap(err, "error preparing message")
	}
	select {
	case c.outgoing <- prepared:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func (c *Connection) readLoop() {
	defer close(c.readLoopDone)
	defer c.beginClosing(websocket.CloseInternalServerErr, "read error")

	for {
		_, p, err := c.conn.ReadMessage()
		if err != nil {
			if _, ok := err.(*websocket.CloseError); !ok {
				select {
				case <-c.close:
				default:
					c.Handler.LogError(errors.Wrap(err, "websocket read error"))
				}
			}
			return
		}

		c.handleMessage(context.Background(), p)
	}
}

func (c *Connection) handleMessage(ctx context.Context, data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		// ignore malformed messages
		return
	}

	switch msg.Type {
	case MessageTypeConnectionInit:
		if err := c.Handler.HandleInit(msg.Payload); err != nil {
			payload := struct {
				Message string `json:"message"`
			}{
				Message: err.Error(),
			}
			if buf, err := jsoniter.Marshal(payload); err != nil {
				c.Handler.LogError(errors.Wrap(err, "unable to marshal graphql-ws connection error payload"))
			} else if err := c.sendMessage(ctx, &Message{
				Id:      msg.Id,
				Type:    MessageTypeConnectionError,
				Payload: buf,
			}); err != nil {
				c.Handler.LogError(errors.Wrap(err, "unable to send graphql-ws connection error"))
			}
			c.beginClosing(websocket.CloseInternalServerErr, "connection init error")
			return
		}

		c.didInit = true
		if err := c.sendMessage(ctx, &Message{
			Id:   msg.Id,
			Type: MessageTypeConnectionAck,
		}); err != nil {
			c.Handler.LogError(errors.Wrap(err, "unable to send graphql-ws connection ack"))
			c.beginClosing(websocket.CloseInternalServerErr, "ack send error")
		} else if err := c.sendMessage(ctx, &Message{
			Type: MessageTypeConnectionKeepAlive,
		}); err != nil {
			c.Handler.LogError(errors.Wrap(err, "unable to send graphql-ws initial keep-alive"))
			c.beginClosing(websocket.CloseInternalServerErr, "keep-alive send error")
		}
	case MessageTypeStart:
		if !c.didInit {
			return
		}

		var payload struct {
			Query         string                 `json:"query"`
			Variables     map[string]interface{} `json:"variables"`
			OperationName string                 `json:"operationName"`
		}
		if err := jsoniter.Unmarshal(msg.Payload, &payload); err != nil {
			// ignore malformed messages
			return
		}
		c.Handler.HandleStart(msg.Id, payload.Query, payload.Variables, payload.OperationName)
	case MessageTypeStop:
		if !c.didInit {
			return
		}

		c.Handler.HandleStop(msg.Id)
		if err := c.sendMessage(context.Background(), &Message{
			Id:   msg.Id,
			Type: MessageTypeComplete,
		}); err != nil {
			c.Handler.LogError(errors.Wrap(err, "unable to send graphql-ws stop response"))
		}
	case MessageTypeConnectionTerminate:
		c.beginClosing(websocket.CloseNormalClosure, "terminate requested by client")
	default:
		// ignore unknown message types
	}
}

var keepAlivePreparedMessage *websocket.PreparedMessage

func init() {
	data, err := jsoniter.Marshal(&Message{
		Type: MessageTypeConnectionKeepAlive,
	})
	if err != nil {
		panic(errors.Wrap(err, "error marshaling message"))
	}
	prepared, err := websocket.NewPreparedMessage(websocket.TextMessage, data)
	if err != nil {
		panic(errors.Wrap(err, "error preparing message"))
	}
	keepAlivePreparedMessage = prepared
}

func (c *Connection) writeLoop() {
	defer c.finishClosing()
	defer close(c.writeLoopDone)

	defer c.conn.Close()

	keepAliveTicker := time.NewTicker(15 * time.Second)
	defer keepAliveTicker.Stop()

	for {
		var msg *websocket.PreparedMessage
		select {
		case outgoing := <-c.outgoing:
			msg = outgoing
		case <-keepAliveTicker.C:
			msg = keepAlivePreparedMessage
		case msg := <-c.closeMessage:
			// make sure we send any outgoing messages before closing (e.g. to make sure we send
			// back the error after a bad init)
			for done := false; !done; {
				select {
				case msg := <-c.outgoing:
					c.conn.SetWriteDeadline(time.Now().Add(time.Second))
					if err := c.conn.WritePreparedMessage(msg); err != nil {
						if !websocket.IsCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseGoingAway) && err != websocket.ErrCloseSent {
							c.Handler.LogError(errors.Wrap(err, "websocket write error"))
						}
						done = true
					}
				default:
					done = true
				}
			}

			// initiate the close handshake
			if err := c.conn.WriteMessage(websocket.CloseMessage, msg); err != nil {
				c.Handler.LogError(errors.Wrap(err, "websocket control write error"))
			}
			// wait for the response, then close the connection
			select {
			case <-c.closeReceived:
			case <-c.readLoopDone:
			case <-time.After(time.Second):
			}
			return
		case <-c.closeReceived:
			// the client initiated the close handshake
			if err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "close requested by client")); err != nil {
				c.Handler.LogError(errors.Wrap(err, "websocket control write error"))
			}
			return
		}

		c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

		if err := c.conn.WritePreparedMessage(msg); err != nil {
			if !websocket.IsCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseGoingAway) && err != websocket.ErrCloseSent {
				c.Handler.LogError(errors.Wrap(err, "websocket write error"))
			}
			return
		}
	}
}

func (c *Connection) beginClosing(code int, text string) {
	c.beginClosingOnce.Do(func() {
		c.closeMessage <- websocket.FormatCloseMessage(code, text)
		close(c.close)
		c.Handler.Cancel()
	})
}

func (c *Connection) finishClosing() {
	<-c.readLoopDone
	<-c.writeLoopDone
	invokeHandler := false
	c.finishClosingOnce.Do(func() {
		invokeHandler = true
	})
	if invokeHandler {
		c.Handler.HandleClose()
	}
}

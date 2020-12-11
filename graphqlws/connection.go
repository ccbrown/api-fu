package graphqlws

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ccbrown/api-fu/graphql"
	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Connection represents a server-side GraphQL-WS connection.
type Connection struct {
	Logger  logrus.FieldLogger
	Handler ConnectionHandler

	conn              *websocket.Conn
	readLoopDone      chan struct{}
	writeLoopDone     chan struct{}
	outgoing          chan *websocket.PreparedMessage
	close             chan struct{}
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
	go c.readLoop()
	go c.writeLoop()
}

// SendData sends the given GraphQL response to the client.
func (c *Connection) SendData(id string, response *graphql.Response) error {
	buf, err := jsoniter.Marshal(response)
	if err != nil {
		return errors.Wrap(err, "unable to marshal graphql response")
	}
	return c.sendMessage(&Message{
		Id:      id,
		Type:    MessageTypeData,
		Payload: json.RawMessage(buf),
	})
}

// SendComplete sends the "complete" message to the client. This should be done after queries are
// executed or subscriptions are stopped.
func (c *Connection) SendComplete(id string) error {
	return c.sendMessage(&Message{
		Id:   id,
		Type: MessageTypeComplete,
	})
}

// Close closes the connection. This must not be called from handler functions.
func (c *Connection) Close() error {
	c.beginClosing()
	c.finishClosing()
	return nil
}

func (c *Connection) sendMessage(msg *Message) error {
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
	default:
		return fmt.Errorf("send buffer full")
	}
	return nil
}

func (c *Connection) readLoop() {
	defer close(c.readLoopDone)
	defer c.beginClosing()

	for {
		_, p, err := c.conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseGoingAway) {
				select {
				case <-c.close:
				default:
					c.Logger.Error(errors.Wrap(err, "websocket read error"))
				}
			}
			return
		}

		c.handleMessage(p)
	}
}

func (c *Connection) handleMessage(data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		c.Logger.WithField("error", err.Error()).Info("malformed graphql-ws message received")
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
				c.Logger.Error(errors.Wrap(err, "unable to marshal graphql-ws connection error payload"))
			} else if err := c.sendMessage(&Message{
				Id:      msg.Id,
				Type:    MessageTypeConnectionError,
				Payload: buf,
			}); err != nil {
				c.Logger.Error(errors.Wrap(err, "unable to send graphql-ws connection error"))
			}
			c.beginClosing()
			return
		}

		c.didInit = true
		if err := c.sendMessage(&Message{
			Id:   msg.Id,
			Type: MessageTypeConnectionAck,
		}); err != nil {
			c.Logger.Error(errors.Wrap(err, "unable to send graphql-ws connection ack"))
			c.beginClosing()
		} else if err := c.sendMessage(&Message{
			Type: MessageTypeConnectionKeepAlive,
		}); err != nil {
			c.Logger.Error(errors.Wrap(err, "unable to send graphql-ws initial keep-alive"))
			c.beginClosing()
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
			c.Logger.WithField("error", err.Error()).Info("malformed graphql-ws message received")
			return
		}
		c.Handler.HandleStart(msg.Id, payload.Query, payload.Variables, payload.OperationName)
	case MessageTypeStop:
		if !c.didInit {
			return
		}

		c.Handler.HandleStop(msg.Id)
		if err := c.sendMessage(&Message{
			Id:   msg.Id,
			Type: MessageTypeComplete,
		}); err != nil {
			c.Logger.Error(errors.Wrap(err, "unable to send graphql-ws stop response"))
		}
	case MessageTypeConnectionTerminate:
		c.beginClosing()
	default:
		c.Logger.Info("unknown graphql-ws message type received")
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
		case outgoing, ok := <-c.outgoing:
			if !ok {
				return
			}
			msg = outgoing
		case <-keepAliveTicker.C:
			msg = keepAlivePreparedMessage
		case <-c.close:
			return
		}

		c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

		if err := c.conn.WritePreparedMessage(msg); err != nil {
			if !websocket.IsCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseGoingAway) && err != websocket.ErrCloseSent {
				c.Logger.Error(errors.Wrap(err, "websocket write error"))
			}
			return
		}
	}
}

func (c *Connection) beginClosing() {
	c.beginClosingOnce.Do(func() {
		close(c.close)
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

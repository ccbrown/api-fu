package graphqltransportws

import (
	"encoding/json"
)

const WebSocketSubprotocol = "graphql-transport-ws"

// MessageType represents a GraphQL-WS message type.
type MessageType string

// MessageType represents a GraphQL-WS message type.
const (
	MessageTypeConnectionInit MessageType = "connection_init"
	MessageTypeConnectionAck  MessageType = "connection_ack"
	MessageTypePing           MessageType = "ping"
	MessageTypePong           MessageType = "pong"
	MessageTypeSubscribe      MessageType = "subscribe"
	MessageTypeNext           MessageType = "next"
	MessageTypeError          MessageType = "error"
	MessageTypeComplete       MessageType = "complete"
)

// Message represents a GraphQL-WS message. This can be used for both client and server messages.
type Message struct {
	Id      string          `json:"id,omitempty"`
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

package graphqlws

import (
	"encoding/json"
)

// MessageType represents a GraphQL-WS message type.
type MessageType string

// MessageType represents a GraphQL-WS message type.
const (
	MessageTypeConnectionInit      MessageType = "connection_init"
	MessageTypeConnectionKeepAlive MessageType = "ka"
	MessageTypeConnectionTerminate MessageType = "connection_terminate"
	MessageTypeConnectionAck       MessageType = "connection_ack"
	MessageTypeComplete            MessageType = "complete"
	MessageTypeData                MessageType = "data"
	MessageTypeStart               MessageType = "start"
	MessageTypeStop                MessageType = "stop"
	MessageTypeError               MessageType = "error"
)

// Message represents a GraphQL-WS message. This can be used for both client and server messages.
type Message struct {
	Id      string          `json:"id,omitempty"`
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

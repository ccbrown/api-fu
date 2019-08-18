package graphqlws

import (
	"encoding/json"
)

type MessageType string

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

type Message struct {
	Id      string          `json:"id,omitempty"`
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

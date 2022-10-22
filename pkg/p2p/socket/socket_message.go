package socket

import (
	"context"

	"github.com/gorilla/websocket"
)

const (
	// EventMessageReceived is the key for socket event.
	EventMessageReceived = "EventMessageReceived"
	// EventGeneralMessageReceived is the key for socket event.
	EventGeneralMessageReceived = "EventGeneralMessageReceived"
	// EventSocketClosed is the key for socket close event.
	EventSocketClosed = "EventSocketClosed"
	// EventMessageLimitExceeded is the key for socket event.
	EventMessageLimitExceeded = "EventMessageLimitExceeded"
	// EventSocketError is the key for socket error event.
	EventSocketError = "EventSocketError"
)

// Request holds request for context.
type Request struct {
	ctx       context.Context
	PeerID    string
	Procedure string
	Data      []byte
}

// Event holds event message from a peer.
type Event struct {
	PeerID string
	Event  string
	Data   []byte
}

// GeneralMessage holds generic message.
type GeneralMessage struct {
	Event string
	Data  []byte
}

// RPCHandler is a definition for accepting the RPC request from a peer.
type RPCHandler func(w ResponseWriter, req *Request)

// HandshakeHandler is function to check before connection.
type HandshakeHandler func(*websocket.Conn) error

// Response holds result.
type Response interface {
	Err() error
	Data() []byte
}

// ResponseWriter is a interface for handler to write.
type ResponseWriter interface {
	Write([]byte)
	Error(error)
}

type responseWriter struct {
	data []byte
	err  error
}

func (w *responseWriter) Write(data []byte) {
	w.data = data
}
func (w *responseWriter) Error(err error) {
	w.err = err
}

// NewResponse creates response.
func NewResponse(data []byte, err error) Response {
	return &response{
		data: data,
		err:  err,
	}
}

type response struct {
	err  error
	data []byte
}

func (r *response) Err() error {
	return r.err
}

func (r *response) Data() []byte {
	return r.data
}

// Message represents the event happens on the socket.
type Message struct {
	kind  string
	event string
	data  []byte
	err   error
}

// Kind returns the type of message.
func (m *Message) Kind() string {
	return m.kind
}

// Event returns the type of event in case it is socket event.
func (m *Message) Event() string {
	return m.event
}

// Data returns the data of event in case it is socket event.
func (m *Message) Data() []byte {
	return m.data
}

// Err returns err if exist.
func (m *Message) Err() error {
	return m.err
}

func newErrorMessage(err error) *Message {
	return &Message{
		kind: EventSocketError,
		err:  err,
	}
}

func newEventMessage(event string, data []byte) *Message {
	return &Message{
		kind:  EventMessageReceived,
		event: event,
		data:  data,
	}
}

func newClosedMessage() *Message {
	return &Message{
		kind: EventSocketClosed,
	}
}

func newLimitExceededMessage() *Message {
	return &Message{
		kind: EventMessageLimitExceeded,
	}
}

func newGeneralMessage(event string, data []byte) *Message {
	return &Message{
		kind:  EventGeneralMessageReceived,
		event: event,
		data:  data,
	}
}

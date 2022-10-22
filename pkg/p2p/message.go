package p2p

import (
	"github.com/LiskHQ/lisk-engine/pkg/p2p/socket"
)

// Event holds event message from a peer
//
//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen
type Event struct {
	peerID string
	event  string
	data   []byte
}

func newEvent(peerID string, event string, data []byte) *Event {
	return &Event{
		peerID: peerID,
		event:  event,
		data:   data,
	}
}

// PeerID returns sender peer id.
func (e *Event) PeerID() string {
	return e.peerID
}

// Event returns event type.
func (e *Event) Event() string {
	return e.event
}

// Data returns event payload.
func (e *Event) Data() []byte {
	return e.data
}

// Request holds request from a peer.
type Request socket.Request

type Response socket.Response

// RPCHandler is a definition for accepting the RPC request from a peer.
type RPCHandler func(w ResponseWriter, r *Request)

type EventHandler func(event *Event)

// ResponseWriter is a interface for handler to write.
type ResponseWriter socket.ResponseWriter

type rpcGetPeersResponse struct {
	Peers [][]byte `fieldNumber:"1"`
}

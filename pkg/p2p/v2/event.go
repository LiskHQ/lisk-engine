package p2p

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

// Event holds event message from a peer.
type Event struct {
	peerID string
	event  string
	data   []byte
}

// newEvent creates a new event.
func newEvent(peerID string, event string, data []byte) *Event {
	return &Event{
		peerID: peerID,
		event:  event,
		data:   data,
	}
}

// PeerID returns sender peer ID.
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

// EventHandler is a handler function for event received from a peer.
type EventHandler func(event *Event)

package p2p

// Event holds event message from a peer.
type Event struct {
	peerID PeerID
	topic  string
	data   []byte
}

// NewEvent creates a new event.
func NewEvent(peerID PeerID, topic string, data []byte) *Event {
	return &Event{
		peerID: peerID,
		topic:  topic,
		data:   data,
	}
}

// PeerID returns sender peer ID.
func (e *Event) PeerID() PeerID {
	return e.peerID
}

// Topic returns topic of the event.
func (e *Event) Topic() string {
	return e.topic
}

// Data returns event payload.
func (e *Event) Data() []byte {
	return e.data
}

// EventHandler is a handler function for event received from a peer.
type EventHandler func(event *Event)

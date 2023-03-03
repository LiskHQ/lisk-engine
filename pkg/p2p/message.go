package p2p

import (
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

// Request is a request message type sent to other peer.
type Request struct {
	ID        string `fieldNumber:"1" json:"id"`        // Message ID.
	Procedure string `fieldNumber:"2" json:"procedure"` // Procedure to be called.
	Data      []byte `fieldNumber:"3" json:"data"`      // Request data.
	Timestamp int64  `json:"timestamp"`                 // Unix time when the message was received.
	PeerID    string `json:"peerID"`                    // ID of peer that created the request message.
}

// responseMsg is a response message type received from a peer in response to a request message.
type responseMsg struct {
	ID        string `fieldNumber:"1" json:"id"`    // Message ID. It is the same as the ID of the requested message.
	Timestamp int64  `json:"timestamp"`             // Unix time when the message was received.
	PeerID    string `json:"peerID"`                // ID of peer that created the response message.
	Data      []byte `fieldNumber:"2" json:"data"`  // Response data.
	Error     string `fieldNumber:"3" json:"error"` // Error message in case of an error.
}

// Message is a message type sent to other peers in the network over GossipSub.
type Message struct {
	Timestamp int64  `json:"timestamp"`            // Unix time when the message was received.
	Data      []byte `fieldNumber:"1" json:"data"` // Message data (payload).
}

// newRequestMessage creates a new request message.
func newRequestMessage(peerID peer.ID, procedure string, data []byte) *Request {
	return &Request{
		ID:        uuid.New().String(),
		Timestamp: time.Now().Unix(),
		PeerID:    peerID.String(),
		Procedure: procedure,
		Data:      data,
	}
}

// newResponseMessage creates a new response message.
func newResponseMessage(reqMsgID string, data []byte, err error) *responseMsg {
	errString := ""
	if err != nil {
		errString = err.Error()
	}

	return &responseMsg{
		ID:        reqMsgID,
		Timestamp: time.Now().Unix(),
		Data:      data,
		Error:     errString,
	}
}

// NewMessage creates a new message.
func NewMessage(data []byte) *Message {
	return &Message{
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
}

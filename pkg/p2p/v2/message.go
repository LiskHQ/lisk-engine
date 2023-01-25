package p2p

import (
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

// MessageRequestType is a type of a request message.
type MessageRequestType string

// MessageRequestType enum.
const (
	MessageRequestTypePing       MessageRequestType = "ping"       // Ping request message type. Used to check if a peer is alive.
	MessageRequestTypeKnownPeers MessageRequestType = "knownPeers" // Get all known peers request message type.
)

// RequestMsg is a request message type sent to other peer.
type RequestMsg struct {
	ID        string `fieldNumber:"1" json:"id"`        // Message ID.
	Timestamp int64  `fieldNumber:"2" json:"timestamp"` // Unix time when the message was received.
	PeerID    string `fieldNumber:"3" json:"peerID"`    // ID of peer that created the request message.
	Procedure string `fieldNumber:"4" json:"procedure"` // Procedure to be called.
	Data      []byte `fieldNumber:"5" json:"data"`      // Request data.
}

// ResponseMsg is a response message type received from a peer in response to a request message.
type ResponseMsg struct {
	ID        string `fieldNumber:"1" json:"id"`        // Message ID. It is the same as the ID of the requested message.
	Timestamp int64  `fieldNumber:"2" json:"timestamp"` // Unix time when the message was received.
	PeerID    string `fieldNumber:"3" json:"peerID"`    // ID of peer that created the response message.
	Data      []byte `fieldNumber:"4" json:"data"`      // Response data.
}

// Message is a message type sent to other peers in the network over GossipSub.
type Message struct {
	Timestamp int64  `fieldNumber:"1" json:"timestamp"` // Unix time when the message was received.
	Data      []byte `fieldNumber:"2" json:"data"`      // Message data (payload).
}

// newRequestMessage creates a new request message.
func newRequestMessage(peerID peer.ID, procedure MessageRequestType, data []byte) *RequestMsg {
	return &RequestMsg{
		ID:        uuid.New().String(),
		Timestamp: time.Now().Unix(),
		PeerID:    peerID.String(),
		Procedure: string(procedure),
		Data:      data,
	}
}

// newResponseMessage creates a new response message.
func newResponseMessage(peerID peer.ID, reqMsgID string, data []byte) *ResponseMsg {
	return &ResponseMsg{
		ID:        reqMsgID,
		Timestamp: time.Now().Unix(),
		PeerID:    peerID.String(),
		Data:      data,
	}
}

// newMessage creates a new message.
func newMessage(data []byte) *Message {
	return &Message{
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
}

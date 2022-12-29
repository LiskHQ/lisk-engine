package p2p

import (
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"

	pb "github.com/LiskHQ/lisk-engine/pkg/p2p/v2/pb"
)

// MessageRequestType is a type of a request message.
type MessageRequestType uint32

// MessageRequestType enum.
const (
	MessageRequestTypePing       MessageRequestType = iota // Ping request message type. Used to check if a peer is alive.
	MessageRequestTypeKnownPeers                           // Get all known peers request message type.
)

// ResponseMsg is a response message type sent to a channel.
type ResponseMsg struct {
	ID        string // Message ID.
	Timestamp int64  // Unix time.
	PeerID    string // ID of peer that created the response message.
	ReqMsgID  string // ID of the requested message.
	Data      []byte // Response data.
	Err       error  // Error message in case of an error.
}

// newRequestMessage creates a new request message.
func newRequestMessage(peerID peer.ID, procedure MessageRequestType, data []byte) *pb.Request {
	return &pb.Request{
		MsgData:   newMessageData(peerID),
		Procedure: (uint32)(procedure),
		Data:      data,
	}
}

// newResponseMessage creates a new response message.
func newResponseMessage(peerID peer.ID, reqMsgID string, data []byte) *pb.Response {
	return &pb.Response{
		MsgData:  newMessageData(peerID),
		ReqMsgID: reqMsgID,
		Data:     data,
	}
}

// newMessageData creates a new message data.
func newMessageData(peerID peer.ID) *pb.MessageData {
	return &pb.MessageData{
		Id:        uuid.New().String(),
		Timestamp: time.Now().Unix(),
		PeerID:    peerID.String(),
	}
}

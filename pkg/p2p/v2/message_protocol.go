package p2p

import (
	"context"
	"io"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const messageProtocolID = "/lisk/message/0.0.1"

// MessageProtocol type.
type MessageProtocol struct {
	peer *Peer
}

// NewMessageProtocol creates a new message protocol with a stream handler.
func NewMessageProtocol(peer *Peer) *MessageProtocol {
	mp := &MessageProtocol{peer: peer}
	peer.host.SetStreamHandler(messageProtocolID, mp.onMessageReceive)
	mp.peer.logger.Infof("Message protocol is set")
	return mp
}

// onMessageReceive is a handler for a received message.
func (mp *MessageProtocol) onMessageReceive(s network.Stream) {
	buf, err := io.ReadAll(s)
	if err != nil {
		_ = s.Reset()
		mp.peer.logger.Errorf("Error onMessageReceive: %v", err)
		return
	}
	s.Close()
	mp.peer.logger.Infof("Data from %v received: %s", s.Conn().RemotePeer().String(), string(buf))
}

// SendMessage sends a message to a peer using a message protocol.
func (mp *MessageProtocol) SendMessage(ctx context.Context, id peer.ID, msg string) error {
	return mp.peer.sendProtoMessage(ctx, id, messageProtocolID, msg)
}

package main

import (
	"context"
	"encoding/json"

	"github.com/libp2p/go-libp2p/core/peer"

	p2p "github.com/LiskHQ/lisk-engine/pkg/p2p/v2"
)

// ChatRoomBufSize is the number of incoming messages to buffer for each topic.
const ChatRoomBufSize = 128

// ChatRoom represents a subscription to a single PubSub topic. Messages
// can be published to the topic with ChatRoom.Publish, and received
// messages are pushed to the Messages channel.
type ChatRoom struct {
	// Messages is a channel of messages received from other peers in the chat room
	Messages chan *ChatMessage

	ctx  context.Context
	gs   *p2p.GossipSub
	host *p2p.Peer

	roomName string
	self     peer.ID
	nick     string
}

// ChatMessage gets converted to/from JSON and sent in the body of pubsub messages.
type ChatMessage struct {
	Message    string
	SenderID   string
	SenderNick string
}

// JoinChatRoom tries to subscribe to the PubSub topic for the room name, returning
// a ChatRoom on success.
func JoinChatRoom(ctx context.Context, gs *p2p.GossipSub, h *p2p.Peer, ch chan *ChatMessage, nickname string, roomName string) (*ChatRoom, error) {
	cr := &ChatRoom{
		ctx:      ctx,
		gs:       gs,
		host:     h,
		self:     h.GetHost().ID(),
		nick:     nickname,
		roomName: roomName,
		Messages: ch,
	}

	return cr, nil
}

// Publish sends a message to the pubsub topic.
func (cr *ChatRoom) Publish(message string) error {
	m := ChatMessage{
		Message:    message,
		SenderID:   cr.self.Pretty(),
		SenderNick: cr.nick,
	}
	msgBytes, err := json.Marshal(m)
	if err != nil {
		return err
	}
	msg := new(p2p.Message)
	msg.Data = msgBytes
	return cr.gs.Publish(context.Background(), topicName(cr.roomName), msg)
}

// ListPeers returns an array of ID.
func (cr *ChatRoom) ListPeers() []peer.ID {
	return cr.host.ConnectedPeers()
}

// Read messages from the subscription and push them onto the Messages channel.
func readMessage(event *p2p.Event, ch chan *ChatMessage) {
	cm := new(ChatMessage)
	err := json.Unmarshal(event.Data(), cm)
	if err != nil {
		return
	}
	// send valid messages onto the Messages channel
	ch <- cm
}

package p2p

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

const messageProtocolReqID = "/lisk/message/req/0.0.1"
const messageProtocolResID = "/lisk/message/res/0.0.1"

const messageResponseTimeout = 3 * time.Second // Time to wait for a response message before returning an error

// MessageProtocol type.
type MessageProtocol struct {
	ctx     context.Context
	peer    *Peer
	resMu   sync.Mutex
	resCh   map[string]chan<- *ResponseMsg
	timeout time.Duration
}

// NewMessageProtocol creates a new message protocol with a stream handler.
func NewMessageProtocol(ctx context.Context, peer *Peer) *MessageProtocol {
	mp := &MessageProtocol{ctx: ctx, peer: peer, resCh: make(map[string]chan<- *ResponseMsg), timeout: messageResponseTimeout}
	peer.host.SetStreamHandler(messageProtocolReqID, mp.onMessageReqReceive)
	peer.host.SetStreamHandler(messageProtocolResID, mp.onMessageResReceive)
	mp.peer.logger.Infof("Message protocol is set")
	return mp
}

// onMessageReqReceive is a handler for a received request message.
func (mp *MessageProtocol) onMessageReqReceive(s network.Stream) {
	buf, err := io.ReadAll(s)
	if err != nil {
		_ = s.Reset()
		mp.peer.logger.Errorf("Error onMessageReqReceive: %v", err)
		return
	}
	s.Close()
	mp.peer.logger.Debugf("Data from %v received: %s", s.Conn().RemotePeer().String(), string(buf))

	newMsg := &RequestMsg{}
	if err := newMsg.Decode(buf); err != nil {
		mp.peer.logger.Errorf("Error unmarshalling message: %v", err)
		return
	}
	mp.peer.logger.Debugf("Request message received: %v", newMsg)

	// TODO: Implement a proper procedure (requests) handling (registering, unregistering, handler functions, etc.) (GH issue #13)
	switch (MessageRequestType)(newMsg.Procedure) {
	case MessageRequestTypePing:
		mp.peer.logger.Debugf("Ping request received")

		rtt, err := mp.peer.PingMultiTimes(mp.ctx, s.Conn().RemotePeer())
		if err != nil {
			mp.peer.logger.Errorf("Ping error: %v", err)
		}
		var sum time.Duration
		for _, i := range rtt {
			sum += i
		}
		avg := time.Duration(float64(sum) / float64(len(rtt)))

		err = mp.SendResponseMessage(mp.ctx, s.Conn().RemotePeer(), newMsg.ID, []byte(fmt.Sprintf("Average RTT with you: %v", avg)))
		if err != nil {
			mp.peer.logger.Errorf("Error sending response message: %v", err)
		}
	case MessageRequestTypeKnownPeers:
		mp.peer.logger.Debugf("Get known peers request received")
		peers := mp.peer.KnownPeers()
		err := mp.SendResponseMessage(mp.ctx, s.Conn().RemotePeer(), newMsg.ID, []byte(fmt.Sprintf("All known peers: %v", peers)))
		if err != nil {
			mp.peer.logger.Errorf("Error sending response message: %v", err)
		}
	}
}

// onMessageResReceive is a handler for a received response message.
func (mp *MessageProtocol) onMessageResReceive(s network.Stream) {
	buf, err := io.ReadAll(s)
	if err != nil {
		_ = s.Reset()
		mp.peer.logger.Errorf("Error onMessageResReceive: %v", err)
		return
	}
	s.Close()
	mp.peer.logger.Debugf("Data from %v received: %s", s.Conn().RemotePeer().String(), string(buf))

	newMsg := &ResponseMsg{}
	if err := newMsg.Decode(buf); err != nil {
		mp.peer.logger.Errorf("Error unmarshalling message: %v", err)
		return
	}
	mp.peer.logger.Debugf("Response message received: %v", newMsg)

	mp.resMu.Lock()
	defer mp.resMu.Unlock()
	if ch, ok := mp.resCh[newMsg.ID]; ok {
		if ch != nil {
			ch <- &ResponseMsg{
				ID:        newMsg.ID,
				Timestamp: newMsg.Timestamp,
				PeerID:    newMsg.PeerID,
				Data:      newMsg.Data,
			}
		}
		delete(mp.resCh, newMsg.ID)
	} else {
		mp.peer.logger.Warningf("Response message received for unknown request ID: %v", newMsg.ID)
	}
}

// SendRequestMessage sends a request message to a peer using a message protocol.
func (mp *MessageProtocol) SendRequestMessage(ctx context.Context, id peer.ID, procedure MessageRequestType, data []byte, ch chan<- *ResponseMsg) error {
	reqMsg := newRequestMessage(mp.peer.ID(), procedure, data)
	if err := mp.sendProtoMessage(ctx, id, messageProtocolReqID, reqMsg); err != nil {
		return err
	}

	mp.resMu.Lock()
	defer mp.resMu.Unlock()
	mp.resCh[reqMsg.ID] = ch

	// Start a goroutine to wait for a timeout.
	go func() {
		select {
		case <-ctx.Done():
			return

		case <-time.After(mp.timeout):
			mp.resMu.Lock()
			defer mp.resMu.Unlock()

			resCh, ok := mp.resCh[reqMsg.ID]
			if !ok {
				// Response already received
				return
			}
			// Timeout occurs. Send a timeout error response and delete the channel from the map.
			resCh <- &ResponseMsg{Err: errors.New("timeout")}
			delete(mp.resCh, reqMsg.ID)
			return
		}
	}()

	return nil
}

// SendResponseMessage sends a response message to a peer using a message protocol.
func (mp *MessageProtocol) SendResponseMessage(ctx context.Context, id peer.ID, reqMsgID string, data []byte) error {
	resMsg := newResponseMessage(mp.peer.ID(), reqMsgID, data)
	return mp.sendProtoMessage(ctx, id, messageProtocolResID, resMsg)
}

// sendProtoMessage sends a message to a peer using a stream.
func (mp *MessageProtocol) sendProtoMessage(ctx context.Context, id peer.ID, pId protocol.ID, msg codec.Encodable) error {
	s, err := mp.peer.host.NewStream(network.WithUseTransient(ctx, "Transient connections are allowed."), id, pId)
	if err != nil {
		return err
	}
	defer s.Close()

	data, err := msg.Encode()
	if err != nil {
		return err
	}

	n, err := s.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return errors.New("error while sending a message, did not sent a whole message")
	}

	return nil
}

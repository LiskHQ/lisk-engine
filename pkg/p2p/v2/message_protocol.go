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
	"github.com/LiskHQ/lisk-engine/pkg/log"
)

const messageProtocolReqID = "/lisk/message/req/0.0.1"
const messageProtocolResID = "/lisk/message/res/0.0.1"

const messageResponseTimeout = 3 * time.Second // Time to wait for a response message before returning an error

// MessageProtocol type.
type MessageProtocol struct {
	logger      log.Logger
	peer        *Peer
	resMu       sync.Mutex
	resCh       map[string]chan<- *Response
	timeout     time.Duration
	rpcHandlers map[string]RPCHandler
}

// NewMessageProtocol creates a new message protocol.
func NewMessageProtocol() *MessageProtocol {
	mp := &MessageProtocol{resCh: make(map[string]chan<- *Response), timeout: messageResponseTimeout, rpcHandlers: make(map[string]RPCHandler)}
	return mp
}

// Start starts a message protocol with a stream handlers.
func (mp *MessageProtocol) Start(ctx context.Context, logger log.Logger, peer *Peer) {
	mp.logger = logger
	mp.peer = peer
	peer.host.SetStreamHandler(messageProtocolReqID, func(s network.Stream) {
		mp.onRequest(ctx, s)
	})
	peer.host.SetStreamHandler(messageProtocolResID, func(s network.Stream) {
		mp.onResponse(ctx, s)
	})
	mp.logger.Infof("Message protocol is started")
}

// onRequest is a handler for a received request message.
func (mp *MessageProtocol) onRequest(ctx context.Context, s network.Stream) {
	buf, err := io.ReadAll(s)
	if err != nil {
		_ = s.Reset()
		mp.logger.Errorf("Error onRequest: %v", err)
		return
	}
	s.Close()
	mp.logger.Debugf("Data from %v received: %s", s.Conn().RemotePeer().String(), string(buf))

	blockPeer := func() {
		remoteID := s.Conn().RemotePeer()
		mp.peer.BlockPeer(remoteID)
		closeErr := mp.peer.Disconnect(ctx, remoteID)
		if closeErr != nil {
			mp.logger.Errorf("Blocked peer is not disconnected with error:", err)
		}
	}

	newMsg := newRequestMessage(s.Conn().RemotePeer(), "", nil)
	if err := newMsg.Decode(buf); err != nil {
		blockPeer()
		mp.logger.Errorf("Error while decoding message: %v", err)
		return
	}
	mp.logger.Debugf("Request message received: %+v", newMsg)

	handler, exist := mp.rpcHandlers[newMsg.Procedure]
	if !exist {
		blockPeer()
		mp.logger.Errorf("rpcHandler %s is not registered", newMsg.Procedure)
		return
	}
	mp.logger.Debugf("%s request received", newMsg.Procedure)
	w := &responseWriter{}
	handler(w, newMsg)
	err = mp.SendResponseMessage(ctx, s.Conn().RemotePeer(), newMsg.ID, w.data, w.err)
	if err != nil {
		mp.logger.Errorf("Error sending response message: %v", err)
		return
	}
}

// onResponse is a handler for a received response message.
func (mp *MessageProtocol) onResponse(ctx context.Context, s network.Stream) {
	buf, err := io.ReadAll(s)
	if err != nil {
		_ = s.Reset()
		mp.logger.Errorf("Error onResponse: %v", err)
		return
	}
	s.Close()
	mp.logger.Debugf("Data from %v received: %s", s.Conn().RemotePeer().String(), string(buf))

	newMsg := newResponseMessage("", nil, nil)
	if err := newMsg.Decode(buf); err != nil {
		remoteID := s.Conn().RemotePeer()
		mp.peer.BlockPeer(remoteID)
		closeErr := mp.peer.Disconnect(ctx, remoteID)
		if closeErr != nil {
			mp.logger.Errorf("Blocked peer is not disconnected with error:", err)
		}
		mp.logger.Errorf("Error while decoding message: %v", err)
		return
	}
	mp.logger.Debugf("Response message received: %+v", newMsg)

	mp.resMu.Lock()
	defer mp.resMu.Unlock()
	if ch, ok := mp.resCh[newMsg.ID]; ok {
		var resError error
		if newMsg.Error != "" {
			resError = errors.New(newMsg.Error)
		}
		ch <- newResponse(
			newMsg.Timestamp,
			s.Conn().RemotePeer().String(),
			newMsg.Data,
			resError,
		)
	} else {
		mp.logger.Warningf("Response message received for unknown request ID: %v", newMsg.ID)
	}
}

// RegisterRPCHandler registers a new RPC handler function.
func (mp *MessageProtocol) RegisterRPCHandler(name string, handler RPCHandler) error {
	if mp.peer != nil {
		return errors.New("cannot register RPC handler after MessageProtocol is started")
	}
	if _, ok := mp.rpcHandlers[name]; ok {
		return fmt.Errorf("rpcHandler %s is already registered", name)
	}
	mp.rpcHandlers[name] = handler
	return nil
}

// SendRequestMessage sends a request message to a peer using a message protocol.
func (mp *MessageProtocol) SendRequestMessage(ctx context.Context, id peer.ID, procedure string, data []byte) (*Response, error) {
	reqMsg := newRequestMessage(mp.peer.ID(), procedure, data)
	if err := mp.sendMessage(ctx, id, messageProtocolReqID, reqMsg); err != nil {
		return nil, err
	}

	ch := make(chan *Response)
	mp.resMu.Lock()
	mp.resCh[reqMsg.ID] = ch
	mp.resMu.Unlock()

	// Wait for a response message or timeout
	select {
	case resMsg := <-ch:
		mp.resMu.Lock()
		delete(mp.resCh, reqMsg.ID)
		mp.resMu.Unlock()
		return resMsg, nil

	case <-time.After(mp.timeout):
		// Timeout occurs.
		mp.resMu.Lock()
		delete(mp.resCh, reqMsg.ID)
		mp.resMu.Unlock()
		return nil, errors.New("timeout")

	case <-ctx.Done():
		mp.resMu.Lock()
		delete(mp.resCh, reqMsg.ID)
		mp.resMu.Unlock()
		return nil, ctx.Err()
	}
}

// SendResponseMessage sends a response message to a peer using a message protocol.
func (mp *MessageProtocol) SendResponseMessage(ctx context.Context, id peer.ID, reqMsgID string, data []byte, err error) error {
	resMsg := newResponseMessage(reqMsgID, data, err)
	return mp.sendMessage(ctx, id, messageProtocolResID, resMsg)
}

// sendMessage sends a message to a peer using a stream.
func (mp *MessageProtocol) sendMessage(ctx context.Context, id peer.ID, pId protocol.ID, msg codec.Encodable) error {
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

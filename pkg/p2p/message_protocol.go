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

const messageResponseTimeout = 3 * time.Second // Time to wait for a response message before returning an error
const messageMaxRetries = 3                    // Maximum number of retries for a request message

var errTimeout = errors.New("timeout")

func messageProtocolReqID(chainID []byte, version string) protocol.ID {
	return protocol.ID(fmt.Sprintf("/lisk/message/req/%s/%s", codec.Hex(chainID).String(), version))
}

func messageProtocolResID(chainID []byte, version string) protocol.ID {
	return protocol.ID(fmt.Sprintf("/lisk/message/res/%s/%s", codec.Hex(chainID).String(), version))
}

type RPCHandlerOption func(*MessageProtocol, string) error

// MessageProtocol type.
type MessageProtocol struct {
	logger      log.Logger
	peer        *Peer
	resMu       sync.Mutex
	resCh       map[string]chan<- *Response
	timeout     time.Duration
	rpcHandlers map[string]RPCHandler
	rateLimit   *rateLimit
	chainID     []byte
	version     string
}

// newMessageProtocol creates a new message protocol.
func newMessageProtocol(chainID []byte, version string) *MessageProtocol {
	mp := &MessageProtocol{
		resCh:       make(map[string]chan<- *Response),
		timeout:     messageResponseTimeout,
		rpcHandlers: make(map[string]RPCHandler),
		rateLimit:   newRateLimit(),
		chainID:     chainID,
		version:     version,
	}
	return mp
}

// RegisterRPCHandler registers a new RPC handler function.
func (mp *MessageProtocol) RegisterRPCHandler(name string, handler RPCHandler, opts ...RPCHandlerOption) error {
	if mp.peer != nil {
		return errors.New("cannot register RPC handler after MessageProtocol is started")
	}
	if _, ok := mp.rpcHandlers[name]; ok {
		return fmt.Errorf("rpcHandler %s is already registered", name)
	}
	mp.rpcHandlers[name] = handler
	if err := mp.rateLimit.addRPCMessageCounter(name); err != nil {
		return err
	}

	for _, opt := range opts {
		err := opt(mp, name)
		if err != nil {
			return err
		}
	}

	return nil
}

// RequestFrom sends a request message to a peer using a message protocol.
func (mp *MessageProtocol) RequestFrom(ctx context.Context, peerID PeerID, procedure string, data []byte) Response {
	response, err := mp.request(ctx, peerID, procedure, data)
	if err != nil {
		return Response{err: err}
	}
	return *response
}

// Broadcast sends a request message to all connected peers using a message protocol.
func (mp *MessageProtocol) Broadcast(ctx context.Context, procedure string, data []byte) error {
	peers := mp.peer.ConnectedPeers()
	for _, peerID := range peers {
		if _, err := mp.request(ctx, peerID, procedure, data); err != nil {
			return err
		}
	}
	return nil
}

// start starts a message protocol with a stream handlers.
func (mp *MessageProtocol) start(ctx context.Context, logger log.Logger, peer *Peer) {
	mp.logger = logger
	mp.peer = peer
	mp.rateLimit.start(logger, peer)

	peer.host.SetStreamHandler(messageProtocolReqID(mp.chainID, mp.version), func(s network.Stream) {
		mp.onRequest(ctx, s)
	})
	peer.host.SetStreamHandler(messageProtocolResID(mp.chainID, mp.version), mp.onResponse)
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

	remoteAddr := s.Conn().RemoteMultiaddr()
	newMsg := newRequestMessage(s.Conn().RemotePeer(), "", nil)
	if err := newMsg.Decode(buf); err != nil {
		mp.logger.Errorf("Error while decoding message: %v", err)
		err = mp.peer.banPeer(remoteAddr)
		if err != nil {
			mp.logger.Errorf("banPeer error: %v", err)
		}
		return
	}
	mp.logger.Debugf("Request message received: %+v", newMsg)

	handler, exist := mp.rpcHandlers[newMsg.Procedure]
	if !exist {
		mp.logger.Errorf("rpcHandler %s is not registered", newMsg.Procedure)
		err = mp.peer.banPeer(remoteAddr)
		if err != nil {
			mp.logger.Errorf("banPeer error: %v", err)
		}
		return
	}
	mp.logger.Debugf("%s request received", newMsg.Procedure)

	// Rate limiting
	remoteID := s.Conn().RemotePeer()
	mp.rateLimit.increaseCounter(newMsg.Procedure, remoteID)
	err = mp.rateLimit.checkLimit(newMsg.Procedure, remoteID, remoteAddr)
	if err != nil {
		mp.logger.Errorf("Rate limit error: %v", err)
		return
	}

	w := &responseWriter{}
	handler(w, newMsg)
	err = mp.respond(ctx, s.Conn().RemotePeer(), newMsg.ID, newMsg.Procedure, w.data, w.err)
	if err != nil {
		mp.logger.Errorf("Error sending response message: %v", err)
		return
	}
}

// onResponse is a handler for a received response message.
func (mp *MessageProtocol) onResponse(s network.Stream) {
	buf, err := io.ReadAll(s)
	if err != nil {
		_ = s.Reset()
		mp.logger.Errorf("Error onResponse: %v", err)
		return
	}
	s.Close()
	mp.logger.Debugf("Data from %v received: %s", s.Conn().RemotePeer().String(), string(buf))

	remoteAddr := s.Conn().RemoteMultiaddr()
	newMsg := newResponseMessage("", "", nil, nil)
	if err := newMsg.Decode(buf); err != nil {
		mp.logger.Errorf("Error while decoding message: %v", err)
		err = mp.peer.banPeer(remoteAddr)
		if err != nil {
			mp.logger.Errorf("banPeer error: %v", err)
		}
		return
	}
	mp.logger.Debugf("Response message received: %+v", newMsg)

	_, exist := mp.rpcHandlers[newMsg.Procedure]
	if !exist {
		mp.logger.Errorf("rpcHandler %s for received message is not registered", newMsg.Procedure)
		err = mp.peer.banPeer(remoteAddr)
		if err != nil {
			mp.logger.Errorf("banPeer error: %v", err)
		}
		return
	}

	// Rate limiting
	remoteID := s.Conn().RemotePeer()
	mp.rateLimit.increaseCounter(newMsg.Procedure, remoteID)
	err = mp.rateLimit.checkLimit(newMsg.Procedure, remoteID, remoteAddr)
	if err != nil {
		mp.logger.Errorf("Rate limit error: %v", err)
		return
	}

	mp.resMu.Lock()
	defer mp.resMu.Unlock()
	if ch, ok := mp.resCh[newMsg.ID]; ok {
		var resError error
		if newMsg.Error != "" {
			resError = errors.New(newMsg.Error)
		}
		ch <- NewResponse(
			newMsg.Timestamp,
			s.Conn().RemotePeer(),
			newMsg.Data,
			resError,
		)
	} else {
		mp.logger.Warningf("Response message received for unknown request ID: %v", newMsg.ID)
	}
}

// request to a peer using a message protocol and has a retry mechanism.
func (mp *MessageProtocol) request(ctx context.Context, id peer.ID, procedure string, data []byte) (*Response, error) {
	var (
		err error
		res *Response
	)

	for i := 0; i <= messageMaxRetries; i++ {
		if i > 0 {
			mp.logger.Debugf("Retrying request message to %v. Retry count: %d", id, i)
		}
		res, err = mp.sendRequestMessage(ctx, id, procedure, data)
		if err != nil {
			if errors.Is(err, errTimeout) {
				continue
			}
			return nil, err
		}
		return res, nil
	}
	return nil, err
}

// sendRequestMessage to a peer using a message protocol.
func (mp *MessageProtocol) sendRequestMessage(ctx context.Context, id peer.ID, procedure string, data []byte) (*Response, error) {
	reqMsg := newRequestMessage(mp.peer.ID(), procedure, data)
	if err := mp.send(ctx, id, messageProtocolReqID(mp.chainID, mp.version), reqMsg); err != nil {
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
		return nil, errTimeout

	case <-ctx.Done():
		mp.resMu.Lock()
		delete(mp.resCh, reqMsg.ID)
		mp.resMu.Unlock()
		return nil, ctx.Err()
	}
}

// respond a message to a peer using a message protocol.
func (mp *MessageProtocol) respond(ctx context.Context, id peer.ID, reqMsgID string, procedure string, data []byte, err error) error {
	resMsg := newResponseMessage(reqMsgID, procedure, data, err)
	return mp.send(ctx, id, messageProtocolResID(mp.chainID, mp.version), resMsg)
}

// send a message to a peer using a stream.
func (mp *MessageProtocol) send(ctx context.Context, id peer.ID, pId protocol.ID, msg codec.Encodable) error {
	s, err := mp.peer.host.NewStream(network.WithUseTransient(ctx, "Transient connections are allowed."), id, pId)
	if err != nil {
		return err
	}
	defer s.Close()

	data := msg.Encode()

	n, err := s.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return errors.New("error while sending a message, did not sent a whole message")
	}

	return nil
}

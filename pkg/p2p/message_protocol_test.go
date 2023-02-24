package p2p

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"

	cfg "github.com/LiskHQ/lisk-engine/pkg/engine/config"
	"github.com/LiskHQ/lisk-engine/pkg/log"
)

type testConn struct {
	network.Conn
}

func (c testConn) RemotePeer() peer.ID {
	return peer.ID("testPeerID")
}

type testStream struct {
	network.Stream
	data []byte
}

func (s testStream) Read(p []byte) (n int, err error) {
	copy(p, s.data)
	return len(s.data), io.EOF
}

func (s testStream) Close() error {
	return nil
}

func (s testStream) Conn() network.Conn {
	return testConn{}
}

func TestMessageProtocol_NewMessageProtocol(t *testing.T) {
	assert := assert.New(t)

	mp := NewMessageProtocol()
	assert.Nil(mp.logger)
	assert.Nil(mp.peer)
	assert.Equal(0, len(mp.resCh))
	assert.Equal(3*time.Second, mp.timeout)
	assert.Equal(0, len(mp.rpcHandlers))
}

func TestMessageProtocol_Start(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfgNet := cfg.NetworkConfig{}
	_ = cfgNet.InsertDefault()
	wg := &sync.WaitGroup{}
	p, _ := NewPeer(ctx, wg, logger, cfgNet)

	mp := NewMessageProtocol()
	mp.Start(ctx, logger, p)
	assert.Equal(logger, mp.logger)
	assert.Equal(p, mp.peer)
}

func TestMessageProtocol_OnRequest(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var tests = []struct {
		name      string
		procedure string
		want      string
	}{
		{"ping request message", "ping", "Request received"},
		{"known peers request message", "knownPeers", "Request received"},
	}

	wg := &sync.WaitGroup{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := log.NewDefaultProductionLogger()
			loggerTest := testLogger{Logger: logger}
			cfgNet := cfg.NetworkConfig{}
			_ = cfgNet.InsertDefault()
			p, _ := NewPeer(ctx, wg, &loggerTest, cfgNet)
			mp := NewMessageProtocol()
			mp.RegisterRPCHandler(tt.procedure, func(w ResponseWriter, req *RequestMsg) {
				mp.logger.Debugf("Request received")
				w.Write([]byte(testResponseData))
			})
			mp.Start(ctx, &loggerTest, p)

			stream := testStream{}
			reqMsg := newRequestMessage(testPeerID, tt.procedure, []byte(""))
			data, _ := reqMsg.Encode()
			stream.data = data
			mp.onRequest(ctx, stream)

			idx := slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, "Request message received") })
			assert.NotEqual(-1, idx)

			idx = slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, tt.want) })
			assert.NotEqual(-1, idx)
		})
	}
}

func TestMessageProtocol_OnResponse(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	loggerTest := testLogger{Logger: logger}
	cfgNet := cfg.NetworkConfig{}
	_ = cfgNet.InsertDefault()
	wg := &sync.WaitGroup{}
	p, _ := NewPeer(ctx, wg, &loggerTest, cfgNet)
	mp := NewMessageProtocol()
	mp.Start(ctx, &loggerTest, p)
	ch := make(chan *Response, 1)
	mp.resCh[testReqMsgID] = ch

	stream := testStream{}
	reqMsg := newResponseMessage(testReqMsgID, []byte(testResponseData), errors.New(testError))
	data, _ := reqMsg.Encode()
	stream.data = data
	mp.onResponse(stream)

	select {
	case response := <-ch:
		assert.Equal(testResponseData, string(response.Data()))
		assert.Equal(testError, response.Error().Error())
		break
	case <-time.After(testTimeout):
		t.Fatalf("timeout occurs, response message was not created and sent to the channel")
	}

	assert.Equal(1, len(mp.resCh))
	idx := slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, "Response message received") })
	assert.NotEqual(-1, idx)
}

func TestMessageProtocol_OnResponseUnknownRequestID(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	loggerTest := testLogger{Logger: logger}
	cfgNet := cfg.NetworkConfig{}
	_ = cfgNet.InsertDefault()
	wg := &sync.WaitGroup{}
	p, _ := NewPeer(ctx, wg, &loggerTest, cfgNet)
	mp := NewMessageProtocol()
	mp.Start(ctx, &loggerTest, p)
	// There is no channel for the request ID "testReqMsgID"

	stream := testStream{}
	reqMsg := newResponseMessage(testReqMsgID, []byte(testResponseData), nil)
	data, _ := reqMsg.Encode()
	stream.data = data
	mp.onResponse(stream)

	idx := slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, "Response message received for unknown request ID") })
	assert.NotEqual(-1, idx)
}

func TestMessageProtocol_RegisterRPCHandler(t *testing.T) {
	assert := assert.New(t)

	testHandler := func(w ResponseWriter, req *RequestMsg) {
	}

	mp := NewMessageProtocol()
	err := mp.RegisterRPCHandler(testRPC, testHandler)
	assert.Nil(err)

	assert.NotNil(mp.rpcHandlers[testRPC])

	f1 := *(*unsafe.Pointer)(unsafe.Pointer(&testHandler))
	handler := mp.rpcHandlers[testRPC]
	f2 := *(*unsafe.Pointer)(unsafe.Pointer(&handler))
	assert.True(f1 == f2)
}

func TestMessageProtocol_RegisterRPCHandlerMessageProtocolRunning(t *testing.T) {
	assert := assert.New(t)

	testHandler := func(w ResponseWriter, req *RequestMsg) {
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfgNet := cfg.NetworkConfig{}
	_ = cfgNet.InsertDefault()
	wg := &sync.WaitGroup{}
	p, _ := NewPeer(ctx, wg, logger, cfgNet)

	mp := NewMessageProtocol()
	mp.Start(ctx, logger, p)

	err := mp.RegisterRPCHandler(testRPC, testHandler)
	assert.NotNil(err)
	assert.Equal("cannot register RPC handler after MessageProtocol is started", err.Error())

	_, exist := mp.rpcHandlers[testRPC]
	assert.False(exist)
}

func TestMessageProtocol_RegisterRPCHandlerAlreadyRegistered(t *testing.T) {
	assert := assert.New(t)

	testHandler := func(w ResponseWriter, req *RequestMsg) {
	}

	mp := NewMessageProtocol()

	err := mp.RegisterRPCHandler(testRPC, testHandler)
	assert.Nil(err)
	_, exist := mp.rpcHandlers[testRPC]
	assert.True(exist)

	err = mp.RegisterRPCHandler(testRPC, testHandler)
	assert.NotNil(err)
	assert.Equal("rpcHandler testRPC is already registered", err.Error())
}

func TestMessageProtocol_SendRequestMessage(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfgNet := cfg.NetworkConfig{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfgNet.InsertDefault()

	wg := &sync.WaitGroup{}
	p1, _ := NewPeer(ctx, wg, logger, cfgNet)
	mp1 := NewMessageProtocol()
	mp1.Start(ctx, logger, p1)
	p2, _ := NewPeer(ctx, wg, logger, cfgNet)
	mp2 := NewMessageProtocol()
	mp2.RegisterRPCHandler(testRPC, func(w ResponseWriter, req *RequestMsg) {
		w.Write([]byte("Average RTT with you:"))
	})
	mp2.Start(ctx, logger, p2)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	err := p1.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)

	response, err := mp1.SendRequestMessage(ctx, p2.ID(), testRPC, []byte(testRequestData))
	assert.Nil(err)
	assert.Equal(p2.ID().String(), response.PeerID())
	assert.Contains(string(response.Data()), "Average RTT with you:")
}

func TestMessageProtocol_SendRequestMessageRPCHandlerError(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfgNet := cfg.NetworkConfig{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfgNet.InsertDefault()

	wg := &sync.WaitGroup{}
	p1, _ := NewPeer(ctx, wg, logger, cfgNet)
	mp1 := NewMessageProtocol()
	mp1.Start(ctx, logger, p1)
	p2, _ := NewPeer(ctx, wg, logger, cfgNet)
	mp2 := NewMessageProtocol()
	mp2.RegisterRPCHandler(testRPC, func(w ResponseWriter, req *RequestMsg) {
		w.Error(errors.New("Test RPC handler error!"))
	})
	mp2.Start(ctx, logger, p2)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	err := p1.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)

	response, err := mp1.SendRequestMessage(ctx, p2.ID(), testRPC, []byte(testRequestData))
	assert.Nil(err)
	assert.Equal(p2.ID().String(), response.PeerID())
	assert.Equal([]byte{}, response.Data())
	assert.Contains(response.Error().Error(), "Test RPC handler error!")
}

func TestMessageProtocol_SendRequestMessageTimeout(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfgNet := cfg.NetworkConfig{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfgNet.InsertDefault()

	wg := &sync.WaitGroup{}
	p1, _ := NewPeer(ctx, wg, logger, cfgNet)
	mp1 := NewMessageProtocol()
	mp1.Start(ctx, logger, p1)
	mp1.timeout = time.Millisecond * 20 // Reduce timeout to 20 ms to speed up test
	// Remove response message stream handler to simulate timeout
	p1.host.RemoveStreamHandler(messageProtocolResID)
	p2, _ := NewPeer(ctx, wg, logger, cfgNet)
	mp2 := NewMessageProtocol()
	mp2.Start(ctx, logger, p2)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	err := p1.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)

	response, err := mp1.SendRequestMessage(ctx, p2.ID(), testRPC, []byte(testRequestData))
	assert.Nil(response)
	assert.Equal("timeout", err.Error())
}

func TestMessageProtocol_SendResponseMessage(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfgNet := cfg.NetworkConfig{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfgNet.InsertDefault()

	wg := &sync.WaitGroup{}
	p1, _ := NewPeer(ctx, wg, logger, cfgNet)
	mp1 := NewMessageProtocol()
	mp1.Start(ctx, logger, p1)
	p2, _ := NewPeer(ctx, wg, logger, cfgNet)
	mp2 := NewMessageProtocol()
	mp2.Start(ctx, logger, p2)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	err := p1.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)
	ch := make(chan *Response, 1)
	mp2.resCh[testReqMsgID] = ch
	err = mp1.SendResponseMessage(ctx, p2.ID(), testReqMsgID, []byte(testResponseData), errors.New(testError))
	assert.Nil(err)

	select {
	case response := <-ch:
		assert.Equal(p1.ID().String(), response.PeerID())
		assert.Equal(testResponseData, string(response.Data()))
		assert.Equal(testError, response.Error().Error())
		break
	case <-time.After(testTimeout):
		t.Fatalf("timeout occurs, response message was not received")
	}
}

type TestMessageReceive struct {
	msg  string
	done chan any
}

func (tmr *TestMessageReceive) onMessageReceive(s network.Stream) {
	buf, _ := io.ReadAll(s)
	s.Close()
	tmr.msg = string(buf)
	tmr.done <- "done"
}

func TestMessageProtocol_SendProtoMessage(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfgNet := cfg.NetworkConfig{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfgNet.InsertDefault()
	tmr := TestMessageReceive{done: make(chan any)}

	wg := &sync.WaitGroup{}
	p1, _ := NewPeer(ctx, wg, logger, cfgNet)
	p2, _ := NewPeer(ctx, wg, logger, cfgNet)
	p2.host.SetStreamHandler(messageProtocolReqID, tmr.onMessageReceive)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	_ = p1.Connect(ctx, *p2AddrInfo)
	msg := newRequestMessage(p1.ID(), testProcedure, []byte(testRequestData))
	mp := NewMessageProtocol()
	mp.Start(ctx, logger, p1)
	err := mp.sendMessage(ctx, p2.ID(), messageProtocolReqID, msg)
	assert.Nil(err)

	select {
	case <-tmr.done:
		break
	case <-time.After(time.Second * pingTimeout):
		t.Fatalf("timeout occurs, message was not delivered to a peer")
	}

	assert.Contains(tmr.msg, testRequestData)
}

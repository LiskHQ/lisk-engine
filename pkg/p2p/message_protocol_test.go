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

	mp := newMessageProtocol(testChainID, testVersion)
	assert.Nil(mp.logger)
	assert.Nil(mp.peer)
	assert.Equal(0, len(mp.resCh))
	assert.Equal(3*time.Second, mp.timeout)
	assert.Equal(0, len(mp.rpcHandlers))
	assert.Equal(testChainID, mp.chainID)
	assert.Equal(testVersion, mp.version)
}

func TestMessageProtocol_Start(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{}
	_ = cfg.insertDefault()
	wg := &sync.WaitGroup{}
	p, _ := newPeer(ctx, wg, logger, []byte{}, cfg)

	mp := newMessageProtocol(testChainID, testVersion)
	mp.start(ctx, logger, p)
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
			cfg := &Config{}
			_ = cfg.insertDefault()
			p, _ := newPeer(ctx, wg, &loggerTest, []byte{}, cfg)
			mp := newMessageProtocol(testChainID, testVersion)
			mp.RegisterRPCHandler(tt.procedure, func(w ResponseWriter, req *Request) {
				mp.logger.Debugf("Request received")
				w.Write([]byte(testResponseData))
			}, RateLimit{Limit: 10, Penalty: 10})
			mp.start(ctx, &loggerTest, p)

			stream := testStream{}
			reqMsg := newRequestMessage(testPeerID, tt.procedure, []byte(""))
			data, _ := reqMsg.Encode()
			stream.data = data
			mp.onRequest(ctx, stream)

			idx := slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, "Error sending response message") })
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
	cfg := &Config{}
	_ = cfg.insertDefault()
	wg := &sync.WaitGroup{}
	p, _ := newPeer(ctx, wg, &loggerTest, []byte{}, cfg)
	mp := newMessageProtocol(testChainID, testVersion)
	testHandler := func(w ResponseWriter, req *Request) {}
	err := mp.RegisterRPCHandler(testRPC, testHandler, RateLimit{Limit: 10, Penalty: 10})
	assert.Nil(err)
	mp.start(ctx, &loggerTest, p)
	ch := make(chan *Response, 1)
	mp.resCh[testReqMsgID] = ch

	stream := testStream{}
	reqMsg := newResponseMessage(testReqMsgID, testRPC, []byte(testResponseData), errors.New(testError))
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
	cfg := &Config{}
	_ = cfg.insertDefault()
	wg := &sync.WaitGroup{}
	p, _ := newPeer(ctx, wg, &loggerTest, []byte{}, cfg)
	mp := newMessageProtocol(testChainID, testVersion)
	testHandler := func(w ResponseWriter, req *Request) {}
	mp.RegisterRPCHandler(testProcedure, testHandler, RateLimit{Limit: 10, Penalty: 10})
	mp.start(ctx, &loggerTest, p)
	// There is no channel for the request ID "testReqMsgID"

	stream := testStream{}
	reqMsg := newResponseMessage(testReqMsgID, testProcedure, []byte(testResponseData), nil)
	data, _ := reqMsg.Encode()
	stream.data = data
	mp.onResponse(stream)

	idx := slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, "Response message received for unknown request ID") })
	assert.NotEqual(-1, idx)
}

func TestMessageProtocol_RegisterRPCHandler(t *testing.T) {
	assert := assert.New(t)

	testHandler := func(w ResponseWriter, req *Request) {
	}

	mp := newMessageProtocol(testChainID, testVersion)
	err := mp.RegisterRPCHandler(testRPC, testHandler, RateLimit{Limit: 10, Penalty: 10})
	assert.Nil(err)

	assert.NotNil(mp.rpcHandlers[testRPC])
	assert.NotNil(mp.rateLimits[testRPC])

	f1 := *(*unsafe.Pointer)(unsafe.Pointer(&testHandler))
	handler := mp.rpcHandlers[testRPC]
	f2 := *(*unsafe.Pointer)(unsafe.Pointer(&handler))
	assert.True(f1 == f2)
}

func TestMessageProtocol_RegisterRPCHandlerMessageProtocolRunning(t *testing.T) {
	assert := assert.New(t)

	testHandler := func(w ResponseWriter, req *Request) {
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{}
	_ = cfg.insertDefault()
	wg := &sync.WaitGroup{}
	p, _ := newPeer(ctx, wg, logger, []byte{}, cfg)

	mp := newMessageProtocol(testChainID, testVersion)
	mp.start(ctx, logger, p)

	err := mp.RegisterRPCHandler(testRPC, testHandler, RateLimit{Limit: 10, Penalty: 10})
	assert.NotNil(err)
	assert.Equal("cannot register RPC handler after MessageProtocol is started", err.Error())

	_, exist := mp.rpcHandlers[testRPC]
	assert.False(exist)
}

func TestMessageProtocol_RegisterRPCHandlerAlreadyRegistered(t *testing.T) {
	assert := assert.New(t)

	testHandler := func(w ResponseWriter, req *Request) {
	}

	mp := newMessageProtocol(testChainID, testVersion)

	err := mp.RegisterRPCHandler(testRPC, testHandler, RateLimit{Limit: 10, Penalty: 10})
	assert.Nil(err)
	_, exist := mp.rpcHandlers[testRPC]
	assert.True(exist)

	err = mp.RegisterRPCHandler(testRPC, testHandler, RateLimit{Limit: 10, Penalty: 10})
	assert.NotNil(err)
	assert.Equal("rpcHandler testRPC is already registered", err.Error())
}

func TestMessageProtocol_SendRequestMessage(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()

	wg := &sync.WaitGroup{}
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	mp1 := newMessageProtocol(testChainID, testVersion)
	testHandler := func(w ResponseWriter, req *Request) {}
	mp1.RegisterRPCHandler(testRPC, testHandler, RateLimit{Limit: 10, Penalty: 10})
	mp1.start(ctx, logger, p1)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	mp2 := newMessageProtocol(testChainID, testVersion)
	mp2.RegisterRPCHandler(testRPC, func(w ResponseWriter, req *Request) {
		w.Write([]byte("Average RTT with you:"))
	}, RateLimit{Limit: 10, Penalty: 10})
	mp2.start(ctx, logger, p2)
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

	err := p1.Connect(ctx, *p2AddrInfo)
	assert.NoError(err)

	response, err := mp1.request(ctx, p2.ID(), testRPC, []byte(testRequestData))
	assert.Nil(err)
	assert.Equal(p2.ID(), response.PeerID())
	assert.Contains(string(response.Data()), "Average RTT with you:")
}

func TestMessageProtocol_SendRequestMessage_differentVersion(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()

	wg := &sync.WaitGroup{}
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	mp1 := newMessageProtocol(testChainID, testVersion)
	mp1.start(ctx, logger, p1)

	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	mp2 := newMessageProtocol([]byte{9, 9, 9, 9}, "9.9")
	mp2.start(ctx, logger, p2)
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])
	err := p1.Connect(ctx, *p2AddrInfo)
	assert.NoError(err)

	_, err = mp1.request(ctx, p2.ID(), testRPC, []byte(testRequestData))
	assert.ErrorContains(err, "protocol not supported")
}

func TestMessageProtocol_SendRequestMessageRPCHandlerError(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()

	wg := &sync.WaitGroup{}
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	mp1 := newMessageProtocol(testChainID, testVersion)
	testHandler := func(w ResponseWriter, req *Request) {}
	mp1.RegisterRPCHandler(testRPC, testHandler, RateLimit{Limit: 10, Penalty: 10})
	mp1.start(ctx, logger, p1)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	mp2 := newMessageProtocol(testChainID, testVersion)
	mp2.RegisterRPCHandler(testRPC, func(w ResponseWriter, req *Request) {
		w.Error(errors.New("Test RPC handler error!"))
	}, RateLimit{Limit: 10, Penalty: 10})
	mp2.start(ctx, logger, p2)
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

	err := p1.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)

	response, err := mp1.request(ctx, p2.ID(), testRPC, []byte(testRequestData))
	assert.Nil(err)
	assert.Equal(p2.ID(), response.PeerID())
	assert.Equal([]byte{}, response.Data())
	assert.Contains(response.Error().Error(), "Test RPC handler error!")
}

func TestMessageProtocol_SendRequestMessageTimeout(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()

	wg := &sync.WaitGroup{}
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	mp1 := newMessageProtocol(testChainID, testVersion)
	mp1.start(ctx, logger, p1)
	mp1.timeout = time.Millisecond * 20 // Reduce timeout to 20 ms to speed up test
	// Remove response message stream handler to simulate timeout
	p1.host.RemoveStreamHandler(messageProtocolReqID(testChainID, testVersion))
	p1.host.RemoveStreamHandler(messageProtocolResID(testChainID, testVersion))

	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	mp2 := newMessageProtocol(testChainID, testVersion)
	mp2.RegisterRPCHandler(testRPC, func(w ResponseWriter, req *Request) {
		w.Write([]byte("Average RTT with you:"))
	}, RateLimit{Limit: 10, Penalty: 10})
	mp2.start(ctx, logger, p2)
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

	err := p1.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)

	response, err := mp1.request(ctx, p2.ID(), testRPC, []byte(testRequestData))
	assert.Nil(response)
	assert.Equal("timeout", err.Error())
}

func TestMessageProtocol_SendResponseMessage(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()

	wg := &sync.WaitGroup{}
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	mp1 := newMessageProtocol(testChainID, testVersion)
	testHandler := func(w ResponseWriter, req *Request) {}
	mp1.RegisterRPCHandler(testProcedure, testHandler, RateLimit{Limit: 10, Penalty: 10})
	mp1.start(ctx, logger, p1)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	mp2 := newMessageProtocol(testChainID, testVersion)
	mp2.RegisterRPCHandler(testProcedure, testHandler, RateLimit{Limit: 10, Penalty: 10})
	mp2.start(ctx, logger, p2)
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

	err := p1.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)
	ch := make(chan *Response, 1)
	mp2.resCh[testReqMsgID] = ch
	err = mp1.respond(ctx, p2.ID(), testReqMsgID, testProcedure, []byte(testResponseData), errors.New(testError))
	assert.Nil(err)

	select {
	case response := <-ch:
		assert.Equal(p1.ID(), response.PeerID())
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

func TestMessageProtocol_sendMessage(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()
	tmr := TestMessageReceive{done: make(chan any)}

	wg := &sync.WaitGroup{}

	// check sending from matching chainID/version
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfg)

	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	p2.host.SetStreamHandler(messageProtocolReqID(testChainID, testVersion), tmr.onMessageReceive)
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

	_ = p1.Connect(ctx, *p2AddrInfo)
	msg := newRequestMessage(p1.ID(), testProcedure, []byte(testRequestData))
	mp := newMessageProtocol(testChainID, testVersion)
	mp.start(ctx, logger, p1)
	err := mp.send(ctx, p2.ID(), messageProtocolReqID(testChainID, testVersion), msg)
	assert.Nil(err)

	select {
	case <-tmr.done:
		break
	case <-time.After(time.Second * pingTimeout):
		t.Fatalf("timeout occurs, message was not delivered to a peer")
	}

	assert.Contains(tmr.msg, testRequestData)
}

func TestMessageProtocol_sendMessage_differentVersion(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()

	wg := &sync.WaitGroup{}

	// check sending from matching chainID/version
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfg)

	mp := newMessageProtocol(testChainID, testVersion)
	mp.start(ctx, logger, p1)

	// check sending from different chainID/version
	tmr := TestMessageReceive{done: make(chan any)}

	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	p2.host.SetStreamHandler(messageProtocolReqID([]byte{9, 9, 9, 9}, "9.9"), tmr.onMessageReceive)
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

	_ = p1.Connect(ctx, *p2AddrInfo)
	msg := newRequestMessage(p1.ID(), testProcedure, []byte(testRequestData))
	err := mp.send(ctx, p2.ID(), messageProtocolReqID(testChainID, testVersion), msg)
	assert.ErrorContains(err, "protocol not supported")
}

func TestMessageProtocol_RateLimiter(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()

	node1 := NewConnection(cfg)
	testHandler := func(w ResponseWriter, req *Request) {}
	node1.RegisterRPCHandler(testRPC, testHandler, RateLimit{Limit: 5, Penalty: 25})
	node1.MessageProtocol.rateLimiterInterval = time.Millisecond * 100 // Decrease the interval to speed up the test
	err := node1.Start(logger, []byte{})
	assert.Nil(err)
	node1Addrs, _ := node1.MultiAddress()
	node1AddrInfo, _ := AddrInfoFromMultiAddr(node1Addrs[0])
	node1.connGater.intervalCheck = time.Millisecond * 50 // Decrease the interval to speed up the test

	node2 := NewConnection(cfg)
	node2.RegisterRPCHandler(testRPC, func(w ResponseWriter, req *Request) {
		w.Write([]byte("Average RTT with you:"))
	}, RateLimit{Limit: 5, Penalty: 25})
	err = node2.Start(logger, []byte{})
	assert.Nil(err)
	node2Addrs, _ := node2.MultiAddress()
	node2AddrInfo, _ := AddrInfoFromMultiAddr(node2Addrs[0])

	err = node1.Connect(ctx, *node2AddrInfo)
	assert.NoError(err)

	// Wait 4 times for the rate limiter to apply the penalty to node2 (4 * 25 = 100)
	for i := 0; i < 4; i++ {
		// Send 6 requests to node2 (1 too many) that a penalty will be applied
		for j := 0; j < 6; j++ {
			_, err := node1.request(ctx, node2.ID(), testRPC, []byte(testRequestData))
			assert.Nil(err)
		}
		time.Sleep(time.Millisecond * 130) // Wait for the rate limiter to apply the penalty
	}

	// Wait for the rate limiter to ban the peer
	waitForTestCondition(t, func() int { return len(node1.ConnectedPeers()) }, 0, testTimeout)

	// Check that a peer is banned
	assert.Equal(0, len(node1.ConnectedPeers()))

	// Try to connect node1 with node2 (it should not be possible because node2 is banned)
	err = node1.Connect(ctx, *node2AddrInfo)
	assert.NotNil(err)

	// Try to connect node2 with node1 (it should not be possible because node2 is banned)
	err = node2.Connect(ctx, *node1AddrInfo)
	assert.NotNil(err)
}

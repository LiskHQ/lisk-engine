package p2p

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

const testTimeout = time.Second * 3 // Timeout for the test

type testLogger struct {
	log.Logger
	logs []string
}

func (l *testLogger) Debugf(msg string, others ...interface{}) {
	l.logs = append(l.logs, msg)
}

func (l *testLogger) Warningf(msg string, others ...interface{}) {
	l.logs = append(l.logs, msg)
}

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
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{}
	_ = config.InsertDefault()
	p, _ := NewPeer(context.Background(), logger, config)

	mp := NewMessageProtocol()
	mp.Start(context.Background(), logger, p)
	assert.Equal(t, logger, mp.logger)
	assert.Equal(t, p, mp.peer)
}

func TestMessageProtocol_OnRequest(t *testing.T) {
	var tests = []struct {
		name      string
		procedure string
		want      string
	}{
		{"ping request message", "ping", "Request received"},
		{"known peers request message", "knownPeers", "Request received"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := log.NewDefaultProductionLogger()
			loggerTest := testLogger{Logger: logger}
			config := Config{}
			_ = config.InsertDefault()
			p, _ := NewPeer(context.Background(), &loggerTest, config)
			mp := NewMessageProtocol()
			mp.RegisterRPCHandler(tt.procedure, func(w ResponseWriter, req *RequestMsg) {
				mp.logger.Debugf("Request received")
				w.Write([]byte("Test response message"))
			})
			mp.Start(context.Background(), &loggerTest, p)

			stream := testStream{}
			reqMsg := newRequestMessage("TestRemotePeerID", tt.procedure, []byte(""))
			data, _ := reqMsg.Encode()
			stream.data = data
			mp.onRequest(context.Background(), stream)

			idx := slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, "Request message received") })
			assert.NotEqual(t, -1, idx)

			idx = slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, tt.want) })
			assert.NotEqual(t, -1, idx)
		})
	}
}

func TestMessageProtocol_OnResponse(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	loggerTest := testLogger{Logger: logger}
	config := Config{}
	_ = config.InsertDefault()
	p, _ := NewPeer(context.Background(), &loggerTest, config)
	mp := NewMessageProtocol()
	mp.Start(context.Background(), &loggerTest, p)
	ch := make(chan *Response, 1)
	mp.resCh["123456"] = ch

	stream := testStream{}
	reqMsg := newResponseMessage("123456", []byte("Test response message"), errors.New("Test error message"))
	data, _ := reqMsg.Encode()
	stream.data = data
	mp.onResponse(stream)

	select {
	case response := <-ch:
		assert.Equal(t, "Test response message", string(response.Data()))
		assert.Equal(t, "Test error message", response.Error().Error())
		break
	case <-time.After(testTimeout):
		t.Fatalf("timeout occurs, response message was not created and sent to the channel")
	}

	assert.Equal(t, 1, len(mp.resCh))
	idx := slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, "Response message received") })
	assert.NotEqual(t, -1, idx)
}

func TestMessageProtocol_OnResponseUnknownRequestID(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	loggerTest := testLogger{Logger: logger}
	config := Config{}
	_ = config.InsertDefault()
	p, _ := NewPeer(context.Background(), &loggerTest, config)
	mp := NewMessageProtocol()
	mp.Start(context.Background(), &loggerTest, p)
	// There is no channel for the request ID "123456"

	stream := testStream{}
	reqMsg := newResponseMessage("123456", []byte("Test response message"), nil)
	data, _ := reqMsg.Encode()
	stream.data = data
	mp.onResponse(stream)

	idx := slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, "Response message received for unknown request ID") })
	assert.NotEqual(t, -1, idx)
}

func TestMessageProtocol_RegisterRPCHandler(t *testing.T) {
	assert := assert.New(t)

	testHandler := func(w ResponseWriter, req *RequestMsg) {
	}

	mp := NewMessageProtocol()
	err := mp.RegisterRPCHandler("testRPC", testHandler)
	assert.Nil(err)

	assert.NotNil(mp.rpcHandlers["testRPC"])

	f1 := *(*unsafe.Pointer)(unsafe.Pointer(&testHandler))
	handler := mp.rpcHandlers["testRPC"]
	f2 := *(*unsafe.Pointer)(unsafe.Pointer(&handler))
	assert.True(f1 == f2)
}

func TestMessageProtocol_RegisterRPCHandlerMessageProtocolRunning(t *testing.T) {
	assert := assert.New(t)

	testHandler := func(w ResponseWriter, req *RequestMsg) {
	}

	ctx := context.Background()
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{}
	_ = config.InsertDefault()
	p, _ := NewPeer(context.Background(), logger, config)

	mp := NewMessageProtocol()
	mp.Start(ctx, logger, p)

	err := mp.RegisterRPCHandler("testRPC", testHandler)
	assert.NotNil(err)
	assert.Equal("cannot register RPC handler after MessageProtocol is started", err.Error())

	_, exist := mp.rpcHandlers["testRPC"]
	assert.False(exist)
}

func TestMessageProtocol_RegisterRPCHandlerAlreadyRegistered(t *testing.T) {
	assert := assert.New(t)

	testHandler := func(w ResponseWriter, req *RequestMsg) {
	}

	mp := NewMessageProtocol()

	err := mp.RegisterRPCHandler("testRPC", testHandler)
	assert.Nil(err)
	_, exist := mp.rpcHandlers["testRPC"]
	assert.True(exist)

	err = mp.RegisterRPCHandler("testRPC", testHandler)
	assert.NotNil(err)
	assert.Equal("rpcHandler testRPC is already registered", err.Error())
}

func TestMessageProtocol_SendRequestMessage(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{AllowIncomingConnections: true, Addresses: []string{"/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"}}
	_ = config.InsertDefault()

	p1, _ := NewPeer(context.Background(), logger, config)
	mp1 := NewMessageProtocol()
	mp1.Start(context.Background(), logger, p1)
	p2, _ := NewPeer(context.Background(), logger, config)
	mp2 := NewMessageProtocol()
	mp2.RegisterRPCHandler("ping", func(w ResponseWriter, req *RequestMsg) {
		w.Write([]byte("Average RTT with you:"))
	})
	mp2.Start(context.Background(), logger, p2)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	err := p1.Connect(context.Background(), *p2AddrInfo)
	assert.Nil(t, err)

	response, err := mp1.SendRequestMessage(context.Background(), p2.ID(), "ping", []byte("Test protocol request message"))
	assert.Nil(t, err)
	assert.Equal(t, p2.ID().String(), response.PeerID())
	assert.Contains(t, string(response.Data()), "Average RTT with you:")
}

func TestMessageProtocol_SendRequestMessageRPCHandlerError(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{AllowIncomingConnections: true, Addresses: []string{"/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"}}
	_ = config.InsertDefault()

	p1, _ := NewPeer(context.Background(), logger, config)
	mp1 := NewMessageProtocol()
	mp1.Start(context.Background(), logger, p1)
	p2, _ := NewPeer(context.Background(), logger, config)
	mp2 := NewMessageProtocol()
	mp2.RegisterRPCHandler("ping", func(w ResponseWriter, req *RequestMsg) {
		w.Error(errors.New("Test RPC handler error!"))
	})
	mp2.Start(context.Background(), logger, p2)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	err := p1.Connect(context.Background(), *p2AddrInfo)
	assert.Nil(t, err)

	response, err := mp1.SendRequestMessage(context.Background(), p2.ID(), "ping", []byte("Test protocol request message"))
	assert.Nil(t, err)
	assert.Equal(t, p2.ID().String(), response.PeerID())
	assert.Equal(t, []byte{}, response.Data())
	assert.Contains(t, response.Error().Error(), "Test RPC handler error!")
}

func TestMessageProtocol_SendRequestMessageTimeout(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{AllowIncomingConnections: true, Addresses: []string{"/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"}}
	_ = config.InsertDefault()

	p1, _ := NewPeer(context.Background(), logger, config)
	mp1 := NewMessageProtocol()
	mp1.Start(context.Background(), logger, p1)
	mp1.timeout = time.Millisecond * 20 // Reduce timeout to 20 ms to speed up test
	// Remove response message stream handler to simulate timeout
	p1.host.RemoveStreamHandler(messageProtocolResID)
	p2, _ := NewPeer(context.Background(), logger, config)
	mp2 := NewMessageProtocol()
	mp2.Start(context.Background(), logger, p2)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	err := p1.Connect(context.Background(), *p2AddrInfo)
	assert.Nil(t, err)

	response, err := mp1.SendRequestMessage(context.Background(), p2.ID(), "ping", []byte("Test protocol request message"))
	assert.Nil(t, response)
	assert.Equal(t, "timeout", err.Error())
}

func TestMessageProtocol_SendResponseMessage(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{AllowIncomingConnections: true, Addresses: []string{"/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"}}
	_ = config.InsertDefault()

	p1, _ := NewPeer(context.Background(), logger, config)
	mp1 := NewMessageProtocol()
	mp1.Start(context.Background(), logger, p1)
	p2, _ := NewPeer(context.Background(), logger, config)
	mp2 := NewMessageProtocol()
	mp2.Start(context.Background(), logger, p2)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	err := p1.Connect(context.Background(), *p2AddrInfo)
	assert.Nil(t, err)
	ch := make(chan *Response, 1)
	mp2.resCh["123456"] = ch
	err = mp1.SendResponseMessage(context.Background(), p2.ID(), "123456", []byte("Test protocol response message"), errors.New("test error"))
	assert.Nil(t, err)

	select {
	case response := <-ch:
		assert.Equal(t, p1.ID().String(), response.PeerID())
		assert.Equal(t, "Test protocol response message", string(response.Data()))
		assert.Equal(t, "test error", response.Error().Error())
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
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{AllowIncomingConnections: true, Addresses: []string{"/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"}}
	_ = config.InsertDefault()
	tmr := TestMessageReceive{done: make(chan any)}

	p1, _ := NewPeer(context.Background(), logger, config)
	p2, _ := NewPeer(context.Background(), logger, config)
	p2.host.SetStreamHandler(messageProtocolReqID, tmr.onMessageReceive)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	_ = p1.Connect(context.Background(), *p2AddrInfo)
	msg := newRequestMessage(p1.ID(), "ping", []byte("Test protocol message"))
	mp := NewMessageProtocol()
	mp.Start(context.Background(), logger, p1)
	err := mp.sendMessage(context.Background(), p2.ID(), messageProtocolReqID, msg)
	assert.Nil(t, err)

	select {
	case <-tmr.done:
		break
	case <-time.After(time.Second * pingTimeout):
		t.Fatalf("timeout occurs, message was not delivered to a peer")
	}

	assert.Contains(t, tmr.msg, "Test protocol message")
}

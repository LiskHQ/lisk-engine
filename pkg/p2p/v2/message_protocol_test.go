package p2p

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/proto"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

const testTimeout = 3 // Timeout for the test

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

func TestNewMessageProtocol(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	conf := Config{DummyConfigurationFeatureEnable: true}
	p, _ := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)

	mp := NewMessageProtocol(context.Background(), p)
	assert.Equal(t, p, mp.peer)
	assert.Equal(t, 0, len(mp.resCh))
}

func TestOnMessageReqReceive(t *testing.T) {
	var tests = []struct {
		name      string
		procedure MessageRequestType
		want      string
	}{
		{"ping request message", MessageRequestTypePing, "Ping request received"},
		{"known peers request message", MessageRequestTypeKnownPeers, "Get known peers request received"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := log.NewDefaultProductionLogger()
			loggerTest := testLogger{Logger: logger}
			conf := Config{DummyConfigurationFeatureEnable: true}
			p, _ := NewPeer(context.Background(), &loggerTest, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
			mp := NewMessageProtocol(context.Background(), p)

			stream := testStream{}
			reqMsg := newRequestMessage("TestRemotePeerID", tt.procedure, []byte(""))
			data, _ := proto.Marshal(reqMsg)
			stream.data = data
			mp.onMessageReqReceive(stream)

			idx := slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, "Request message received") })
			assert.NotEqual(t, -1, idx)

			idx = slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, tt.want) })
			assert.NotEqual(t, -1, idx)
		})
	}
}

func TestOnMessageResReceive(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	loggerTest := testLogger{Logger: logger}
	conf := Config{DummyConfigurationFeatureEnable: true}
	p, _ := NewPeer(context.Background(), &loggerTest, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	mp := NewMessageProtocol(context.Background(), p)
	ch := make(chan *ResponseMsg, 1)
	mp.resCh["123456"] = ch

	stream := testStream{}
	reqMsg := newResponseMessage("TestRemotePeerID", "123456", []byte("Test response message"))
	data, _ := proto.Marshal(reqMsg)
	stream.data = data
	mp.onMessageResReceive(stream)

	select {
	case response := <-ch:
		assert.Equal(t, "BRTJxkTmkyEaEb2LYQYnEP", response.PeerID)
		assert.Equal(t, "123456", response.ReqMsgID)
		assert.Equal(t, "Test response message", string(response.Data))
		assert.Equal(t, nil, response.Err)
		break
	case <-time.After(time.Second * testTimeout):
		t.Fatalf("timeout occurs, response message was not created and sent to the channel")
	}

	assert.Equal(t, 0, len(mp.resCh))
	idx := slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, "Response message received") })
	assert.NotEqual(t, -1, idx)
}

func TestOnMessageResReceiveNilChannel(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	loggerTest := testLogger{Logger: logger}
	conf := Config{DummyConfigurationFeatureEnable: true}
	p, _ := NewPeer(context.Background(), &loggerTest, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	mp := NewMessageProtocol(context.Background(), p)
	// channel is not created
	mp.resCh["123456"] = nil

	stream := testStream{}
	reqMsg := newResponseMessage("TestRemotePeerID", "123456", []byte("Test response message"))
	data, _ := proto.Marshal(reqMsg)
	stream.data = data
	mp.onMessageResReceive(stream)

	assert.Equal(t, 0, len(mp.resCh))
	idx := slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, "Response message received") })
	assert.NotEqual(t, -1, idx)
}

func TestOnMessageResReceiveUnknownRequestID(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	loggerTest := testLogger{Logger: logger}
	conf := Config{DummyConfigurationFeatureEnable: true}
	p, _ := NewPeer(context.Background(), &loggerTest, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	mp := NewMessageProtocol(context.Background(), p)
	// There is no channel for the request ID "123456"

	stream := testStream{}
	reqMsg := newResponseMessage("TestRemotePeerID", "123456", []byte("Test response message"))
	data, _ := proto.Marshal(reqMsg)
	stream.data = data
	mp.onMessageResReceive(stream)

	idx := slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, "Response message received for unknown request ID") })
	assert.NotEqual(t, -1, idx)
}

func TestSendRequestMessage(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	conf := Config{DummyConfigurationFeatureEnable: true}

	p1, _ := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	mp1 := NewMessageProtocol(context.Background(), p1)
	p2, _ := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	_ = NewMessageProtocol(context.Background(), p2)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	err := p1.Connect(context.Background(), *p2AddrInfo)
	assert.Nil(t, err)
	ch := make(chan *ResponseMsg, 1)
	err = mp1.SendRequestMessage(context.Background(), p2.ID(), MessageRequestTypePing, []byte("Test protocol request message"), ch)
	assert.Nil(t, err)

	select {
	case response := <-ch:
		assert.Equal(t, p2.ID().String(), response.PeerID)
		assert.Contains(t, string(response.Data), "Average RTT with you:")
		assert.Equal(t, nil, response.Err)
		break
	case <-time.After(time.Second * testTimeout):
		t.Fatalf("timeout occurs, response message was not received")
	}
}

func TestSendRequestMessageTimeout(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	conf := Config{DummyConfigurationFeatureEnable: true}

	p1, _ := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	mp1 := NewMessageProtocol(context.Background(), p1)
	mp1.timeout = time.Millisecond * 20 // Reduce timeout to 20 ms to speed up test
	// Remove response message stream handler to simulate timeout
	p1.host.RemoveStreamHandler(messageProtocolResID)
	p2, _ := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	_ = NewMessageProtocol(context.Background(), p2)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	err := p1.Connect(context.Background(), *p2AddrInfo)
	assert.Nil(t, err)
	ch := make(chan *ResponseMsg, 1)
	err = mp1.SendRequestMessage(context.Background(), p2.ID(), MessageRequestTypePing, []byte("Test protocol request message"), ch)
	assert.Nil(t, err)

	select {
	case response := <-ch:
		assert.Equal(t, "timeout", response.Err.Error())
		break
	case <-time.After(time.Second * testTimeout * 2):
		t.Fatalf("timeout occurs, response message was not received")
	}
}

func TestSendResponseMessage(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	conf := Config{DummyConfigurationFeatureEnable: true}

	p1, _ := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	mp1 := NewMessageProtocol(context.Background(), p1)
	p2, _ := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	mp2 := NewMessageProtocol(context.Background(), p2)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	err := p1.Connect(context.Background(), *p2AddrInfo)
	assert.Nil(t, err)
	ch := make(chan *ResponseMsg, 1)
	mp2.resCh["123456"] = ch
	err = mp1.SendResponseMessage(context.Background(), p2.ID(), "123456", []byte("Test protocol response message"))
	assert.Nil(t, err)

	select {
	case response := <-ch:
		assert.Equal(t, p1.ID().String(), response.PeerID)
		assert.Equal(t, "123456", response.ReqMsgID)
		assert.Equal(t, "Test protocol response message", string(response.Data))
		assert.Equal(t, nil, response.Err)
		break
	case <-time.After(time.Second * testTimeout):
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

func TestSendProtoMessage(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	conf := Config{DummyConfigurationFeatureEnable: true}
	tmr := TestMessageReceive{done: make(chan any)}

	p1, _ := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	p2, _ := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	p2.host.SetStreamHandler(messageProtocolReqID, tmr.onMessageReceive)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	_ = p1.Connect(context.Background(), *p2AddrInfo)
	msg := newRequestMessage(p1.ID(), MessageRequestTypePing, []byte("Test protocol message"))
	mp := NewMessageProtocol(context.Background(), p1)
	err := mp.sendProtoMessage(context.Background(), p2.ID(), messageProtocolReqID, msg)
	assert.Nil(t, err)

	select {
	case <-tmr.done:
		break
	case <-time.After(time.Second * pingTimeout):
		t.Fatalf("timeout occurs, message was not delivered to a peer")
	}

	assert.Contains(t, tmr.msg, "Test protocol message")
}

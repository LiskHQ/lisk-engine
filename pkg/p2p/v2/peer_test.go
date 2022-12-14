package p2p

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	conf := Config{DummyConfigurationFeatureEnable: true}
	p, err := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	assert.Nil(t, err)
	assert.NotNil(t, p.host)
	assert.NotNil(t, p.MessageProtocol)
}

func TestClose(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	conf := Config{DummyConfigurationFeatureEnable: true}
	p, _ := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	err := p.host.Close()
	assert.Nil(t, err)
}

func TestConnect(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	conf := Config{DummyConfigurationFeatureEnable: true}
	p1, _ := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	p2, _ := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	err := p1.Connect(context.Background(), *p2AddrInfo)
	assert.Nil(t, err)
	assert.Equal(t, p2.ID(), p1.ConnectedPeers()[0])
}

func TestPingMultiTimes(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	conf := Config{DummyConfigurationFeatureEnable: true}
	p1, _ := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	p2, _ := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	_ = p1.Connect(context.Background(), *p2AddrInfo)
	rtt, err := p1.PingMultiTimes(context.Background(), p2.ID())
	assert.Nil(t, err)
	assert.Equal(t, numOfPingMessages, len(rtt))
}

func TestPing(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	conf := Config{DummyConfigurationFeatureEnable: true}
	p1, _ := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	p2, _ := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	_ = p1.Connect(context.Background(), *p2AddrInfo)
	_, err := p1.Ping(context.Background(), p2.ID())
	assert.Nil(t, err)
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
	p2.host.SetStreamHandler(messageProtocolID, tmr.onMessageReceive)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	_ = p1.Connect(context.Background(), *p2AddrInfo)
	err := p1.sendProtoMessage(context.Background(), p2.ID(), messageProtocolID, "Test protocol message")
	assert.Nil(t, err)

	select {
	case <-tmr.done:
		break
	case <-time.After(time.Second * pingTimeout):
		t.Fatalf("timeout occurs, message was not delivered to a peer")
	}

	assert.Contains(t, tmr.msg, "Test protocol message")
}

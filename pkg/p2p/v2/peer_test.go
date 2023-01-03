package p2p

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

func TestPeer_Create(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	conf := Config{DummyConfigurationFeatureEnable: true}
	p, err := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	assert.Nil(t, err)
	assert.NotNil(t, p.host)
}

func TestPeer_Close(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	conf := Config{DummyConfigurationFeatureEnable: true}
	p, _ := NewPeer(context.Background(), logger, conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	err := p.Close()
	assert.Nil(t, err)
}

func TestPeer_Connect(t *testing.T) {
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

func TestPeer_PingMultiTimes(t *testing.T) {
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

func TestPeer_Ping(t *testing.T) {
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

func TestPeer_PeerSource(t *testing.T) {
	ch := peerSource(context.Background(), 5)

	for i := 0; i < 5; i++ {
		addr := <-ch
		assert.NotNil(t, addr)
		assert.Equal(t, 1, len(addr.Addrs))
	}

	// Test there is no more addresses sent
	addr := <-ch
	assert.NotNil(t, addr)
	assert.Equal(t, 0, len(addr.Addrs))
}

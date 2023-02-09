package p2p

import (
	"context"
	"fmt"
	"testing"
	"time"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

func TestPeer_New(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	config := Config{}
	_ = config.InsertDefault()
	p, err := NewPeer(context.Background(), logger, config)
	assert.Nil(err)
	assert.NotNil(p.host)
	assert.NotNil(p.peerbook)
}

func TestPeer_Close(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{}
	_ = config.InsertDefault()
	p, _ := NewPeer(context.Background(), logger, config)
	err := p.Close()
	assert.Nil(t, err)
}

func TestPeer_Connect(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{AllowIncomingConnections: true, Addresses: []string{"/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"}}
	_ = config.InsertDefault()
	p1, _ := NewPeer(context.Background(), logger, config)
	p2, _ := NewPeer(context.Background(), logger, config)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	err := p1.Connect(context.Background(), *p2AddrInfo)
	assert.Nil(t, err)
	assert.Equal(t, p2.ID(), p1.ConnectedPeers()[0])
}

func TestPeer_ConnectIPIsBlacklisted(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	config1 := Config{AllowIncomingConnections: true, Addresses: []string{"/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"}}
	_ = config1.InsertDefault()
	config2 := Config{AllowIncomingConnections: true, Addresses: []string{"/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"}}
	_ = config2.InsertDefault()
	p1, _ := NewPeer(ctx, logger, config1)
	p2, _ := NewPeer(ctx, logger, config2)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	// p2 is blacklisted
	p1.peerbook.blacklistedIPs = []string{"127.0.0.1"}

	err := p1.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)
	assert.Equal(0, len(p1.ConnectedPeers()))
}

func TestPeer_ConnectIPIsBanned(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	config1 := Config{AllowIncomingConnections: true, Addresses: []string{"/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"}}
	_ = config1.InsertDefault()
	config2 := Config{AllowIncomingConnections: true, Addresses: []string{"/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"}}
	_ = config2.InsertDefault()
	p1, _ := NewPeer(ctx, logger, config1)
	p2, _ := NewPeer(ctx, logger, config2)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	// p2 is banned
	p1.peerbook.bannedIPs = []BannedIP{{ip: "127.0.0.1", timestamp: 123456}}

	err := p1.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)
	assert.Equal(0, len(p1.ConnectedPeers()))
}

func TestPeer_Disconnect(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{AllowIncomingConnections: true, Addresses: []string{"/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"}}
	_ = config.InsertDefault()
	p1, _ := NewPeer(context.Background(), logger, config)
	p2, _ := NewPeer(context.Background(), logger, config)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	err := p1.Connect(context.Background(), *p2AddrInfo)
	assert.Nil(t, err)
	assert.Equal(t, p2.ID(), p1.ConnectedPeers()[0])

	err = p1.Disconnect(context.Background(), p2.ID())
	assert.Nil(t, err)
	assert.Equal(t, 0, len(p1.ConnectedPeers()))
}

func TestPeer_DisallowIncomingConnections(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	config1 := Config{AllowIncomingConnections: true, Addresses: []string{"/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"}}
	_ = config1.InsertDefault()
	config2 := Config{AllowIncomingConnections: false, Addresses: []string{"/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"}}
	_ = config2.InsertDefault()
	p1, _ := NewPeer(context.Background(), logger, config1)
	p2, _ := NewPeer(context.Background(), logger, config2)
	p1Addrs, _ := p1.P2PAddrs()
	p1AddrInfo, _ := PeerInfoFromMultiAddr(p1Addrs[0].String())
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	// p1 is not allowed to connect to p2
	err := p1.Connect(context.Background(), *p2AddrInfo)
	assert.NotNil(t, err)
	assert.Equal(t, 0, len(p1.ConnectedPeers()))

	// p2 is allowed to connect to p1
	err = p2.Connect(context.Background(), *p1AddrInfo)
	assert.Nil(t, err)
	assert.Equal(t, p1.ID(), p2.ConnectedPeers()[0])
	err = p2.Disconnect(context.Background(), p1.ID())
	assert.Nil(t, err)
}

func TestP2PAddrs(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	ip4quic := fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic", 12345)
	ip4tcp := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 12345)
	config := Config{Addresses: []string{ip4quic, ip4tcp}, AllowIncomingConnections: true}
	_ = config.InsertDefault()
	p, err := NewPeer(context.Background(), logger, config)
	assert.Nil(t, err)

	addrs, err := p.P2PAddrs()
	assert.Nil(t, err)
	assert.Equal(t, len(addrs), 2)
}

func TestPeer_PingMultiTimes(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{AllowIncomingConnections: true, Addresses: []string{"/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"}}
	_ = config.InsertDefault()
	p1, _ := NewPeer(context.Background(), logger, config)
	p2, _ := NewPeer(context.Background(), logger, config)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	_ = p1.Connect(context.Background(), *p2AddrInfo)
	rtt, err := p1.PingMultiTimes(context.Background(), p2.ID())
	assert.Nil(t, err)
	assert.Equal(t, numOfPingMessages, len(rtt))
}

func TestPeer_Ping(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{AllowIncomingConnections: true, Addresses: []string{"/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"}}
	_ = config.InsertDefault()
	p1, _ := NewPeer(context.Background(), logger, config)
	p2, _ := NewPeer(context.Background(), logger, config)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	_ = p1.Connect(context.Background(), *p2AddrInfo)
	_, err := p1.Ping(context.Background(), p2.ID())
	assert.Nil(t, err)
}

func TestPeer_PeerSource(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()

	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/80")
	addr2, _ := ma.NewMultiaddr("/ip4/5.6.7.8/udp/90")
	addr3, _ := ma.NewMultiaddr("/ip4/5.6.7.8/udp/90")

	knownPeers := []AddressInfo2{
		{ID: "11111", Addrs: []ma.Multiaddr{addr1}},
		{ID: "22222", Addrs: []ma.Multiaddr{addr2}},
		{ID: "33333", Addrs: []ma.Multiaddr{addr3}},
	}

	config := Config{AllowIncomingConnections: true, Addresses: []string{"/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"}, KnownPeers: knownPeers}
	_ = config.InsertDefault()
	p, _ := NewPeer(ctx, logger, config)

	ch := p.peerSource(ctx, 3)

	for i := 0; i < 3; i++ {
		select {
		case addr := <-ch:
			assert.NotNil(addr)
			assert.Equal(1, len(addr.Addrs))
			break
		case <-time.After(testTimeout):
			t.Fatalf("timeout occurs, peer address info was not sent to the channel")
		}
	}

	// Test there is no more addresses sent
	addr := <-ch
	assert.NotNil(addr)
	assert.Equal(0, len(addr.Addrs))
}

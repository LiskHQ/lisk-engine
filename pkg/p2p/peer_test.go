package p2p

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"

	cfg "github.com/LiskHQ/lisk-engine/pkg/engine/config"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	ps "github.com/LiskHQ/lisk-engine/pkg/p2p/v2/pubsub"
)

func TestPeer_New(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	config := cfg.NetworkConfig{}
	_ = config.InsertDefault()
	wg := &sync.WaitGroup{}
	p, err := NewPeer(context.Background(), wg, logger, config)
	assert.Nil(err)
	assert.NotNil(p.host)
	assert.NotNil(p.peerbook)
}

func TestPeer_Close(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	config := cfg.NetworkConfig{}
	_ = config.InsertDefault()
	wg := &sync.WaitGroup{}
	p, _ := NewPeer(context.Background(), wg, logger, config)
	err := p.Close()
	assert.Nil(err)
}

func TestPeer_Connect(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	config := cfg.NetworkConfig{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = config.InsertDefault()
	wg := &sync.WaitGroup{}
	p1, _ := NewPeer(ctx, wg, logger, config)
	p2, _ := NewPeer(ctx, wg, logger, config)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	err := p1.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)
	assert.Equal(p2.ID(), p1.ConnectedPeers()[0])
}
func TestPeer_Disconnect(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	config := cfg.NetworkConfig{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = config.InsertDefault()
	wg := &sync.WaitGroup{}
	p1, _ := NewPeer(ctx, wg, logger, config)
	p2, _ := NewPeer(ctx, wg, logger, config)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	err := p1.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)
	assert.Equal(p2.ID(), p1.ConnectedPeers()[0])

	err = p1.Disconnect(ctx, p2.ID())
	assert.Nil(err)
	assert.Equal(0, len(p1.ConnectedPeers()))
}

func TestPeer_DisallowIncomingConnections(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	config1 := cfg.NetworkConfig{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = config1.InsertDefault()
	config2 := cfg.NetworkConfig{AllowIncomingConnections: false, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = config2.InsertDefault()
	wg := &sync.WaitGroup{}
	p1, _ := NewPeer(ctx, wg, logger, config1)
	p2, _ := NewPeer(ctx, wg, logger, config2)
	p1Addrs, _ := p1.P2PAddrs()
	p1AddrInfo, _ := PeerInfoFromMultiAddr(p1Addrs[0].String())
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	// p1 is not allowed to connect to p2
	err := p1.Connect(ctx, *p2AddrInfo)
	assert.NotNil(err)
	assert.Equal(0, len(p1.ConnectedPeers()))

	// p2 is allowed to connect to p1
	err = p2.Connect(ctx, *p1AddrInfo)
	assert.Nil(err)
	assert.Equal(p1.ID(), p2.ConnectedPeers()[0])
	err = p2.Disconnect(ctx, p1.ID())
	assert.Nil(err)
}

func TestPeer_TestP2PAddrs(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	ip4quic := fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic", 12345)
	ip4tcp := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 12345)
	config := cfg.NetworkConfig{Addresses: []string{ip4quic, ip4tcp}, AllowIncomingConnections: true}
	_ = config.InsertDefault()
	wg := &sync.WaitGroup{}
	p, err := NewPeer(context.Background(), wg, logger, config)
	assert.Nil(err)

	addrs, err := p.P2PAddrs()
	assert.Nil(err)
	assert.Equal(len(addrs), 2)
}

func TestPeer_BlacklistedPeers(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	wg := &sync.WaitGroup{}

	config := Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = config.InsertDefault()

	// create peer1, will be used for blacklisting via gossipsub
	p1, _ := NewPeer(ctx, wg, logger, config)
	p1Addrs, _ := p1.P2PAddrs()
	p1AddrInfo, _ := PeerInfoFromMultiAddr(p1Addrs[0].String())

	// create peer2, will be used for blacklisting via connection gater (peer ID)
	p2, _ := NewPeer(ctx, wg, logger, config)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	// create peer3, will be used for blacklisting via connection gater (IP address)
	p3, _ := NewPeer(ctx, wg, logger, config)
	p3Addrs, _ := p3.P2PAddrs()
	p3AddrInfo, _ := PeerInfoFromMultiAddr(p3Addrs[0].String())

	config2 := Config{BlacklistedIPs: []string{"127.0.0.1"}} // blacklisting peer2
	p, _ := NewPeer(ctx, wg, logger, config2)

	gs := NewGossipSub()
	_ = gs.RegisterEventHandler(testTopic1, func(event *Event) {})

	sk := ps.NewScoreKeeper()
	err := gs.Start(ctx, wg, logger, p, sk, config2)
	assert.Nil(err)

	_ = p.host.Connect(ctx, *p1AddrInfo) // Connect directly using host to avoid check regarding blacklisted IP address
	_ = p.host.Connect(ctx, *p2AddrInfo)
	_ = p.host.Connect(ctx, *p3AddrInfo)

	gs.ps.BlacklistPeer(p1.ID())                             // blacklisting peer1
	_, err = gs.peer.connGater.addPenalty(p2.ID(), maxScore) // blacklisting peer2
	assert.Nil(err)
	gs.peer.connGater.blockAddr(net.ParseIP(p3Addrs[0].String())) // blacklisting peer3

	peers := p.BlacklistedPeers()
	assert.Equal(3, len(peers))

	idx := slices.IndexFunc(peers, func(s peer.AddrInfo) bool { return strings.Contains(s.ID.String(), p1.ID().String()) })
	assert.NotEqual(-1, idx)

	idx = slices.IndexFunc(peers, func(s peer.AddrInfo) bool { return strings.Contains(s.ID.String(), p2.ID().String()) })
	assert.NotEqual(-1, idx)

	idx = slices.IndexFunc(peers, func(s peer.AddrInfo) bool { return strings.Contains(s.ID.String(), p3.ID().String()) })
	assert.NotEqual(-1, idx)
}

func TestPeer_PingMultiTimes(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	config := cfg.NetworkConfig{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = config.InsertDefault()
	wg := &sync.WaitGroup{}
	p1, _ := NewPeer(ctx, wg, logger, config)
	p2, _ := NewPeer(ctx, wg, logger, config)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	_ = p1.Connect(ctx, *p2AddrInfo)
	rtt, err := p1.PingMultiTimes(ctx, p2.ID())
	assert.Nil(err)
	assert.Equal(numOfPingMessages, len(rtt))
}

func TestPeer_Ping(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	config := cfg.NetworkConfig{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = config.InsertDefault()
	wg := &sync.WaitGroup{}
	p1, _ := NewPeer(ctx, wg, logger, config)
	p2, _ := NewPeer(ctx, wg, logger, config)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	_ = p1.Connect(ctx, *p2AddrInfo)
	_, err := p1.Ping(ctx, p2.ID())
	assert.Nil(err)
}

func TestPeer_PeerSource(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	wg := &sync.WaitGroup{}

	config := cfg.NetworkConfig{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = config.InsertDefault()

	// Peer 1
	p1, _ := NewPeer(ctx, wg, logger, config)
	p1Addrs, _ := p1.P2PAddrs()
	p1AddrInfo, _ := PeerInfoFromMultiAddr(p1Addrs[0].String())

	// Peer 2
	p2, _ := NewPeer(ctx, wg, logger, config)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	// Peer 3
	p3, _ := NewPeer(ctx, wg, logger, config)
	p3Addrs, _ := p3.P2PAddrs()
	p3AddrInfo, _ := PeerInfoFromMultiAddr(p3Addrs[0].String())

	// Peer which will be connected to other peers
	p, _ := NewPeer(ctx, wg, logger, config)
	err := p.Connect(ctx, *p1AddrInfo)
	assert.Nil(err)
	err = p.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)
	err = p.Connect(ctx, *p3AddrInfo)
	assert.Nil(err)

	// Test peer source
	ch := p.peerSource(ctx, 3)

	for i := 0; i < 3; i++ {
		select {
		case addr := <-ch:
			assert.NotNil(addr)
			assert.Equal(2, len(addr.Addrs))
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

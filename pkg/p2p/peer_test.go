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
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

func TestPeer_New(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{}
	_ = cfg.insertDefault()
	wg := &sync.WaitGroup{}
	p, err := newPeer(context.Background(), wg, logger, []byte{}, cfg)
	assert.Nil(err)
	assert.NotNil(p.host)
	assert.NotNil(p.peerbook)
}

func TestPeer_NewStaticPeerID(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{}
	_ = cfg.insertDefault()
	wg := &sync.WaitGroup{}
	p1, err := newPeer(context.Background(),
		wg,
		logger,
		[]byte{1, 2, 3, 4, 5, 6, 7, 8},
		cfg)
	assert.Nil(err)
	assert.Equal("12D3KooWKwTLPoFzjLMuUAM8rSiUDUnJFhpMLtLQ7zKq5cCPoNhQ", p1.ID().Pretty())

	p2, err := newPeer(context.Background(),
		wg,
		logger,
		[]byte{1, 2, 3, 4, 5, 6, 7, 8,
			9, 10, 11, 12, 13, 14, 15, 16,
			17, 18, 19, 20, 21, 22, 23, 24,
			25, 26, 27, 28, 29, 30, 31, 32},
		cfg)
	assert.Nil(err)
	assert.Equal("12D3KooWGW7EBxs51H6huNYsJhCgrk7xRHmURiQ2w3FoVp9bekVi", p2.ID().Pretty())

	p3, err := newPeer(context.Background(),
		wg,
		logger,
		[]byte{},
		cfg)
	assert.Nil(err)

	p4, err := newPeer(context.Background(),
		wg,
		logger,
		[]byte{},
		cfg)
	assert.Nil(err)

	assert.NotEqual(p3.ID().Pretty(), p4.ID().Pretty())
}

func TestPeer_Close(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{}
	_ = cfg.insertDefault()
	wg := &sync.WaitGroup{}
	p, _ := newPeer(context.Background(), wg, logger, []byte{}, cfg)
	err := p.close()
	assert.Nil(err)
}

func TestPeer_Connect(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()
	wg := &sync.WaitGroup{}
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

	err := p1.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)
	assert.Equal(p2.ID(), p1.ConnectedPeers()[0])
}
func TestPeer_Disconnect(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()
	wg := &sync.WaitGroup{}
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

	err := p1.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)
	assert.Equal(p2.ID(), p1.ConnectedPeers()[0])

	err = p1.Disconnect(p2.ID())
	assert.Nil(err)
	assert.Equal(0, len(p1.ConnectedPeers()))
}

func TestPeer_DisallowIncomingConnections(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfg1 := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg1.insertDefault()
	cfg2 := &Config{AllowIncomingConnections: false, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg2.insertDefault()
	wg := &sync.WaitGroup{}
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfg1)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfg2)
	p1Addrs, _ := p1.MultiAddress()
	p1AddrInfo, _ := AddrInfoFromMultiAddr(p1Addrs[0])
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

	// p1 is not allowed to connect to p2
	err := p1.Connect(ctx, *p2AddrInfo)
	assert.NotNil(err)
	assert.Equal(0, len(p1.ConnectedPeers()))

	// p2 is allowed to connect to p1
	err = p2.Connect(ctx, *p1AddrInfo)
	assert.Nil(err)
	assert.Equal(p1.ID(), p2.ConnectedPeers()[0])
	err = p2.Disconnect(p1.ID())
	assert.Nil(err)
}

func TestPeer_New_ConnectionManager(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfg := Config{AllowIncomingConnections: true,
		Addresses:           []string{testIPv4TCP, testIPv4UDP},
		MinNumOfConnections: 2,
		MaxNumOfConnections: 4}
	_ = cfg.insertDefault()

	// Adjust the connection manager options to make the test run faster
	connMgrOptions = []connmgr.Option{
		connmgr.WithGracePeriod(time.Nanosecond),
		connmgr.WithSilencePeriod(time.Second),
	}

	wg := &sync.WaitGroup{}
	p, err := newPeer(ctx, wg, logger, []byte{}, &cfg)
	assert.Nil(err)

	// Create four new peers and connect them to our peer
	p1, _ := newPeer(ctx, wg, logger, []byte{}, &cfg)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, &cfg)
	p3, _ := newPeer(ctx, wg, logger, []byte{}, &cfg)
	p4, _ := newPeer(ctx, wg, logger, []byte{}, &cfg)
	p1Addrs, _ := p1.MultiAddress()
	p1AddrInfo, _ := AddrInfoFromMultiAddr(p1Addrs[0])
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])
	p3Addrs, _ := p3.MultiAddress()
	p3AddrInfo, _ := AddrInfoFromMultiAddr(p3Addrs[0])
	p4Addrs, _ := p4.MultiAddress()
	p4AddrInfo, _ := AddrInfoFromMultiAddr(p4Addrs[0])

	err = p.Connect(ctx, *p1AddrInfo)
	assert.Nil(err)
	err = p.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)
	err = p.Connect(ctx, *p3AddrInfo)
	assert.Nil(err)
	err = p.Connect(ctx, *p4AddrInfo)
	assert.Nil(err)

	// Check if the number of connected peers is the same as the one we have connected
	assert.Equal(4, len(p.ConnectedPeers()))

	// Wait for the connection manager to disconnect one of the peers
	time.Sleep(time.Second + time.Millisecond*200)

	// Check that the number of connected peers is equal to the max number of connections
	assert.Equal(2, len(p.ConnectedPeers()))
}

func TestPeer_TestP2PAddrs(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	ip4quic := fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic", 12345)
	ip4tcp := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 12345)
	cfg := &Config{Addresses: []string{ip4quic, ip4tcp}, AllowIncomingConnections: true}
	_ = cfg.insertDefault()
	wg := &sync.WaitGroup{}
	p, err := newPeer(context.Background(), wg, logger, []byte{}, cfg)
	assert.Nil(err)

	addrs, err := p.MultiAddress()
	assert.Nil(err)
	assert.Equal(len(addrs), 2)
}

func TestPeer_BlacklistedPeers(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	wg := &sync.WaitGroup{}

	cfg1 := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg1.insertDefault()

	// create peer1, will be used for blacklisting via gossipsub
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfg1)
	p1Addrs, _ := p1.MultiAddress()
	p1AddrInfo, _ := AddrInfoFromMultiAddr(p1Addrs[0])

	// create peer2, will be used for blacklisting via connection gater (peer ID)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfg1)
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

	// create peer3, will be used for blacklisting via connection gater (IP address)
	p3, _ := newPeer(ctx, wg, logger, []byte{}, cfg1)
	p3Addrs, _ := p3.MultiAddress()
	p3AddrInfo, _ := AddrInfoFromMultiAddr(p3Addrs[0])

	cfg2 := &Config{BlacklistedIPs: []string{"127.0.0.1"}} // blacklisting peer2
	p, _ := newPeer(ctx, wg, logger, []byte{}, cfg2)

	gs := newGossipSub(testChainID, testVersion)
	_ = gs.RegisterEventHandler(testTopic1, func(event *Event) {}, nil)

	sk := newScoreKeeper()
	err := gs.start(ctx, wg, logger, p, sk, cfg2)
	assert.Nil(err)

	_ = p.host.Connect(ctx, *p1AddrInfo) // Connect directly using host to avoid check regarding blacklisted IP address
	_ = p.host.Connect(ctx, *p2AddrInfo)
	_ = p.host.Connect(ctx, *p3AddrInfo)

	gs.ps.BlacklistPeer(p1.ID())                                    // blacklisting peer1
	_, err = gs.peer.connGater.addPenalty(p2.ID(), MaxPenaltyScore) // blacklisting peer2
	assert.Nil(err)
	gs.peer.connGater.blockAddr(net.ParseIP(p3Addrs[0])) // blacklisting peer3

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
	cfg := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()
	wg := &sync.WaitGroup{}
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

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
	cfg := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()
	wg := &sync.WaitGroup{}
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

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

	cfg := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()

	// Peer 1
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	p1Addrs, _ := p1.MultiAddress()
	p1AddrInfo, _ := AddrInfoFromMultiAddr(p1Addrs[0])

	// Peer 2
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

	// Peer 3
	p3, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	p3Addrs, _ := p3.MultiAddress()
	p3AddrInfo, _ := AddrInfoFromMultiAddr(p3Addrs[0])

	// Peer which will be connected to other peers
	p, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
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

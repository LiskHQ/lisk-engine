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

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

func TestPeer_New(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	cfgNet := &Config{}
	_ = cfgNet.insertDefault()
	wg := &sync.WaitGroup{}
	p, err := newPeer(context.Background(), wg, logger, []byte{}, cfgNet)
	assert.Nil(err)
	assert.NotNil(p.host)
	assert.NotNil(p.peerbook)
}

func TestPeer_NewStaticPeerID(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	cfgNet := &Config{}
	_ = cfgNet.insertDefault()
	wg := &sync.WaitGroup{}
	p1, err := newPeer(context.Background(),
		wg,
		logger,
		[]byte{1, 2, 3, 4, 5, 6, 7, 8},
		cfgNet)
	assert.Nil(err)
	assert.Equal("12D3KooWKwTLPoFzjLMuUAM8rSiUDUnJFhpMLtLQ7zKq5cCPoNhQ", p1.ID().Pretty())

	p2, err := newPeer(context.Background(),
		wg,
		logger,
		[]byte{1, 2, 3, 4, 5, 6, 7, 8,
			9, 10, 11, 12, 13, 14, 15, 16,
			17, 18, 19, 20, 21, 22, 23, 24,
			25, 26, 27, 28, 29, 30, 31, 32},
		cfgNet)
	assert.Nil(err)
	assert.Equal("12D3KooWGW7EBxs51H6huNYsJhCgrk7xRHmURiQ2w3FoVp9bekVi", p2.ID().Pretty())

	p3, err := newPeer(context.Background(),
		wg,
		logger,
		[]byte{},
		cfgNet)
	assert.Nil(err)

	p4, err := newPeer(context.Background(),
		wg,
		logger,
		[]byte{},
		cfgNet)
	assert.Nil(err)

	assert.NotEqual(p3.ID().Pretty(), p4.ID().Pretty())
}

func TestPeer_Close(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	cfgNet := &Config{}
	_ = cfgNet.insertDefault()
	wg := &sync.WaitGroup{}
	p, _ := newPeer(context.Background(), wg, logger, []byte{}, cfgNet)
	err := p.close()
	assert.Nil(err)
}

func TestPeer_Connect(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfgNet := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfgNet.insertDefault()
	wg := &sync.WaitGroup{}
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet)
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
	cfgNet := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfgNet.insertDefault()
	wg := &sync.WaitGroup{}
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet)
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
	cfgNet1 := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfgNet1.insertDefault()
	cfgNet2 := &Config{AllowIncomingConnections: false, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfgNet2.insertDefault()
	wg := &sync.WaitGroup{}
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet1)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet2)
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

func TestPeer_TestP2PAddrs(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	ip4quic := fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic", 12345)
	ip4tcp := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 12345)
	cfgNet := &Config{Addresses: []string{ip4quic, ip4tcp}, AllowIncomingConnections: true}
	_ = cfgNet.insertDefault()
	wg := &sync.WaitGroup{}
	p, err := newPeer(context.Background(), wg, logger, []byte{}, cfgNet)
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

	cfgNet1 := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfgNet1.insertDefault()

	// create peer1, will be used for blacklisting via gossipsub
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet1)
	p1Addrs, _ := p1.MultiAddress()
	p1AddrInfo, _ := AddrInfoFromMultiAddr(p1Addrs[0])

	// create peer2, will be used for blacklisting via connection gater (peer ID)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet1)
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

	// create peer3, will be used for blacklisting via connection gater (IP address)
	p3, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet1)
	p3Addrs, _ := p3.MultiAddress()
	p3AddrInfo, _ := AddrInfoFromMultiAddr(p3Addrs[0])

	cfgNet2 := &Config{BlacklistedIPs: []string{"127.0.0.1"}} // blacklisting peer2
	p, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet2)

	gs := newGossipSub(testChainID, testVersion)
	_ = gs.RegisterEventHandler(testTopic1, func(event *Event) {}, nil)

	sk := newScoreKeeper()
	err := gs.start(ctx, wg, logger, p, sk, cfgNet2)
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
	cfgNet := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfgNet.insertDefault()
	wg := &sync.WaitGroup{}
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet)
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
	cfgNet := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfgNet.insertDefault()
	wg := &sync.WaitGroup{}
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet)
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

	cfgNet := &Config{AllowIncomingConnections: true, Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfgNet.insertDefault()

	// Peer 1
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet)
	p1Addrs, _ := p1.MultiAddress()
	p1AddrInfo, _ := AddrInfoFromMultiAddr(p1Addrs[0])

	// Peer 2
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet)
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

	// Peer 3
	p3, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet)
	p3Addrs, _ := p3.MultiAddress()
	p3AddrInfo, _ := AddrInfoFromMultiAddr(p3Addrs[0])

	// Peer which will be connected to other peers
	p, _ := newPeer(ctx, wg, logger, []byte{}, cfgNet)
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

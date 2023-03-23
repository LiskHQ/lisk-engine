package p2p

import (
	"context"
	"sync"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

func TestDiscovery_NextValidPeer(t *testing.T) {
	assert := assert.New(t)

	peer1, err := AddrInfoFromMultiAddr("/ip4/127.0.0.1/tcp/46371/p2p/12D3KooWJG7D1qKgQFB6J36hyBqUmjFQL3JV7GWbV5WWcaFSdtr2")
	assert.Nil(err)
	peer2, err := AddrInfoFromMultiAddr("/ip4/127.0.0.1/tcp/4637/p2p/12D3KooWGZiHSKRQUsyki2iDLhcTUUqC9QS4m4Ng466VWVmhbKff")
	assert.Nil(err)
	peer3, err := AddrInfoFromMultiAddr("/ip4/127.0.0.1/tcp/463/p2p/12D3KooWLWEWJf94qaw3okyPNta85cgUnmS73g35trKBH4DdqgUM")
	assert.Nil(err)
	peer4, err := AddrInfoFromMultiAddr("/ip4/127.0.0.1/tcp/463/p2p/12D3KooWKYpHFG4ZAahNZ5ovbKc8gTzdmGiqwH2fL3tXAbE4VqFF")
	assert.Nil(err)
	peer5, err := AddrInfoFromMultiAddr("/ip4/127.0.0.1/tcp/46/p2p/12D3KooWQuH3Zyfjue27bNwxBQ5kuK2DkdEGfedm3M9kPj8NxY4t")
	assert.Nil(err)

	peers := []peer.AddrInfo{*peer1, *peer2, *peer3, *peer4, *peer5}

	// Create a new peer
	logger, _ := log.NewDefaultProductionLogger()
	wg := &sync.WaitGroup{}
	cfg := &Config{}
	_ = cfg.insertDefault()
	peer, _ := newPeer(context.Background(), wg, logger, []byte{}, cfg)

	d := Discovery{peer: peer}

	peerIndex := len(peers)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(4, peerIndex)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(3, peerIndex)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(2, peerIndex)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(1, peerIndex)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(0, peerIndex)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(-1, peerIndex)
}

func TestDiscovery_NextValidPeer_SkipOurself(t *testing.T) {
	assert := assert.New(t)

	// Create a new peer
	logger, _ := log.NewDefaultProductionLogger()
	wg := &sync.WaitGroup{}
	cfg := &Config{}
	_ = cfg.insertDefault()
	p, _ := newPeer(context.Background(), wg, logger, []byte{}, cfg)
	pInfo := peer.AddrInfo{
		ID:    p.ID(),
		Addrs: p.host.Addrs(),
	}

	peer1, err := AddrInfoFromMultiAddr("/ip4/127.0.0.10/tcp/46371/p2p/12D3KooWJG7D1qKgQFB6J36hyBqUmjFQL3JV7GWbV5WWcaFSdtr2")
	assert.Nil(err)
	peer2, err := AddrInfoFromMultiAddr("/ip4/127.0.0.20/tcp/4637/p2p/12D3KooWGZiHSKRQUsyki2iDLhcTUUqC9QS4m4Ng466VWVmhbKff")
	assert.Nil(err)
	peer3, err := AddrInfoFromMultiAddr("/ip4/127.0.0.30/tcp/463/p2p/12D3KooWLWEWJf94qaw3okyPNta85cgUnmS73g35trKBH4DdqgUM")
	assert.Nil(err)
	peer4, err := AddrInfoFromMultiAddr("/ip4/127.0.0.40/tcp/463/p2p/12D3KooWKYpHFG4ZAahNZ5ovbKc8gTzdmGiqwH2fL3tXAbE4VqFF")
	assert.Nil(err)
	peer5, err := AddrInfoFromMultiAddr("/ip4/127.0.0.50/tcp/46/p2p/12D3KooWQuH3Zyfjue27bNwxBQ5kuK2DkdEGfedm3M9kPj8NxY4t")
	assert.Nil(err)

	// Add ourself to the list of peers two times
	peers := []peer.AddrInfo{*peer1, *peer2, pInfo, *peer3, *peer4, pInfo, *peer5}

	d := Discovery{peer: p}

	// Check that we skip ourself and that there are five available peers
	peerIndex := len(peers)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(6, peerIndex)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(4, peerIndex)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(3, peerIndex)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(1, peerIndex)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(0, peerIndex)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(-1, peerIndex)
}

func TestDiscovery_NextValidPeer_ConnectedPeers(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	wg := &sync.WaitGroup{}

	cfg := &Config{Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()

	// Create two peers
	p1, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	p1Addrs, _ := p1.MultiAddress()
	p1AddrInfo, _ := AddrInfoFromMultiAddr(p1Addrs[0])
	p1Info := peer.AddrInfo{
		ID:    p1.ID(),
		Addrs: p1.host.Addrs(),
	}
	p2Info := peer.AddrInfo{
		ID:    p2.ID(),
		Addrs: p2.host.Addrs(),
	}
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

	// Peer which will be connected to other peers
	p, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	err := p.Connect(ctx, *p1AddrInfo)
	assert.Nil(err)
	err = p.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)

	// Check if the number of connected peers is the same as the one we have connected to
	assert.Equal(2, len(p.ConnectedPeers()))

	peer1, err := AddrInfoFromMultiAddr("/ip4/127.0.0.10/tcp/46371/p2p/12D3KooWJG7D1qKgQFB6J36hyBqUmjFQL3JV7GWbV5WWcaFSdtr2")
	assert.Nil(err)
	peer2, err := AddrInfoFromMultiAddr("/ip4/127.0.0.20/tcp/4637/p2p/12D3KooWGZiHSKRQUsyki2iDLhcTUUqC9QS4m4Ng466VWVmhbKff")
	assert.Nil(err)
	peer3, err := AddrInfoFromMultiAddr("/ip4/127.0.0.30/tcp/463/p2p/12D3KooWLWEWJf94qaw3okyPNta85cgUnmS73g35trKBH4DdqgUM")
	assert.Nil(err)
	peer4, err := AddrInfoFromMultiAddr("/ip4/127.0.0.40/tcp/463/p2p/12D3KooWKYpHFG4ZAahNZ5ovbKc8gTzdmGiqwH2fL3tXAbE4VqFF")
	assert.Nil(err)
	peer5, err := AddrInfoFromMultiAddr("/ip4/127.0.0.50/tcp/46/p2p/12D3KooWQuH3Zyfjue27bNwxBQ5kuK2DkdEGfedm3M9kPj8NxY4t")
	assert.Nil(err)

	// Add two connected peers to the list
	peers := []peer.AddrInfo{*peer1, *peer2, p1Info, *peer3, p2Info, *peer4, *peer5}

	d := Discovery{peer: p}

	// Check that we skip all connected peers and that there are five available peers
	peerIndex := len(peers)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(6, peerIndex)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(5, peerIndex)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(3, peerIndex)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(1, peerIndex)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(0, peerIndex)
	peerIndex = d.nextValidPeer(peers, peerIndex)
	assert.Equal(-1, peerIndex)
}

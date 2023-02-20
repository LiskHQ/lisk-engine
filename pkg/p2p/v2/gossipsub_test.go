package p2p

import (
	"context"
	"strings"
	"sync"
	"testing"
	"unsafe"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	ps "github.com/LiskHQ/lisk-engine/pkg/p2p/v2/pubsub"
)

func TestGossipSub_NewGossipSub(t *testing.T) {
	assert := assert.New(t)

	gs := NewGossipSub()
	assert.NotNil(gs)
	assert.NotNil(gs.topics)
	assert.NotNil(gs.subscriptions)
	assert.NotNil(gs.eventHandlers)
}

func TestGossipSub_Start(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{}
	_ = config.InsertDefault()
	p, _ := NewPeer(ctx, wg, logger, config)
	sk := ps.NewScoreKeeper()

	gs := NewGossipSub()
	err := gs.RegisterEventHandler(testTopic1, func(event *Event) {})
	assert.Nil(err)

	err = gs.Start(ctx, wg, logger, p, sk, config)
	assert.Nil(err)

	assert.NotNil(gs.logger)
	assert.NotNil(gs.peer)
	assert.NotNil(gs.ps)
	assert.NotNil(gs.blacklist)

	assert.Equal(1, len(gs.topics))
	assert.Equal(1, len(gs.subscriptions))
	_, exist := gs.topics[testTopic1]
	assert.True(exist)
	_, exist = gs.subscriptions[testTopic1]
	assert.True(exist)
}

func TestGossipSub_CreateSubscriptionHandlers(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	wg := &sync.WaitGroup{}

	gs := NewGossipSub()
	gs.logger = logger

	gs.RegisterEventHandler(testTopic1, func(event *Event) {})
	gs.RegisterEventHandler(testTopic2, func(event *Event) {})
	gs.RegisterEventHandler(testTopic3, func(event *Event) {})

	host, _ := libp2p.New()
	gossipSub, _ := pubsub.NewGossipSub(ctx, host)
	gs.ps = gossipSub

	err := gs.createSubscriptionHandlers(ctx, wg)
	assert.Nil(err)

	assert.Equal(3, len(gs.topics))
	assert.Equal(3, len(gs.subscriptions))

	_, exist := gs.topics[testTopic1]
	assert.True(exist)
	_, exist = gs.subscriptions[testTopic1]
	assert.True(exist)

	_, exist = gs.topics[testTopic2]
	assert.True(exist)
	_, exist = gs.subscriptions[testTopic2]
	assert.True(exist)

	_, exist = gs.topics[testTopic3]
	assert.True(exist)
	_, exist = gs.subscriptions[testTopic3]
	assert.True(exist)
}

func TestGossipSub_RegisterEventHandler(t *testing.T) {
	assert := assert.New(t)

	testHandler := func(event *Event) {
	}

	gs := NewGossipSub()
	err := gs.RegisterEventHandler(testEvent, testHandler)
	assert.Nil(err)

	_, exist := gs.topics[testEvent]
	assert.True(exist)
	_, exist = gs.subscriptions[testEvent]
	assert.True(exist)

	assert.NotNil(gs.eventHandlers[testEvent])

	f1 := *(*unsafe.Pointer)(unsafe.Pointer(&testHandler))
	handler := gs.eventHandlers[testEvent]
	f2 := *(*unsafe.Pointer)(unsafe.Pointer(&handler))
	assert.True(f1 == f2)
}

func TestGossipSub_RegisterEventHandlerGossipSubRunning(t *testing.T) {
	assert := assert.New(t)

	testHandler := func(event *Event) {
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{}
	_ = config.InsertDefault()
	p, _ := NewPeer(ctx, wg, logger, config)
	sk := ps.NewScoreKeeper()

	gs := NewGossipSub()
	gs.Start(ctx, wg, logger, p, sk, config)

	err := gs.RegisterEventHandler(testEvent, testHandler)
	assert.NotNil(err)
	assert.Equal(err, ErrGossipSubIsRunning)

	_, exist := gs.topics[testEvent]
	assert.False(exist)
	_, exist = gs.subscriptions[testEvent]
	assert.False(exist)
	_, exist = gs.eventHandlers[testEvent]
	assert.False(exist)
}

func TestGossipSub_RegisterEventHandlerAlreadyRegistered(t *testing.T) {
	assert := assert.New(t)

	testHandler := func(event *Event) {
	}

	gs := NewGossipSub()

	err := gs.RegisterEventHandler(testEvent, testHandler)
	assert.Nil(err)
	_, exist := gs.topics[testEvent]
	assert.True(exist)
	_, exist = gs.subscriptions[testEvent]
	assert.True(exist)
	_, exist = gs.eventHandlers[testEvent]
	assert.True(exist)

	err = gs.RegisterEventHandler(testEvent, testHandler)
	assert.NotNil(err)
	assert.Equal(err, ErrDuplicateHandler)
}

func TestGossipSub_Publish(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{}
	_ = config.InsertDefault()
	p, _ := NewPeer(ctx, wg, logger, config)
	sk := ps.NewScoreKeeper()

	gs := NewGossipSub()
	gs.RegisterEventHandler(testTopic1, func(event *Event) {})

	err := gs.RegisterTopicValidator(testTopic1, testMV)
	assert.Equal(err, ErrGossipSubIsNotRunnig)

	gs.Start(ctx, wg, logger, p, sk, config)
	assert.Nil(gs.RegisterTopicValidator(testTopic1, testMV))

	err = gs.Publish(ctx, testTopic1, msg)
	assert.Nil(err)

	err = gs.Publish(ctx, testTopic1, invalidMsg)
	assert.EqualError(err, pubsub.RejectValidationFailed)
	// check the validation nofify once
	err = gs.Publish(ctx, testTopic1, invalidMsg)
	assert.Nil(err)
}

func TestGossipSub_PublishTopicNotFound(t *testing.T) {
	assert := assert.New(t)

	gs := NewGossipSub()

	err := gs.Publish(context.Background(), testTopic1, msg)
	assert.Equal(err, ErrTopicNotFound)
}

func TestGossipSub_BlacklistedPeers(t *testing.T) {
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

	// create peer2, will be used for blacklisting via IP address
	p2, _ := NewPeer(ctx, wg, logger, config)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	config2 := Config{BlacklistedIPs: []string{"127.0.0.1"}} // blacklisting peer2
	p3, _ := NewPeer(ctx, wg, logger, config2)

	gs := NewGossipSub()
	_ = gs.RegisterEventHandler(testTopic1, func(event *Event) {})

	sk := ps.NewScoreKeeper()
	err := gs.Start(ctx, wg, logger, p3, sk, config2)
	assert.Nil(err)

	_ = p3.host.Connect(ctx, *p1AddrInfo) // Connect directly using host to avoid check regarding blacklisted IP address
	_ = p3.host.Connect(ctx, *p2AddrInfo)

	gs.ps.BlacklistPeer(p1.ID()) // blacklisting peer1

	peers := gs.BlacklistedPeers()
	assert.Equal(2, len(peers))

	idx := slices.IndexFunc(peers, func(s peer.AddrInfo) bool { return strings.Contains(s.ID.String(), p1.ID().String()) })
	assert.NotEqual(-1, idx)

	idx = slices.IndexFunc(peers, func(s peer.AddrInfo) bool { return strings.Contains(s.ID.String(), p2.ID().String()) })
	assert.NotEqual(-1, idx)
}

func TestGossipSub_BlacklistedPeersGossipSubNotRunning(t *testing.T) {
	assert := assert.New(t)

	gs := NewGossipSub()

	peers := gs.BlacklistedPeers()
	assert.Equal(0, len(peers))
}

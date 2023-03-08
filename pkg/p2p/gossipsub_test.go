package p2p

import (
	"context"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/log"
)

var (
	testMessageData    = []byte("testMessageData")
	testMessageInvalid = []byte("testMessageInvalid")
)

func TestGetMessageID(t *testing.T) {
	assert := assert.New(t)

	msg := pubsub_pb.Message{}
	hash := getMessageID(&msg)
	expected := codec.Hex(crypto.Hash([]byte{})).String()
	assert.Equal(expected, hash)
}

func TestGossipSub_NewGossipSub(t *testing.T) {
	assert := assert.New(t)

	gs := newGossipSub(testChainID, testVersion)
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
	cfg := &Config{}
	_ = cfg.insertDefault()
	p, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	sk := newScoreKeeper()

	gs := newGossipSub(testChainID, testVersion)
	err := gs.RegisterEventHandler(testTopic1, func(event *Event) {}, nil)
	assert.Nil(err)

	err = gs.start(ctx, wg, logger, p, sk, cfg)
	assert.Nil(err)

	assert.NotNil(gs.logger)
	assert.NotNil(gs.peer)
	assert.NotNil(gs.ps)

	assert.Equal(1, len(gs.topics))
	assert.Equal(1, len(gs.subscriptions))
	_, exist := gs.topics[gs.formatTopic(testTopic1)]
	assert.True(exist)
	_, exist = gs.subscriptions[gs.formatTopic(testTopic1)]
	assert.True(exist)
}

func TestGossipSub_CreateSubscriptionHandlers(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	wg := &sync.WaitGroup{}

	gs := newGossipSub(testChainID, testVersion)
	gs.logger = logger

	gs.RegisterEventHandler(testTopic1, func(event *Event) {}, nil)
	gs.RegisterEventHandler(testTopic2, func(event *Event) {}, nil)
	gs.RegisterEventHandler(testTopic3, func(event *Event) {}, nil)

	host, _ := libp2p.New()
	gossipSub, _ := pubsub.NewGossipSub(ctx, host)
	gs.ps = gossipSub

	err := gs.createSubscriptionHandlers(ctx, wg)
	assert.Nil(err)

	assert.Equal(3, len(gs.topics))
	assert.Equal(3, len(gs.subscriptions))

	_, exist := gs.topics[gs.formatTopic(testTopic1)]
	assert.True(exist)
	_, exist = gs.subscriptions[gs.formatTopic(testTopic1)]
	assert.True(exist)

	_, exist = gs.topics[gs.formatTopic(testTopic2)]
	assert.True(exist)
	_, exist = gs.subscriptions[gs.formatTopic(testTopic2)]
	assert.True(exist)

	_, exist = gs.topics[gs.formatTopic(testTopic3)]
	assert.True(exist)
	_, exist = gs.subscriptions[gs.formatTopic(testTopic3)]
	assert.True(exist)
}

func TestGossipSub_RegisterEventHandler(t *testing.T) {
	assert := assert.New(t)

	testHandler := func(event *Event) {
	}
	testValidator := func(context.Context, *Message) ValidationResult {
		return ValidationAccept
	}

	gs := newGossipSub(testChainID, testVersion)
	err := gs.RegisterEventHandler(testEvent, testHandler, testValidator)
	assert.Nil(err)

	_, exist := gs.topics[gs.formatTopic(testEvent)]
	assert.True(exist)
	_, exist = gs.subscriptions[gs.formatTopic(testEvent)]
	assert.True(exist)

	assert.NotNil(gs.eventHandlers[gs.formatTopic(testEvent)])

	f1 := *(*unsafe.Pointer)(unsafe.Pointer(&testHandler))
	handler := gs.eventHandlers[gs.formatTopic(testEvent)]
	f2 := *(*unsafe.Pointer)(unsafe.Pointer(&handler))
	assert.True(f1 == f2)
}

func TestGossipSub_RegisterEventHandlerGossipSubRunning(t *testing.T) {
	assert := assert.New(t)

	testHandler := func(event *Event) {
	}
	testValidator := func(context.Context, *Message) ValidationResult {
		return ValidationAccept
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}
	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{}
	_ = cfg.insertDefault()
	p, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	sk := newScoreKeeper()

	gs := newGossipSub(testChainID, testVersion)
	gs.start(ctx, wg, logger, p, sk, cfg)

	err := gs.RegisterEventHandler(testEvent, testHandler, testValidator)
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
	testValidator := func(context.Context, *Message) ValidationResult {
		return ValidationAccept
	}

	gs := newGossipSub(testChainID, testVersion)

	err := gs.RegisterEventHandler(testEvent, testHandler, testValidator)
	assert.Nil(err)
	_, exist := gs.topics[gs.formatTopic(testEvent)]
	assert.True(exist)
	_, exist = gs.subscriptions[gs.formatTopic(testEvent)]
	assert.True(exist)
	_, exist = gs.eventHandlers[gs.formatTopic(testEvent)]
	assert.True(exist)

	err = gs.RegisterEventHandler(testEvent, testHandler, testValidator)
	assert.NotNil(err)
	assert.Equal(err, ErrDuplicateHandler)
}

func TestGossipSub_Publish(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}
	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{}
	_ = cfg.insertDefault()
	p, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	sk := newScoreKeeper()

	gs := newGossipSub(testChainID, testVersion)
	gs.RegisterEventHandler(testTopic1, func(event *Event) {}, testMV)

	gs.start(ctx, wg, logger, p, sk, cfg)

	err := gs.Publish(ctx, testTopic1, testMessageData)
	assert.Nil(err)

	err = gs.Publish(ctx, testTopic1, testMessageInvalid)
	assert.EqualError(err, pubsub.RejectValidationFailed)
	// check the validation nofify once
	err = gs.Publish(ctx, testTopic1, testMessageInvalid)
	assert.Nil(err)
}

func TestGossipSub_PublishTopicNotFound(t *testing.T) {
	assert := assert.New(t)

	gs := newGossipSub(testChainID, testVersion)

	err := gs.Publish(context.Background(), testTopic1, testMessageData)
	assert.Equal(err, ErrTopicNotFound)
}

func TestGossipSub_BelowMinimumNumberOfConnections(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}
	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{}
	_ = cfg.insertDefault()
	p, _ := newPeer(ctx, wg, logger, []byte{}, cfg)
	sk := newScoreKeeper()

	gs := newGossipSub(testChainID, testVersion)
	gs.RegisterEventHandler(testTopic1, func(event *Event) {}, testMV)

	gs.start(ctx, wg, logger, p, sk, cfg)

	// Create two new peers
	cfg1 := Config{
		AllowIncomingConnections: true,
		Addresses:                []string{testIPv4TCP, testIPv4UDP},
	}
	_ = cfg1.insertDefault()
	p1, _ := newPeer(ctx, wg, logger, []byte{}, &cfg1)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, &cfg1)
	p1Addrs, _ := p1.MultiAddress()
	p1AddrInfo, _ := AddrInfoFromMultiAddr(p1Addrs[0])
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

	// Connect to the peers
	err := gs.peer.Connect(ctx, *p1AddrInfo)
	assert.Nil(err)
	err = gs.peer.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)

	// Check if the number of connected peers is the same as the one we have connected to
	assert.Equal(2, len(gs.peer.ConnectedPeers()))

	// And disconnect from both of them
	err = gs.peer.Disconnect(p1.ID())
	assert.Nil(err)
	err = gs.peer.Disconnect(p2.ID())
	assert.Nil(err)

	// Check if the number of connected peers is zero as we have disconnected from all peers
	assert.Equal(0, len(gs.peer.ConnectedPeers()))

	// From now on, gossipsub should try to connect to the peers we have in the known peers list
	for {
		if len(gs.peer.ConnectedPeers()) == 2 {
			break
		}

		select {
		case <-time.After(testTimeout):
			t.Fatalf("timeout occurs, unable to connect to all wanted peers")
		case <-time.After(time.Millisecond * 100):
			// check if we have connected to all peers
		}
	}

	// Check if the number of connected peers is two as we should have been connected to two peers
	assert.Equal(2, len(gs.peer.ConnectedPeers()))
}

package p2p

import (
	"context"
	"sync"
	"testing"
	"unsafe"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/stretchr/testify/assert"

	cfg "github.com/LiskHQ/lisk-engine/pkg/engine/config"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	ps "github.com/LiskHQ/lisk-engine/pkg/p2p/pubsub"
)

var (
	testMessageData    = []byte("testMessageData")
	testMessageInvalid = []byte("testMessageInvalid")
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
	cfgNet := cfg.NetworkConfig{}
	_ = cfgNet.InsertDefault()
	p, _ := NewPeer(ctx, wg, logger, []byte{}, cfgNet)
	sk := ps.NewScoreKeeper()

	gs := NewGossipSub()
	err := gs.RegisterEventHandler(testTopic1, func(event *Event) {})
	assert.Nil(err)

	err = gs.Start(ctx, wg, logger, p, sk, cfgNet)
	assert.Nil(err)

	assert.NotNil(gs.logger)
	assert.NotNil(gs.peer)
	assert.NotNil(gs.ps)

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
	cfgNet := cfg.NetworkConfig{}
	_ = cfgNet.InsertDefault()
	p, _ := NewPeer(ctx, wg, logger, []byte{}, cfgNet)
	sk := ps.NewScoreKeeper()

	gs := NewGossipSub()
	gs.Start(ctx, wg, logger, p, sk, cfgNet)

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
	cfgNet := cfg.NetworkConfig{}
	_ = cfgNet.InsertDefault()
	p, _ := NewPeer(ctx, wg, logger, []byte{}, cfgNet)
	sk := ps.NewScoreKeeper()

	gs := NewGossipSub()
	gs.RegisterEventHandler(testTopic1, func(event *Event) {})

	err := gs.RegisterTopicValidator(testTopic1, testMV)
	assert.Equal(err, ErrGossipSubIsNotRunnig)

	gs.Start(ctx, wg, logger, p, sk, cfgNet)
	assert.Nil(gs.RegisterTopicValidator(testTopic1, testMV))

	err = gs.Publish(ctx, testTopic1, testMessageData)
	assert.Nil(err)

	err = gs.Publish(ctx, testTopic1, testMessageInvalid)
	assert.EqualError(err, pubsub.RejectValidationFailed)
	// check the validation nofify once
	err = gs.Publish(ctx, testTopic1, testMessageInvalid)
	assert.Nil(err)
}

func TestGossipSub_PublishTopicNotFound(t *testing.T) {
	assert := assert.New(t)

	gs := NewGossipSub()

	err := gs.Publish(context.Background(), testTopic1, testMessageData)
	assert.Equal(err, ErrTopicNotFound)
}

package p2p

import (
	"context"
	"sync"
	"testing"
	"unsafe"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/stretchr/testify/assert"

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

func TestGossipSub_StartGossipSub(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()
	wg := &sync.WaitGroup{}
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{}
	_ = config.InsertDefault()
	p, _ := NewPeer(context.Background(), logger, config)
	sk := ps.NewScoreKeeper()

	gs := NewGossipSub()
	err := gs.RegisterEventHandler("testTopic", func(event *Event) {})
	assert.Nil(err)

	err = gs.StartGossipSub(ctx, wg, logger, p, sk, config)
	assert.Nil(err)

	assert.NotNil(gs.logger)
	assert.Equal(p.GetHost().ID(), gs.peerID)
	assert.NotNil(gs.PubSub)

	assert.Equal(1, len(gs.topics))
	assert.Equal(1, len(gs.subscriptions))
	_, exist := gs.topics["testTopic"]
	assert.True(exist)
	_, exist = gs.subscriptions["testTopic"]
	assert.True(exist)
}

func TestGossipSub_CreateSubscriptionHandlers(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()
	wg := &sync.WaitGroup{}

	gs := NewGossipSub()

	gs.RegisterEventHandler("testTopic1", func(event *Event) {})
	gs.RegisterEventHandler("testTopic2", func(event *Event) {})
	gs.RegisterEventHandler("testTopic3", func(event *Event) {})

	host, _ := libp2p.New()
	gossipSub, _ := pubsub.NewGossipSub(ctx, host)
	gs.PubSub = gossipSub

	err := gs.createSubscriptionHandlers(ctx, wg)
	assert.Nil(err)

	assert.Equal(3, len(gs.topics))
	assert.Equal(3, len(gs.subscriptions))

	_, exist := gs.topics["testTopic1"]
	assert.True(exist)
	_, exist = gs.subscriptions["testTopic1"]
	assert.True(exist)

	_, exist = gs.topics["testTopic2"]
	assert.True(exist)
	_, exist = gs.subscriptions["testTopic2"]
	assert.True(exist)

	_, exist = gs.topics["testTopic3"]
	assert.True(exist)
	_, exist = gs.subscriptions["testTopic3"]
	assert.True(exist)
}

func TestGossipSub_RegisterEventHandler(t *testing.T) {
	assert := assert.New(t)

	testHandler := func(event *Event) {
	}

	gs := NewGossipSub()
	err := gs.RegisterEventHandler("testEvent", testHandler)
	assert.Nil(err)

	_, exist := gs.topics["testEvent"]
	assert.True(exist)
	_, exist = gs.subscriptions["testEvent"]
	assert.True(exist)

	assert.NotNil(gs.eventHandlers["testEvent"])

	f1 := *(*unsafe.Pointer)(unsafe.Pointer(&testHandler))
	handler := gs.eventHandlers["testEvent"]
	f2 := *(*unsafe.Pointer)(unsafe.Pointer(&handler))
	assert.True(f1 == f2)
}

func TestGossipSub_RegisterEventHandlerGossipSubRunning(t *testing.T) {
	assert := assert.New(t)

	testHandler := func(event *Event) {
	}

	ctx := context.Background()
	wg := &sync.WaitGroup{}
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{}
	_ = config.InsertDefault()
	p, _ := NewPeer(context.Background(), logger, config)
	sk := ps.NewScoreKeeper()

	gs := NewGossipSub()
	gs.StartGossipSub(ctx, wg, logger, p, sk, config)

	err := gs.RegisterEventHandler("testEvent", testHandler)
	assert.NotNil(err)
	assert.Equal("cannot register event handler after GossipSub is started", err.Error())

	_, exist := gs.topics["testEvent"]
	assert.False(exist)
	_, exist = gs.subscriptions["testEvent"]
	assert.False(exist)
	_, exist = gs.eventHandlers["testEvent"]
	assert.False(exist)
}

func TestGossipSub_RegisterEventHandlerAlreadyRegistered(t *testing.T) {
	assert := assert.New(t)

	testHandler := func(event *Event) {
	}

	gs := NewGossipSub()

	err := gs.RegisterEventHandler("testEvent", testHandler)
	assert.Nil(err)
	_, exist := gs.subscriptions["testEvent"]
	assert.True(exist)
	_, exist = gs.eventHandlers["testEvent"]
	assert.True(exist)
	_, exist = gs.eventHandlers["testEvent"]
	assert.True(exist)

	err = gs.RegisterEventHandler("testEvent", testHandler)
	assert.NotNil(err)
	assert.Equal("eventHandler testEvent is already registered", err.Error())
}

func TestGossipSub_Publish(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()
	wg := &sync.WaitGroup{}
	logger, _ := log.NewDefaultProductionLogger()
	config := Config{}
	_ = config.InsertDefault()
	p, _ := NewPeer(context.Background(), logger, config)
	sk := ps.NewScoreKeeper()

	gs := NewGossipSub()
	gs.RegisterEventHandler("testTopic", func(event *Event) {})
	gs.StartGossipSub(ctx, wg, logger, p, sk, config)

	err := gs.Publish(ctx, "testTopic", []byte("testMessageData"))
	assert.Nil(err)
}

func TestGossipSub_PublishTopicNotFound(t *testing.T) {
	assert := assert.New(t)

	ctx := context.Background()
	gs := NewGossipSub()

	err := gs.Publish(ctx, "testTopic", []byte("testMessageData"))
	assert.Equal("topic not found", err.Error())
}

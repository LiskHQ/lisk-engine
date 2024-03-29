package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/p2p"

	"github.com/testground/sdk-go/runtime"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type HonestNodeConfig struct {
	// topics to join when node starts
	Topics []TopicConfig

	// whether we're a publisher or a lurker
	Publisher bool

	// pubsub event tracer
	Tracer pubsub.EventTracer

	// Test instance identifier
	Seq int64

	// How long to wait after connecting to bootstrap peers before publishing
	Warmup time.Duration

	// How long to wait for cooldown
	Cooldown time.Duration

	// Gossipsub heartbeat params
	Heartbeat HeartbeatParams

	// whether to flood the network when publishing our own messages.
	// Ignored unless hardening_api build tag is present.
	FloodPublishing bool

	// Params for peer scoring function. Ignored unless hardening_api build tag is present.
	PeerScoreParams ScoreParams

	OverlayParams OverlayParams

	// Params for inspecting the scoring values.
	PeerScoreInspect InspectParams

	// Size of the pubsub validation queue.
	ValidateQueueSize int

	// Size of the pubsub outbound queue.
	OutboundQueueSize int

	// Heartbeat tics for opportunistic grafting
	OpportunisticGraftTicks int
}

type InspectParams struct {
	// The callback function that is called with the peer scores
	Inspect func(map[p2p.PeerID]float64)
	// The interval between calling Inspect (defaults to zero: dont inspect).
	Period time.Duration
}

type topicState struct {
	cfg       TopicConfig
	nMessages int64
	pubTicker *time.Ticker
	done      chan struct{}
}

type PubsubNode struct {
	cfg    HonestNodeConfig
	ctx    context.Context
	runenv *runtime.RunEnv
	conn   p2p.ExtendedConnection
	mutex  sync.RWMutex
	pubwg  sync.WaitGroup
}

// NewPubsubNode prepares the given Host to act as an honest pubsub node using the provided HonestNodeConfig.
// The returned PubsubNode will not start immediately; call Run to begin the test behavior.
func NewPubsubNode(runenv *runtime.RunEnv, ctx context.Context, conn p2p.ExtendedConnection, cfg HonestNodeConfig) (*PubsubNode, error) {
	opts, err := pubsubOptions(cfg)
	if err != nil {
		return nil, err
	}

	// Set the heartbeat initial delay and interval
	pubsub.GossipSubHeartbeatInitialDelay = cfg.Heartbeat.InitialDelay
	pubsub.GossipSubHeartbeatInterval = cfg.Heartbeat.Interval

	// call RegisterEventHandler per topics
	err = conn.StartGossipSub(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("error making new gossipsub: %s", err)
	}

	p := PubsubNode{
		cfg:    cfg,
		ctx:    ctx,
		runenv: runenv,
		conn:   conn,
	}

	return &p, nil
}

func (p *PubsubNode) log(msg string, args ...interface{}) {
	id := p.conn.ID().Pretty()
	idSuffix := id[len(id)-8:]
	prefix := fmt.Sprintf("[honest %d %s] ", p.cfg.Seq, idSuffix)
	p.runenv.RecordMessage(prefix+msg, args...)
}

func (p *PubsubNode) Run(runtime time.Duration, waitForReadyStateThenConnectAsync func(context.Context) error) error {
	// Wait for all nodes to be in the ready state (including attack nodes)
	// then start connecting (asynchronously)
	if err := waitForReadyStateThenConnectAsync(p.ctx); err != nil {
		return err
	}

	// join initial topics
	p.runenv.RecordMessage("Joining initial topics")
	for _, t := range p.cfg.Topics {
		go p.joinTopic(t, runtime)
	}

	// wait for warmup time to expire
	p.runenv.RecordMessage("Wait for %s warmup time", p.cfg.Warmup)
	select {
	case <-time.After(p.cfg.Warmup):
	case <-p.ctx.Done():
		return p.ctx.Err()
	}

	// ensure we have at least enough peers to fill a mesh after warmup period
	npeers := p.conn.ConnectedPeers().Len()
	if npeers < pubsub.GossipSubD {
		panic(fmt.Errorf("not enough peers after warmup period. Need at least D=%d, have %d", pubsub.GossipSubD, npeers))
	}

	// block until complete
	p.runenv.RecordMessage("Wait for %s run time", runtime)
	select {
	case <-time.After(runtime):
	case <-p.ctx.Done():
		return p.ctx.Err()
	}

	// if we're publishing, wait until we've sent all our messages or the context expires
	if p.cfg.Publisher {
		donech := make(chan struct{}, 1)
		go func() {
			p.pubwg.Wait()
			donech <- struct{}{}
		}()

		select {
		case <-donech:
		case <-p.ctx.Done():
			return p.ctx.Err()
		}
	}

	p.runenv.RecordMessage("Run time complete, cooling down for %s", p.cfg.Cooldown)
	select {
	case <-time.After(p.cfg.Cooldown):
	case <-p.ctx.Done():
		return p.ctx.Err()
	}

	p.runenv.RecordMessage("Cool down complete")

	return nil
}

func (p *PubsubNode) joinTopic(t TopicConfig, runtime time.Duration) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	publishInterval := time.Duration(float64(t.MessageRate.Interval) / t.MessageRate.Quantity)
	totalMessages := int64(runtime / publishInterval)

	if p.cfg.Publisher {
		p.log("publishing to topic %s. message_rate: %.2f/%ds, publishInterval %dms, msg size %d bytes. total expected messages: %d",
			t.Id, t.MessageRate.Quantity, t.MessageRate.Interval/time.Second, publishInterval/time.Millisecond, t.MessageSize, totalMessages)
	} else {
		p.log("joining topic %s as a lurker", t.Id)
	}

	if !p.cfg.Publisher {
		return
	}

	go func() {
		p.runenv.RecordMessage("Wait for %s warmup time before starting publisher", p.cfg.Warmup)
		select {
		case <-time.After(p.cfg.Warmup):
		case <-p.ctx.Done():
			p.runenv.RecordMessage("Context done before warm up time in publisher: %s", p.ctx.Err())
			return
		}

		p.runenv.RecordMessage("Starting publisher with %s publish interval", publishInterval)
		ts := topicState{
			cfg:       t,
			nMessages: totalMessages,
			done:      make(chan struct{}, 1),
		}
		ts.pubTicker = time.NewTicker(publishInterval)
		p.publishLoop(&ts)
	}()
}

func (p *PubsubNode) makeMessage(seq int64, size uint64) ([]byte, error) {
	type msg struct {
		sender string
		seq    int64
		data   []byte
	}
	data := make([]byte, size)
	rand.Read(data)
	m := msg{sender: p.conn.ID().Pretty(), seq: seq, data: data}
	return json.Marshal(m)
}

func (p *PubsubNode) sendMsg(seq int64, ts *topicState) {
	msg, err := p.makeMessage(seq, uint64(ts.cfg.MessageSize))
	if err != nil {
		p.log("error making message for topic %s: %s", ts.cfg.Id, err)
		return
	}
	err = p.conn.Publish(p.ctx, ts.cfg.Id, msg)
	if err != nil && err != context.Canceled {
		p.log("error publishing to %s: %s", ts.cfg.Id, err)
		return
	}
}

func (p *PubsubNode) publishLoop(ts *topicState) {
	var counter int64
	p.pubwg.Add(1)
	defer p.pubwg.Done()
	for {
		select {
		case <-ts.done:
			return
		case <-p.ctx.Done():
			return
		case <-ts.pubTicker.C:
			go p.sendMsg(counter, ts)
			counter++
			if counter > ts.nMessages {
				ts.pubTicker.Stop()
				return
			}
		}
	}
}

package p2p

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	lps "github.com/LiskHQ/lisk-engine/pkg/p2p/pubsub"
)

const maxAllowedTopics = 10

var (
	ErrGossipSubIsNotRunnig = errors.New("gossipSub is not running")
	ErrGossipSubIsRunning   = errors.New("gossipSub is running the action is not possible")
	ErrDuplicateHandler     = errors.New("eventHandler is already registered")
	ErrTopicNotFound        = errors.New("topic not found")
)

// GossipSub type.
type GossipSub struct {
	logger        log.Logger
	peer          *Peer
	ps            *pubsub.PubSub
	topics        map[string]*pubsub.Topic
	subscriptions map[string]*pubsub.Subscription
	eventHandlers map[string]EventHandler
}

// NewGossipSub makes a new GossipSub struct.
func NewGossipSub() *GossipSub {
	return &GossipSub{
		topics:        make(map[string]*pubsub.Topic),
		subscriptions: make(map[string]*pubsub.Subscription),
		eventHandlers: make(map[string]EventHandler),
	}
}

// Start starts a GossipSub based on input parameters.
func (gs *GossipSub) Start(ctx context.Context,
	wg *sync.WaitGroup,
	logger log.Logger,
	p *Peer,
	sk *lps.ScoreKeeper,
	cfg Config,
) error {
	seedNodes, err := lps.ParseAddresses(cfg.SeedPeers)
	if err != nil {
		return err
	}
	fixedNodes, err := lps.ParseAddresses(cfg.FixedPeers)
	if err != nil {
		return err
	}

	bootstrappers := make(map[peer.ID]struct{})
	for _, info := range seedNodes {
		bootstrappers[info.ID] = struct{}{}
	}

	topicParams := make(map[string]*pubsub.TopicScoreParams)
	topics := make([]string, 0, maxAllowedTopics)
	for topic := range gs.topics {
		topicParams[topic] = &pubsub.TopicScoreParams{
			TopicWeight:                    0.1,
			TimeInMeshWeight:               0.0002778,
			TimeInMeshQuantum:              time.Second,
			TimeInMeshCap:                  1,
			FirstMessageDeliveriesWeight:   0.5,
			FirstMessageDeliveriesDecay:    pubsub.ScoreParameterDecay(10 * time.Minute),
			FirstMessageDeliveriesCap:      100,
			InvalidMessageDeliveriesWeight: -1000,
			InvalidMessageDeliveriesDecay:  pubsub.ScoreParameterDecay(time.Hour),
		}
		topics = append(topics, topic)
	}

	var ipWhitelist []*net.IPNet
	options := []pubsub.Option{
		pubsub.WithFloodPublish(true),
		pubsub.WithMessageIdFn(lps.HashMsgID),
		pubsub.WithPeerScore(
			&pubsub.PeerScoreParams{
				AppSpecificScore: func(p peer.ID) float64 {
					_, ok := bootstrappers[p]
					if ok && !cfg.IsSeedNode {
						return 2500
					}
					return 0
				},
				AppSpecificWeight:           1,
				IPColocationFactorThreshold: 5,
				IPColocationFactorWeight:    -100,
				IPColocationFactorWhitelist: ipWhitelist,
				BehaviourPenaltyThreshold:   6,
				BehaviourPenaltyWeight:      -10,
				BehaviourPenaltyDecay:       pubsub.ScoreParameterDecay(time.Hour),
				DecayInterval:               pubsub.DefaultDecayInterval,
				DecayToZero:                 pubsub.DefaultDecayToZero,
				RetainScore:                 6 * time.Hour,
				Topics:                      topicParams,
			},
			&pubsub.PeerScoreThresholds{
				GossipThreshold:             -500,
				PublishThreshold:            -1000,
				GraylistThreshold:           -2500,
				AcceptPXThreshold:           1000,
				OpportunisticGraftThreshold: 3.5,
			},
		),
		pubsub.WithPeerScoreInspect(sk.Update, 10*time.Second),
	}

	if cfg.IsSeedNode {
		pubsub.GossipSubD = 0
		pubsub.GossipSubDscore = 0
		pubsub.GossipSubDlo = 0
		pubsub.GossipSubDhi = 0
		pubsub.GossipSubDout = 0
		pubsub.GossipSubDlazy = 64
		pubsub.GossipSubGossipFactor = 0.25
		pubsub.GossipSubPruneBackoff = 5 * time.Minute
	}

	pgTopicWeights := map[string]float64{}

	var pgParams *pubsub.PeerGaterParams
	if cfg.IsSeedNode {
		pgParams = pubsub.NewPeerGaterParams(
			0.33,
			pubsub.ScoreParameterDecay(2*time.Minute),
			pubsub.ScoreParameterDecay(10*time.Minute),
		).WithTopicDeliveryWeights(pgTopicWeights)
	} else {
		pgParams = pubsub.NewPeerGaterParams(
			0.33,
			pubsub.ScoreParameterDecay(2*time.Minute),
			pubsub.ScoreParameterDecay(time.Hour),
		).WithTopicDeliveryWeights(pgTopicWeights)
	}
	options = append(options, pubsub.WithPeerGater(pgParams))

	options = append(options,
		pubsub.WithDirectPeers(fixedNodes),
		pubsub.WithSubscriptionFilter(
			pubsub.WrapLimitSubscriptionFilter(
				pubsub.NewAllowlistSubscriptionFilter(topics...),
				maxAllowedTopics)))

	// We want to hide the author of the message from the topic subscribers.
	options = append(options, pubsub.WithNoAuthor())

	// We want to enable peer exchange for all peers and not only for seed peers.
	options = append(options, pubsub.WithPeerExchange(true))

	gossipSub, err := pubsub.NewGossipSub(ctx, p.GetHost(), options...)
	if err != nil {
		return err
	}

	gs.logger = logger
	gs.peer = p
	gs.ps = gossipSub

	err = gs.createSubscriptionHandlers(ctx, wg)
	if err != nil {
		return err
	}

	return nil
}

// createSubscriptionHandlers creates a subscription handler for each topic.
func (gs *GossipSub) createSubscriptionHandlers(ctx context.Context, wg *sync.WaitGroup) error {
	if gs.ps == nil {
		return ErrGossipSubIsNotRunnig
	}

	// Join all topics and create a subscription for each of them.
	for t := range gs.topics {
		topic, err := gs.ps.Join(t)
		if err != nil {
			return err
		}
		sub, err := topic.Subscribe()
		if err != nil {
			return err
		}
		gs.topics[t] = topic
		gs.subscriptions[t] = sub
	}

	// Start a goroutine for each subscription.
	for _, sub := range gs.subscriptions {
		wg.Add(1)
		go func(sub *pubsub.Subscription) {
			defer wg.Done()
			for {
				msg, err := sub.Next(ctx)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						gs.logger.Infof("Topic \"%s\" subscription handler stopped", sub.Topic())
						return
					}
					gs.logger.Errorf("Error while receiving message: %s", err)
					continue
				}
				// Only process messages delivered by others.
				if msg.ReceivedFrom == gs.peer.GetHost().ID() {
					continue
				}
				gs.logger.Debugf("Received message: %s", msg.Data)

				m := newMessage(nil)
				err = m.Decode(msg.Data)
				if err != nil {
					gs.logger.Errorf("Error while decoding message: %s", err)
					continue
				}

				handler, ok := gs.eventHandlers[sub.Topic()]
				if !ok {
					gs.logger.Errorf("EventHandler for %s not found", sub.Topic())
					continue
				}
				event := newEvent(msg.ReceivedFrom.String(), sub.Topic(), m.Data)
				handler(event)
			}
		}(sub)
	}

	return nil
}

// RegisterValidator registers a validator for given topic.
func (gs *GossipSub) RegisterTopicValidator(topic string, v Validator) error {
	if gs.ps == nil {
		return ErrGossipSubIsNotRunnig
	}

	return gs.ps.RegisterTopicValidator(topic, newMessageValidator(v))
}

// RegisterEventHandler registers an event handler for an event type.
func (gs *GossipSub) RegisterEventHandler(name string, handler EventHandler) error {
	if gs.ps != nil {
		return ErrGossipSubIsRunning
	}
	_, exist := gs.eventHandlers[name]
	if exist {
		return ErrDuplicateHandler
	}
	gs.topics[name] = nil
	gs.subscriptions[name] = nil
	gs.eventHandlers[name] = handler
	return nil
}

// Publish publishes a message to a topic.
func (gs *GossipSub) Publish(ctx context.Context, topicName string, msg *Message) error {
	data, err := msg.Encode()
	if err != nil {
		return err
	}
	topic := gs.topics[topicName]
	if topic == nil {
		return ErrTopicNotFound
	}
	return topic.Publish(ctx, data)
}

// Start starts the GossipSub event handler.
func gossipSubEventHandler(ctx context.Context, wg *sync.WaitGroup, p *Peer, gs *GossipSub) {
	defer wg.Done()
	gs.logger.Infof("GossipSub event handler started")

	t := time.NewTicker(10 * time.Second)
	var counter = 0

	for {
		select {
		// TODO - remove this timer event after testing (GH issue #19)
		case <-t.C:
			gs.logger.Debugf("GossipSub event handler is alive")
			topicTransactions := "transactions" // Test topic which will be removed after testing
			topicBlocks := "blocks"             // Test topic which will be removed after testing
			topicEvents := "events"             // Test topic which will be removed after testing
			txMsg := newMessage([]byte(fmt.Sprintf("Timer for %s is running and this is a test transaction message: %v", p.ID().String(), counter)))
			err := gs.Publish(ctx, topicTransactions, txMsg)
			if err != nil {
				gs.logger.Errorf("Error while publishing message: %s", err)
			}
			blkMsg := newMessage([]byte(fmt.Sprintf("Timer for %s is running and this is a test block message: %v", p.ID().String(), counter)))
			err = gs.Publish(ctx, topicBlocks, blkMsg)
			if err != nil {
				gs.logger.Errorf("Error while publishing message: %s", err)
			}
			entMsg := newMessage([]byte(fmt.Sprintf("Timer for %s is running and this is a test event message: %v", p.ID().String(), counter)))
			err = gs.Publish(ctx, topicEvents, entMsg)
			if err != nil {
				gs.logger.Errorf("Error while publishing message: %s", err)
			}
			counter++

			p.logger.Debugf("List of connected peers: %v", p.ConnectedPeers())
			p.logger.Debugf("List of known peers: %v", p.knownPeers())
			p.logger.Debugf("List of blacklisted peers: %v", gs.peer.BlacklistedPeers())

			t.Reset(10 * time.Second)
		case <-ctx.Done():
			gs.logger.Infof("GossipSub event handler stopped")
			return
		}
	}
}

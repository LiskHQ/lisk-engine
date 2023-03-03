package p2p

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/engine/config"
	"github.com/LiskHQ/lisk-engine/pkg/log"
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
	logger            log.Logger
	peer              *Peer
	ps                *pubsub.PubSub
	topics            map[string]*pubsub.Topic
	subscriptions     map[string]*pubsub.Subscription
	eventHandlers     map[string]EventHandler
	validatorHandlers map[string]Validator
	chainID           []byte
	version           string
}

// newGossipSub makes a new GossipSub struct.
func newGossipSub(chainID []byte, version string) *GossipSub {
	return &GossipSub{
		topics:            make(map[string]*pubsub.Topic),
		subscriptions:     make(map[string]*pubsub.Subscription),
		eventHandlers:     make(map[string]EventHandler),
		validatorHandlers: make(map[string]Validator),
		chainID:           chainID,
		version:           version,
	}
}

// getMessageID compute the ID for the gossipsub message.
func getMessageID(m *pubsub_pb.Message) string {
	hash := crypto.Hash(m.Data)
	return codec.Hex(hash).String()
}

// start starts a GossipSub based on input parameters.
func (gs *GossipSub) start(ctx context.Context,
	wg *sync.WaitGroup,
	logger log.Logger,
	p *Peer,
	sk *scoreKeeper,
	cfgNet config.NetworkConfig,
) error {
	seedNodes, err := parseAddresses(cfgNet.SeedPeers)
	if err != nil {
		return err
	}
	fixedNodes, err := parseAddresses(cfgNet.FixedPeers)
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
		pubsub.WithMessageIdFn(getMessageID),
		pubsub.WithPeerScore(
			&pubsub.PeerScoreParams{
				AppSpecificScore: func(p peer.ID) float64 {
					_, ok := bootstrappers[p]
					if ok && !cfgNet.IsSeedPeer {
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

	if cfgNet.IsSeedPeer {
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
	if cfgNet.IsSeedPeer {
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

	gossipSub, err := pubsub.NewGossipSub(ctx, p.host, options...)
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

		validator, exist := gs.validatorHandlers[t]
		if exist && validator != nil {
			if err := gs.ps.RegisterTopicValidator(t, newMessageValidator(validator)); err != nil {
				return err
			}
		}
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
				if msg.ReceivedFrom == gs.peer.host.ID() {
					continue
				}
				gs.logger.Debugf("Received message: %s", msg.Data)

				m := NewMessage(nil)
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
				event := NewEvent(msg.ReceivedFrom.String(), sub.Topic(), m.Data)
				handler(event)
			}
		}(sub)
	}

	return nil
}

// RegisterEventHandler registers an event handler for an event type.
func (gs *GossipSub) RegisterEventHandler(name string, handler EventHandler, validator Validator) error {
	if gs.ps != nil {
		return ErrGossipSubIsRunning
	}
	formattedTopic := gs.formatTopic(name)
	_, exist := gs.eventHandlers[formattedTopic]
	if exist {
		return ErrDuplicateHandler
	}
	gs.topics[formattedTopic] = nil
	gs.subscriptions[formattedTopic] = nil
	gs.eventHandlers[formattedTopic] = handler
	gs.validatorHandlers[formattedTopic] = validator
	return nil
}

// Publish publishes a message to a topic.
func (gs *GossipSub) Publish(ctx context.Context, topicName string, data []byte) error {
	msg := NewMessage(data)

	data, err := msg.Encode()
	if err != nil {
		return err
	}
	topic := gs.topics[gs.formatTopic(topicName)]
	if topic == nil {
		return ErrTopicNotFound
	}
	return topic.Publish(ctx, data)
}

func (gs *GossipSub) formatTopic(name string) string {
	return fmt.Sprintf("/%s/%s/%s", codec.Hex(gs.chainID).String(), gs.version, name)
}

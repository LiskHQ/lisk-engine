package p2p

import (
	"context"
	"net"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// GossipSub is wrapper of PubSub.
type GossipSub struct {
	ps *pubsub.PubSub
}

// NewGossipSub makes a new GossipSub based on input parameters.
func NewGossipSub(ctx context.Context,
	h host.Host,
	sk *ScoreKeeper,
	cfg Config,
) (*GossipSub, error) {
	bootNodes, err := ParseAddresses(ctx, cfg.BootNode)
	if err != nil {
		return nil, err
	}

	bootstrappers := make(map[peer.ID]struct{})
	for _, info := range bootNodes {
		bootstrappers[info.ID] = struct{}{}
	}

	msgTopic := MessageTopic(cfg.NetworkName)
	topicParams := map[string]*pubsub.TopicScoreParams{
		msgTopic: {
			TopicWeight:                    0.1,
			TimeInMeshWeight:               0.0002778,
			TimeInMeshQuantum:              time.Second,
			TimeInMeshCap:                  1,
			FirstMessageDeliveriesWeight:   0.5,
			FirstMessageDeliveriesDecay:    pubsub.ScoreParameterDecay(10 * time.Minute),
			FirstMessageDeliveriesCap:      100,
			InvalidMessageDeliveriesWeight: -1000,
			InvalidMessageDeliveriesDecay:  pubsub.ScoreParameterDecay(time.Hour),
		},
	}

	var ipWhitelist []*net.IPNet
	options := []pubsub.Option{
		pubsub.WithFloodPublish(true),
		pubsub.WithMessageIdFn(hashMsgID),
		pubsub.WithPeerScore(
			&pubsub.PeerScoreParams{
				AppSpecificScore: func(p peer.ID) float64 {
					_, ok := bootstrappers[p]
					if ok && !cfg.IsBootstrapNode {
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

	if cfg.IsBootstrapNode {
		pubsub.GossipSubD = 0
		pubsub.GossipSubDscore = 0
		pubsub.GossipSubDlo = 0
		pubsub.GossipSubDhi = 0
		pubsub.GossipSubDout = 0
		pubsub.GossipSubDlazy = 64
		pubsub.GossipSubGossipFactor = 0.25
		pubsub.GossipSubPruneBackoff = 5 * time.Minute
		options = append(options, pubsub.WithPeerExchange(true))
	}

	pgTopicWeights := map[string]float64{
		msgTopic: 1,
	}
	var pgParams *pubsub.PeerGaterParams
	if cfg.IsBootstrapNode {
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
		pubsub.WithDirectPeers(bootNodes),
		pubsub.WithSubscriptionFilter(
			pubsub.WrapLimitSubscriptionFilter(
				pubsub.NewAllowlistSubscriptionFilter(msgTopic),
				100)))
	ps, err := pubsub.NewGossipSub(ctx, h, options...)
	if err != nil {
		return nil, err
	}

	return &GossipSub{ps}, nil
}

// P2pAddrs returns all listen address of GossipSub based on host.
func P2pAddrs(h host.Host) ([]ma.Multiaddr, error) {
	peerInfo := peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}

	return peer.AddrInfoToP2pAddrs(&peerInfo)
}

// PubSub returns the inner of Pubsub.
func (ps *GossipSub) PubSub() *pubsub.PubSub {
	return ps.ps
}

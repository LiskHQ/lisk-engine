package p2p

import (
	"context"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"

	ma "github.com/multiformats/go-multiaddr"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

// ExtendedConnection extends the Connection for some extra utilities functions.
type ExtendedConnection struct {
	*Connection
}

// NewExtendedConnection returns a new ExtendedConnection.
func NewExtendedConnection(logger log.Logger, cfg *Config) *ExtendedConnection {
	return &ExtendedConnection{
		NewConnection(logger, cfg),
	}
}

// StartGossipSub starts a new gossipsub based on input options.
func (ec *ExtendedConnection) StartGossipSub(ctx context.Context, options ...pubsub.Option) error {
	return ec.startWithOption(ctx, &ec.wg, ec.Peer, ec.cfg, options...)
}

func (gs *GossipSub) startWithOption(ctx context.Context,
	wg *sync.WaitGroup,
	p *Peer,
	cfg *Config,
	options ...pubsub.Option,
) error {
	seedNodes, err := parseAddresses(cfg.SeedPeers)
	if err != nil {
		return err
	}
	fixedNodes, err := parseAddresses(cfg.FixedPeers)
	if err != nil {
		return err
	}

	bootstrappers := make(map[PeerID]struct{})
	for _, info := range seedNodes {
		bootstrappers[info.ID] = struct{}{}
	}

	topics := make([]string, 0, maxAllowedTopics)
	for t := range gs.eventHandlers {
		topics = append(topics, t)
	}
	options = append(options,
		pubsub.WithDirectPeers(fixedNodes),
		pubsub.WithMessageIdFn(getMessageID),
		pubsub.WithSubscriptionFilter(
			pubsub.WrapLimitSubscriptionFilter(
				pubsub.NewAllowlistSubscriptionFilter(topics...),
				maxAllowedTopics)),
	)

	// We want to hide the author of the message from the topic subscribers.
	options = append(options, pubsub.WithNoAuthor())

	// We want to enable peer exchange for all peers and not only for seed peers.
	options = append(options, pubsub.WithPeerExchange(true))

	// We want to provide a custom discovery mechanism.
	d := Discovery{peer: p}
	options = append(options, pubsub.WithDiscovery(d))

	gossipSub, err := pubsub.NewGossipSub(ctx, p.host, options...)
	if err != nil {
		return err
	}

	gs.peer = p
	gs.ps = gossipSub

	err = gs.createSubscriptionHandlers(ctx, wg)
	if err != nil {
		return err
	}

	return nil
}

// SetStreamHandler sets a new handler for given protocolID in the host/node.
func (ec *ExtendedConnection) SetStreamHandler(protocolID protocol.ID, hander network.StreamHandler) {
	ec.Peer.host.SetStreamHandler(protocolID, hander)
}

// NewSteam opens a new stream to given peer in the host/node.
func (ec *ExtendedConnection) NewStream(ctx context.Context, pid PeerID, pids ...protocol.ID) (network.Stream, error) {
	return ec.Peer.host.NewStream(ctx, pid, pids...)
}

// ConnToPeer returns the connections in the Network of the host/node for given peerID.
func (ec *ExtendedConnection) ConnsToPeer(pid PeerID) []network.Conn {
	return ec.Peer.host.Network().ConnsToPeer(pid)
}

// SwarmClear returns true and removes a backoff record if the Network of the host/node has the given peerID.
func (ec *ExtendedConnection) SwarmClear(pid PeerID) bool {
	sw, ok := ec.Peer.host.Network().(*swarm.Swarm)
	if ok {
		sw.Backoff().Clear(pid)
	}

	return ok
}

// Info returns an AddrInfo struct with the ID of the host/node and all of its Addrs.
func (ec *ExtendedConnection) Info() *AddrInfo {
	return host.InfoFromHost(ec.Peer.host)
}

// Listen tells the network of the host/node to start listening on given multiaddrs.
// It will be available with an empty addresses of the config in NewRawConnection.
func (ec *ExtendedConnection) Listen(addrs []ma.Multiaddr) error {
	return ec.Peer.host.Network().Listen(addrs...)
}

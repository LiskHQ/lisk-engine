// This file contains some functions which are useful for gossipsub testplan
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
)

func (conn *Connection) SetStreamHandler(protocolID protocol.ID, hander network.StreamHandler) {
	conn.Peer.host.SetStreamHandler(protocolID, hander)
}

func (conn *Connection) NewStream(ctx context.Context, pid PeerID, pids ...protocol.ID) (network.Stream, error) {
	return conn.Peer.host.NewStream(ctx, pid, pids...)
}

func (conn *Connection) ConnsToPeer(pid PeerID) []network.Conn {
	return conn.Peer.host.Network().ConnsToPeer(pid)
}

func (conn *Connection) SwarmBackoff(pid PeerID) bool {
	sw, ok := conn.Peer.host.Network().(*swarm.Swarm)
	if ok {
		sw.Backoff().Clear(pid)
	}

	return ok
}

func (conn *Connection) Info() *AddrInfo {
	return host.InfoFromHost(conn.Peer.host)
}

func (conn *Connection) Listen(addrs []ma.Multiaddr) error {
	return conn.Peer.host.Network().Listen(addrs...)
}

func (conn *Connection) NewGossipSub(ctx context.Context, options ...pubsub.Option) error {
	return conn.startWithOption(ctx, &conn.wg, conn.Peer, conn.cfg, options...)
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

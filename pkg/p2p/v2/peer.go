package p2p

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	lps "github.com/LiskHQ/lisk-engine/pkg/p2p/v2/pubsub"
)

const numOfPingMessages = 5         // Number of sent ping messages in Ping service.
const pingTimeout = time.Second * 5 // Ping service timeout in seconds.

// Connection security option type.
const (
	ConnectionSecurityNone  = "none"  // Do not support any security.
	ConnectionSecurityTLS   = "tls"   // Support TLS connections.
	ConnectionSecurityNoise = "noise" // Support Noise connections.
)

// Peer type - a p2p node.
type Peer struct {
	logger    log.Logger
	host      host.Host
	peerbook  *Peerbook
	peerScore *peerScore
	connGater *connectionGater
}

var autoRelayOptions = []autorelay.Option{
	autorelay.WithNumRelays(2),
	autorelay.WithMaxCandidates(20),
	autorelay.WithMinCandidates(1),
	autorelay.WithBootDelay(3 * time.Minute),
	autorelay.WithBackoff(1 * time.Hour),
	autorelay.WithMaxCandidateAge(30 * time.Minute),
}

var relayServiceOptions = []relay.Option{
	relay.WithResources(
		relay.Resources{
			Limit:                  &relay.RelayLimit{Duration: 2 * time.Minute, Data: 1 << 17 /*128K*/},
			ReservationTTL:         time.Hour,
			MaxReservations:        128,
			MaxCircuits:            16,
			BufferSize:             2048,
			MaxReservationsPerPeer: 4,
			MaxReservationsPerIP:   8,
			MaxReservationsPerASN:  32},
	),
	relay.WithLimit(&relay.RelayLimit{Duration: 2 * time.Minute, Data: 1 << 17 /*128K*/}),
}

// NewPeer creates a peer with a libp2p host and message protocol.
func NewPeer(ctx context.Context, logger log.Logger, config Config) (*Peer, error) {
	// Create a Peer variable in advance to be able to use it in the libp2p options.
	var p *Peer

	opts := []libp2p.Option{
		// Support default transports (TCP, QUIC, WS)
		libp2p.DefaultTransports,
	}

	switch config.AllowIncomingConnections {
	case true:
		if len(config.Addresses) == 0 {
			opts = append(opts, libp2p.NoListenAddrs)
		} else {
			opts = append(opts, libp2p.ListenAddrStrings(config.Addresses...))
		}
	case false:
		opts = append(opts, libp2p.NoListenAddrs)
	}

	connGater, err := newConnGater(time.Hour*24, time.Second*10)
	if err != nil {
		return nil, err
	}

	// Load Blacklist
	connGaterOpt, err := connGater.optionWithBlacklist(config.BlacklistedIPs)
	if err != nil {
		return nil, err
	}
	opts = append(opts, connGaterOpt)

	// Configure connection security.
	security := strings.ToLower(config.ConnectionSecurity)
	switch security {
	case ConnectionSecurityNone:
		opts = append(opts, libp2p.NoSecurity)
	case ConnectionSecurityTLS:
		opts = append(opts, libp2p.Security(libp2ptls.ID, libp2ptls.New))
	case ConnectionSecurityNoise:
		opts = append(opts, libp2p.Security(noise.ID, noise.New))
	default:
		opts = append(opts, libp2p.NoSecurity)
	}

	// Configure peer to provide a service for other peers for determining their reachability status.
	if config.EnableNATService {
		opts = append(opts, libp2p.EnableNATService())
		opts = append(opts, libp2p.AutoNATServiceRateLimit(60, 10, time.Minute))
	}
	opts = append(opts, libp2p.NATPortMap())

	// Enable using relay service from other peers. In case a peer is not reachable from the network,
	// it will try to connect to a relay service from other peers.
	if config.EnableUsingRelayService {
		opts = append(opts, libp2p.EnableRelay())

		autoRelayOptions = append(autoRelayOptions, autorelay.WithPeerSource(func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
			return p.peerSource(ctx, numPeers)
		}, 1*time.Minute))
		opts = append(opts, libp2p.EnableAutoRelay(autoRelayOptions...))
	} else {
		// Relay is enabled by default, so we need to disable it explicitly.
		opts = append(opts, libp2p.DisableRelay())
	}

	// Enable circuit relay service.
	if config.EnableRelayService {
		opts = append(opts, libp2p.EnableRelayService(relayServiceOptions...))
	}

	// Enable hole punching service.
	if config.EnableHolePunching {
		opts = append(opts, libp2p.EnableHolePunching())
	}

	host, err := libp2p.New(opts...)
	if err != nil {
		return nil, err
	}

	peerbook, err := NewPeerbook(config.SeedPeers, config.FixedPeers, config.BlacklistedIPs, config.KnownPeers)
	if err != nil {
		return nil, err
	}

	p = &Peer{logger: logger, host: host, peerbook: peerbook, peerScore: newPeerScore(), connGater: connGater}
	p.logger.Infof("Peer successfully created")
	p.connGater.start(ctx)
	return p, nil
}

// Close a peer.
func (p *Peer) Close() error {
	err := p.host.Close()
	if err != nil {
		return err
	}
	p.logger.Infof("Peer successfully stopped")
	return nil
}

// Connect to a peer.
func (p *Peer) Connect(ctx context.Context, peer peer.AddrInfo) error {
	for _, addr := range peer.Addrs {
		ip := lps.ExtractIP(addr)
		if p.peerbook.isIPBlacklisted(ip) {
			p.logger.Warningf("IP %s is blacklisted. Will not connect to a peer %s", ip, peer.ID)
			return nil
		}
		if p.peerbook.isIPBanned(ip) {
			p.logger.Warningf("IP %s is banned. Will not connect to a peer %s", ip, peer.ID)
			return nil
		}
	}
	return p.host.Connect(ctx, peer)
}

// Disconnect from a peer.
func (p *Peer) Disconnect(ctx context.Context, peer peer.ID) error {
	return p.host.Network().ClosePeer(peer)
}

// ID returns a peers's identifier.
func (p *Peer) ID() peer.ID {
	return p.host.ID()
}

// P2PAddrs returns a peers's listen addresses in multiaddress format.
func (p *Peer) P2PAddrs() ([]ma.Multiaddr, error) {
	peerInfo := peer.AddrInfo{
		ID:    p.ID(),
		Addrs: p.host.Addrs(),
	}
	return peer.AddrInfoToP2pAddrs(&peerInfo)
}

// ConnectedPeers returns a list of all connected peers.
func (p *Peer) ConnectedPeers() []peer.ID {
	return p.host.Network().Peers()
}

// KnownPeers returns a list of all known peers.
func (p *Peer) KnownPeers() peer.IDSlice {
	return p.host.Peerstore().Peers()
}

// GetHost returns a libp2p host.
func (p *Peer) GetHost() host.Host {
	return p.host
}

// PingMultiTimes tries to send ping request to a peer for five times.
func (p *Peer) PingMultiTimes(ctx context.Context, peer peer.ID) (rtt []time.Duration, err error) {
	pingService := ping.NewPingService(p.host)
	ch := pingService.Ping(ctx, peer)

	p.logger.Debugf("Sending %d ping messages to %v", numOfPingMessages, peer)
	for i := 0; i < numOfPingMessages; i++ {
		select {
		case pingRes := <-ch:
			if pingRes.Error != nil {
				return rtt, pingRes.Error
			}
			p.logger.Debugf("Pinged %v in %v", peer, pingRes.RTT)
			rtt = append(rtt, pingRes.RTT)
		case <-ctx.Done():
			return rtt, errors.New("ping canceled")
		case <-time.After(pingTimeout):
			return rtt, errors.New("ping timeout")
		}
	}
	return rtt, nil
}

// Ping tries to send a ping request to a peer.
func (p *Peer) Ping(ctx context.Context, peer peer.ID) (rtt time.Duration, err error) {
	pingService := ping.NewPingService(p.host)
	ch := pingService.Ping(ctx, peer)

	p.logger.Debugf("Sending ping messages to %v", peer)
	select {
	case pingRes := <-ch:
		if pingRes.Error != nil {
			return rtt, pingRes.Error
		}
		p.logger.Debugf("Pinged %v in %v", peer, pingRes.RTT)
		return pingRes.RTT, nil
	case <-ctx.Done():
		return rtt, errors.New("ping canceled")
	case <-time.After(time.Second * pingTimeout):
		return rtt, errors.New("ping timeout")
	}
}

// peerSource returns a channel and sends connected peers (possible relayers) to that channel.
func (p *Peer) peerSource(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
	peerChan := make(chan peer.AddrInfo, 1)

	go func() {
		defer close(peerChan)

		knownPeers := p.peerbook.KnownPeers()

		// Shuffle known peers to avoid always returning the same peers.
		for i := range knownPeers {
			j := rand.Intn(i + 1)
			knownPeers[i], knownPeers[j] = knownPeers[j], knownPeers[i]
		}

		// If there are less known peers than requested, decrease the number of requested peers.
		numKnownPeers := len(knownPeers)
		if numKnownPeers < numPeers {
			numPeers = numKnownPeers
		}
		if numPeers == 0 {
			return
		}

		for {
			select {
			case peerChan <- *knownPeers[numPeers-1]:
				numPeers--
				if numPeers == 0 {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return peerChan
}

// addPenalty will update the score of the given peer ID in peerScore.
func (p *Peer) addPenalty(pid peer.ID, score int) int {
	return p.peerScore.addPenalty(pid, score)
}

// deletePeer removes the given peer ID from peerScore.
func (p *Peer) deletePeer(pid peer.ID) {
	p.peerScore.deletePeer(pid)
}

// BlockPeer blocks the given peer ID.
func (p *Peer) BlockPeer(pid peer.ID) error {
	return p.connGater.blockPeer(pid)
}

// BlockAndDisconnectPeer blocks the given peer ID and immediately try to close the connection.
func (p *Peer) BlockAndDisconnectPeer(ctx context.Context, pid peer.ID) error {
	err := p.connGater.blockPeer(pid)
	if err != nil {
		return err
	}
	return p.Disconnect(ctx, pid)
}

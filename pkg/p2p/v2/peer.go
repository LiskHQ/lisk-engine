package p2p

import (
	"context"
	"errors"
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
)

const numOfPingMessages = 5         // Number of sent ping messages in Ping service.
const pingTimeout = time.Second * 5 // Ping service timeout in seconds.

// Peer security option type.
type PeerSecurityOption uint8

const (
	PeerSecurityNone  PeerSecurityOption = iota // Do not support any security.
	PeerSecurityTLS                             // Support TLS connections.
	PeerSecurityNoise                           // Support Noise connections.
)

// Peer type - a p2p node.
type Peer struct {
	logger log.Logger
	host   host.Host
}

var autoRelayOptions = []autorelay.Option{
	autorelay.WithPeerSource(peerSource, 1*time.Minute),
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
func NewPeer(ctx context.Context, logger log.Logger, conf Config, addrs []string, security PeerSecurityOption) (*Peer, error) {
	opts := []libp2p.Option{
		// Support default transports (TCP, QUIC, WS)
		libp2p.DefaultTransports,
	}

	if len(addrs) == 0 {
		opts = append(opts, libp2p.NoListenAddrs)
	} else {
		opts = append(opts, libp2p.ListenAddrStrings(addrs...))
	}

	switch security {
	case PeerSecurityNone:
		opts = append(opts, libp2p.NoSecurity)
	case PeerSecurityTLS:
		opts = append(opts, libp2p.Security(libp2ptls.ID, libp2ptls.New))
	case PeerSecurityNoise:
		opts = append(opts, libp2p.Security(noise.ID, noise.New))
	default:
		opts = append(opts, libp2p.NoSecurity)
	}

	// Configure peer to provide a service for other peers for determining their reachability status.
	// TODO - get configuration from config file (GH issue #14)
	if conf.DummyConfigurationFeatureEnable {
		opts = append(opts, libp2p.EnableNATService())
		opts = append(opts, libp2p.AutoNATServiceRateLimit(60, 10, time.Minute))
	}
	opts = append(opts, libp2p.NATPortMap())

	// Enable circuit relay service.
	// TODO - get configuration from config file (GH issue #14)
	opts = append(opts, libp2p.EnableRelay())
	opts = append(opts, libp2p.EnableAutoRelay(autoRelayOptions...))
	if conf.DummyConfigurationFeatureEnable {
		opts = append(opts, libp2p.EnableRelayService(relayServiceOptions...))
	}

	// Enable hole punching service.
	// TODO - get configuration from config file (GH issue #14)
	opts = append(opts, libp2p.EnableHolePunching())

	host, err := libp2p.New(opts...)
	if err != nil {
		return nil, err
	}
	p := &Peer{logger: logger, host: host}
	p.logger.Infof("Peer successfully created")
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
	return p.host.Connect(ctx, peer)
}

// ID returns a peers's identifier.
func (p *Peer) ID() peer.ID {
	return p.host.ID()
}

// Addrs returns a peers's listen addresses.
func (p *Peer) Addrs() []ma.Multiaddr {
	return p.host.Addrs()
}

// P2PAddrs returns a peers's listen addresses in multiaddress format.
func (p *Peer) P2PAddrs() ([]ma.Multiaddr, error) {
	peerInfo := peer.AddrInfo{
		ID:    p.ID(),
		Addrs: p.Addrs(),
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
func peerSource(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
	peerChan := make(chan peer.AddrInfo, 1)

	go func() {
		defer close(peerChan)
		// TODO - get list of peers from a Peer list or some other peer book (GH issue #31)
		testAddr, _ := PeerInfoFromMultiAddr("/ip4/159.223.230.202/tcp/4455/p2p/12D3KooWJapB9gVB2eD2D5RTWdRyFaub9jv9DEELZoSMBPeTimzy")
		// TODO - this is an example of how to construct AddrInfo and send it to the channel
		// peerChan <- peer.AddrInfo{ID: r.ID(), Addrs: r.Addrs()}

		for {
			select {
			case peerChan <- *testAddr:
				numPeers--
				if numPeers == 0 {
					return
				}
				// TODO - get another peer address from a Peer list or some other peer book
				// testAddr = ...
			case <-ctx.Done():
				return
			}
		}
	}()

	return peerChan
}

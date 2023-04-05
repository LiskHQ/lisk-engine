package p2p

import (
	"context"
	"crypto/ed25519"
	"errors"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	ma "github.com/multiformats/go-multiaddr"

	lskcrypto "github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/log"
)

type PeerID = peer.ID
type PeerIDs = peer.IDSlice
type AddrInfo = peer.AddrInfo

const (
	numOfPingMessages        = 5                // Number of sent ping messages in Ping service.
	pingTimeout              = time.Second * 5  // Ping service timeout in seconds.
	connectionTimeout        = time.Second * 30 // Connection timeout in seconds.
	expireTimeOfConnGater    = time.Hour * 24   // Peers will be disconnected after this time
	intervalCheckOfConnGater = time.Second * 10 // Check the blocked list based on this period
)

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
	peerbook  *peerbook
	connGater *connectionGater
}

// Only set Peerstore TTLs once.
var ttlPeerstoreSet = false

var connMgrOptions = []connmgr.Option{
	connmgr.WithGracePeriod(time.Minute),
	connmgr.WithSilencePeriod(10 * time.Second),
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

// newPeer creates a peer with a libp2p host and message protocol.
func newPeer(ctx context.Context, wg *sync.WaitGroup, logger log.Logger, seed []byte, cfg *Config) (*Peer, error) {
	// Create a Peer variable in advance to be able to use it in the libp2p options.
	var p *Peer

	opts := []libp2p.Option{
		// Support default transports (TCP, QUIC, WS)
		libp2p.DefaultTransports,
	}

	// Configure libp2p's identity.
	if len(seed) > 0 {
		stdKey := ed25519.NewKeyFromSeed(lskcrypto.Hash(seed))
		priv, _, err := crypto.KeyPairFromStdKey(&stdKey)
		if err != nil {
			return nil, err
		}
		opts = append(opts, libp2p.Identity(priv))
	}

	if len(cfg.Addresses) == 0 {
		opts = append(opts, libp2p.NoListenAddrs)
	} else {
		opts = append(opts, libp2p.ListenAddrStrings(cfg.Addresses...))
	}

	connGater, err := newConnGater(logger, expireTimeOfConnGater, intervalCheckOfConnGater)
	if err != nil {
		return nil, err
	}

	// Load Blacklist
	connGaterOpt, err := connGater.optionWithBlacklist(cfg.BlacklistedIPs)
	if err != nil {
		return nil, err
	}
	opts = append(opts, connGaterOpt)

	// Configure TTLs for libp2p's peerstore.
	if !ttlPeerstoreSet {
		ttlPeerstoreSet = true
		peerstore.AddressTTL = time.Hour                      // AddressTTL is the expiration time of addresses.
		peerstore.TempAddrTTL = time.Minute * 2               // TempAddrTTL is the ttl used for a short lived address.
		peerstore.RecentlyConnectedAddrTTL = time.Minute * 30 // RecentlyConnectedAddrTTL is used when we recently connected to a peer.
		peerstore.OwnObservedAddrTTL = time.Minute * 30       // OwnObservedAddrTTL is used for our own external addresses observed by peers.
	}

	// Configure connection security.
	security := strings.ToLower(cfg.ConnectionSecurity)
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

	// Configure connection manager to prevent too many connections.
	connMgr, err := connmgr.NewConnManager(
		cfg.MinNumOfConnections, // Lowwater
		cfg.MaxNumOfConnections, // HighWater,
		connMgrOptions...,
	)
	if err != nil {
		return nil, err
	}
	opts = append(opts, libp2p.ConnectionManager(connMgr))

	// Configure peer to provide a service for other peers for determining their reachability status.
	if cfg.EnableNATService {
		opts = append(opts, libp2p.EnableNATService())
		opts = append(opts, libp2p.AutoNATServiceRateLimit(60, 10, time.Minute))
	}
	opts = append(opts, libp2p.NATPortMap())

	// Enable using relay service from other peers. In case a peer is not reachable from the network,
	// it will try to connect to a relay service from other peers.
	if cfg.EnableUsingRelayService {
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
	if cfg.EnableRelayService {
		opts = append(opts, libp2p.EnableRelayService(relayServiceOptions...))
	}

	// Enable hole punching service.
	if cfg.EnableHolePunching {
		opts = append(opts, libp2p.EnableHolePunching())
	}

	host, err := libp2p.New(opts...)
	if err != nil {
		return nil, err
	}

	peerbook, err := newPeerbook(cfg.SeedPeers, cfg.FixedPeers, cfg.BlacklistedIPs)
	if err != nil {
		return nil, err
	}

	p = &Peer{logger: logger, host: host, peerbook: peerbook, connGater: connGater}
	p.logger.Infof("Peer successfully created")
	p.connGater.start(ctx, wg)
	return p, nil
}

// close a peer.
func (p *Peer) close() error {
	err := p.host.Close()
	if err != nil {
		return err
	}
	p.logger.Infof("Peer successfully stopped")
	return nil
}

// Connect to a peer using AddrInfo.
// Direct connection is discouraged. Manually connected peer must be manually disconnected.
func (p *Peer) Connect(ctx context.Context, peer AddrInfo) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, connectionTimeout)
	err := p.host.Connect(ctxWithTimeout, peer)
	cancel()
	if err != nil {
		return err
	}
	p.host.Peerstore().AddAddrs(peer.ID, peer.Addrs, peerstore.PermanentAddrTTL)
	return nil
}

// Disconnect from a peer.
func (p *Peer) Disconnect(peer PeerID) error {
	return p.host.Network().ClosePeer(peer)
}

// ID returns a peers's identifier.
func (p *Peer) ID() PeerID {
	return p.host.ID()
}

// MultiAddress returns a peers's listen addresses.
func (p *Peer) MultiAddress() ([]string, error) {
	peerInfo := peer.AddrInfo{
		ID:    p.ID(),
		Addrs: p.host.Addrs(),
	}
	multiaddr, err := peer.AddrInfoToP2pAddrs(&peerInfo)
	if err != nil {
		return nil, err
	}
	addresses := make([]string, len(multiaddr))
	for i, addr := range multiaddr {
		addresses[i] = addr.String()
	}
	return addresses, nil
}

// ConnectedPeers returns a list of all connected peers IDs.
func (p *Peer) ConnectedPeers() PeerIDs {
	return p.host.Network().Peers()
}

// BlacklistedPeers returns a list of blacklisted peers and their addresses.
func (p *Peer) BlacklistedPeers() []AddrInfo {
	blacklistedPeers := make([]peer.AddrInfo, 0)

	for _, knownPeer := range p.knownPeers() {
		// Check if the peer IP is blacklisted in connectionGater.
		peerFound := false
		for _, ip := range p.connGater.listBannedPeers() {
			for _, addr := range knownPeer.Addrs {
				ipKnownPeer := extractIP(addr)
				if ipKnownPeer == ip.String() {
					blacklistedPeers = append(blacklistedPeers, knownPeer)
					peerFound = true
					break
				}
			}
		}
		if peerFound {
			continue
		}

		// Check if the peer IP address is blacklisted in connectionGater.
		for _, ip := range p.connGater.listBlockedAddrs() {
			peerFound := false
			for _, addr := range knownPeer.Addrs {
				ipKnownPeer := extractIP(addr)
				if ipKnownPeer == ip.String() {
					blacklistedPeers = append(blacklistedPeers, knownPeer)
					peerFound = true
					break
				}
			}
			if peerFound {
				break
			}
		}
	}

	return blacklistedPeers
}

// knownPeers returns a list of all known peers with their addresses.
func (p *Peer) knownPeers() []peer.AddrInfo {
	peers := make([]peer.AddrInfo, 0)

	for _, peerID := range p.host.Peerstore().Peers() {
		peers = append(peers, p.host.Peerstore().PeerInfo(peerID))
	}

	return peers
}

// PingMultiTimes tries to send ping request to a peer for five times.
func (p *Peer) PingMultiTimes(ctx context.Context, peer PeerID) (rtt []time.Duration, err error) {
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
func (p *Peer) Ping(ctx context.Context, peer PeerID) (rtt time.Duration, err error) {
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

		knownPeers := p.knownPeers()

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
			case peerChan <- knownPeers[numPeers-1]:
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

// addPenalty will update the score of the given peer IP in connGater.
// The peer will be banned if the score exceeded MaxPenaltyScore and disconnected immediately.
func (p *Peer) addPenalty(addr ma.Multiaddr, score int) error {
	newScore, err := p.connGater.addPenalty(addr, score)
	if err != nil {
		return err
	}
	if newScore >= MaxPenaltyScore {
		addrInfo, err := AddrInfoFromMultiAddr(addr.String())
		if err != nil {
			return err
		}
		return p.Disconnect(addrInfo.ID)
	}

	return nil
}

// banPeer bans the given peer IP and immediately try to close the connection.
func (p *Peer) banPeer(addr ma.Multiaddr) error {
	_, err := p.connGater.addPenalty(addr, MaxPenaltyScore)
	if err != nil {
		return err
	}
	addrInfo, err := AddrInfoFromMultiAddr(addr.String())
	if err != nil {
		return err
	}
	return p.Disconnect(addrInfo.ID)
}

// Package p2p implements P2P protocol using libp2p library.
package p2p

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p/v2/pubsub"
)

type AddressInfo2 peer.AddrInfo // TODO - Rename this type to AddressInfo. (GH issue #19)

const stopTimeout = time.Second * 5 // P2P service stop timeout in seconds.

// TODO - Move this struct into pkg/engine/config/config.go. Optionally, it could be renamed to NetworkConfig. (GH issue #19)
// Config type - a p2p configuration.
type Config struct {
	Version                  string         `json:"version"`
	Addresses                []string       `json:"addresses"`
	ConnectionSecurity       string         `json:"connectionSecurity"`
	AllowIncomingConnections bool           `json:"allowIncomingConnections"`
	EnableNATService         bool           `json:"enableNATService,omitempty"`
	EnableUsingRelayService  bool           `json:"enableUsingRelayService"`
	EnableRelayService       bool           `json:"enableRelayService,omitempty"`
	EnableHolePunching       bool           `json:"enableHolePunching,omitempty"`
	SeedPeers                []string       `json:"seedPeers"`
	FixedPeers               []string       `json:"fixedPeers,omitempty"`
	BlacklistedIPs           []string       `json:"blackListedIPs,omitempty"`
	KnownPeers               []AddressInfo2 `json:"-"`
	MaxInboundConnections    int            `json:"maxInboundConnections"`
	MaxOutboundConnections   int            `json:"maxOutboundConnections"`
	// GossipSub configuration
	IsSeedNode  bool   `json:"isSeedNode,omitempty"`
	NetworkName string `json:"networkName"`
}

func (c *Config) InsertDefault() error {
	if c.Version == "" {
		c.Version = "1.0"
	}
	if c.Addresses == nil {
		c.Addresses = []string{"/ip4/127.0.0.1/tcp/0", "/ip4/127.0.0.1/udp/0/quic"}
	}
	if c.ConnectionSecurity == "" {
		c.ConnectionSecurity = "tls"
	}
	if c.SeedPeers == nil {
		c.SeedPeers = []string{}
	}
	if c.FixedPeers == nil {
		c.FixedPeers = []string{}
	}
	if c.BlacklistedIPs == nil {
		c.BlacklistedIPs = []string{}
	}
	if c.KnownPeers == nil {
		c.KnownPeers = []AddressInfo2{}
	}
	if c.MaxInboundConnections == 0 {
		c.MaxInboundConnections = 100
	}
	if c.MaxOutboundConnections == 0 {
		c.MaxOutboundConnections = 20
	}
	if c.NetworkName == "" {
		c.NetworkName = "lisk-test"
	}
	return nil
}

// P2P type - a p2p service.
type P2P struct {
	logger log.Logger
	cancel context.CancelFunc
	wg     sync.WaitGroup
	config Config
	*MessageProtocol
	*Peer
	*GossipSub
}

// NewP2P creates a new P2P instance.
func NewP2P(config Config) *P2P {
	return &P2P{config: config, MessageProtocol: NewMessageProtocol(), GossipSub: NewGossipSub()}
}

// Start function starts a P2P and all other related services and handlers.
func (p2p *P2P) Start(logger log.Logger) error {
	logger.Infof("Starting P2P module")
	ctx, cancel := context.WithCancel(context.Background())

	peer, err := NewPeer(ctx, logger, p2p.config)
	if err != nil {
		cancel()
		return err
	}
	peer.peerbook.init(logger)

	p2p.MessageProtocol.Start(ctx, logger, peer)

	sk := pubsub.NewScoreKeeper()
	err = p2p.GossipSub.Start(ctx, &p2p.wg, logger, peer, sk, p2p.config)
	if err != nil {
		cancel()
		return err
	}

	p2p.logger = logger
	p2p.cancel = cancel
	p2p.Peer = peer

	p2p.wg.Add(1)
	go natTraversalService(ctx, &p2p.wg, p2p.config, p2p.MessageProtocol)

	p2p.wg.Add(1)
	go p2pEventHandler(ctx, &p2p.wg, peer)

	p2p.wg.Add(1)
	go gossipSubEventHandler(ctx, &p2p.wg, peer, p2p.GossipSub)

	p2p.wg.Add(1)
	go p2p.peerbook.peerBookService(ctx, &p2p.wg, peer)

	logger.Infof("P2P module successfully started")
	return nil
}

// Stop function stops a P2P.
func (p2p *P2P) Stop() error {
	p2p.cancel()

	waitCh := make(chan struct{})
	go func() {
		p2p.wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		// All services stopped successfully. Nothing to do.
	case <-time.After(stopTimeout):
		return errors.New("P2P module failed to stop")
	}

	p2p.logger.Infof("P2P module successfully stopped")
	return nil
}

// p2pEventHandler handles P2P events.
func p2pEventHandler(ctx context.Context, wg *sync.WaitGroup, p *Peer) {
	defer wg.Done()
	p.logger.Infof("P2P event handler started")

	sub, err := p.host.EventBus().Subscribe(event.WildcardSubscription)
	if err != nil {
		p.logger.Errorf("Failed to subscribe to bus events: %v", err)
	}

	for {
		select {
		case e := <-sub.Out():
			if ev, ok := e.(event.EvtPeerIdentificationFailed); ok {
				p.logger.Debugf("New P2P event received. Peer identification failed: %v", ev)
			}
			if ev, ok := e.(event.EvtLocalProtocolsUpdated); ok {
				p.logger.Debugf("New P2P event received. Local protocols updated: %v", ev)
			}
			if ev, ok := e.(event.EvtLocalAddressesUpdated); ok {
				p.logger.Debugf("New P2P event received. Local addresses updated: %v", ev)
			}
			if ev, ok := e.(event.EvtPeerConnectednessChanged); ok {
				p.logger.Debugf("New P2P event received. Peer connectedness changed: %v", ev)
			}
			if ev, ok := e.(event.EvtPeerProtocolsUpdated); ok {
				p.logger.Debugf("New P2P event received. Peer protocols updated: %v", ev)
			}
			if ev, ok := e.(event.EvtPeerIdentificationCompleted); ok {
				p.logger.Debugf("New P2P event received. Peer identification completed: %v", ev)
			}
		case <-ctx.Done():
			p.logger.Infof("P2P event handler stopped")
			return
		}
	}
}

// ApplyPenalty updates the score of the given PeerID and blocks the peer if the
// score exceeded. Also disconnected the peer immediately.
func (p2p *P2P) ApplyPenalty(ctx context.Context, pid PeerAddrInfo, score int) (err error) {
	newScore := p2p.addPenalty(pid.ID, score)
	if newScore >= 100 {
		p2p.logger.Infof("Banning peer for exceeding max penalty")
		err = p2p.Disconnect(ctx, pid.ID)
		p2p.ps.BlacklistPeer(pid.ID)
	}

	return
}

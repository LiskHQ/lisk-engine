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
)

type PeerID peer.ID

const stopTimeout = time.Second * 5 // P2P service stop timeout in seconds.

// TODO - Move this struct into pkg/engine/config/config.go. Optionally, it could be renamed to NetworkConfig. (GH issue #19)
// Config type - a p2p configuration.
type Config struct {
	Version                  string   `json:"version"`
	Addresses                []string `json:"addresses"`
	ConnectionSecurity       string   `json:"connectionSecurity"`
	AllowIncomingConnections bool     `json:"allowIncomingConnections"`
	EnableNATService         bool     `json:"enableNATService,omitempty"`
	EnableUsingRelayService  bool     `json:"enableUsingRelayService"`
	EnableRelayService       bool     `json:"enableRelayService,omitempty"`
	EnableHolePunching       bool     `json:"enableHolePunching,omitempty"`
	SeedPeers                []PeerID `json:"seedPeers"`
	FixedPeers               []PeerID `json:"fixedPeers,omitempty"`
	BlacklistedIPs           []string `json:"blackListedIPs,omitempty"`
	MaxInboundConnections    int      `json:"maxInboundConnections"`
	MaxOutboundConnections   int      `json:"maxOutboundConnections"`
	// GossipSub configuration
	IsSeedNode  bool     `json:"isSeedNode,omitempty"`
	NetworkName string   `json:"networkName"`
	SeedNodes   []string `json:"seedNodes"` // TODO - Join seedNodes and seedPeers into one field. (GH issue #18)
}

func (c *Config) InsertDefault() error {
	if c.Version == "" {
		c.Version = "1.0"
	}
	if c.Addresses == nil {
		c.Addresses = []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}
	}
	if c.ConnectionSecurity == "" {
		c.ConnectionSecurity = "tls"
	}
	if c.SeedPeers == nil {
		c.SeedPeers = []PeerID{}
	}
	if c.FixedPeers == nil {
		c.FixedPeers = []PeerID{}
	}
	if c.BlacklistedIPs == nil {
		c.BlacklistedIPs = []string{}
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
	if c.SeedNodes == nil {
		c.SeedNodes = []string{}
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
}

// NewP2P creates a new P2P instance.
func NewP2P(config Config) *P2P {
	return &P2P{config: config}
}

// Start function starts a P2P and all other related services.
func (p2p *P2P) Start(logger log.Logger) error {
	logger.Infof("Starting P2P module")
	ctx, cancel := context.WithCancel(context.Background())

	peer, err := NewPeer(ctx, logger, p2p.config)
	if err != nil {
		cancel()
		return err
	}

	mp := NewMessageProtocol(ctx, logger, peer)

	p2p.logger = logger
	p2p.cancel = cancel
	p2p.MessageProtocol = mp
	p2p.Peer = peer

	p2p.wg.Add(1)
	go natTraversalService(ctx, &p2p.wg, p2p.config, mp)

	p2p.wg.Add(1)
	go p2pEventHandler(ctx, &p2p.wg, peer)

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
	p.logger.Infof("P2P service started")

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
			p.logger.Infof("P2P service stopped")
			return
		}
	}
}

// Package p2p implements P2P protocol using libp2p library.
package p2p

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/event"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

const stopTimeout = time.Second * 5 // P2P service stop timeout in seconds.

// TODO - get configuration from config file (GH issue #14)
type Config struct {
	DummyConfigurationFeatureEnable bool
}

// P2P type - a p2p service.
type P2P struct {
	logger log.Logger
	cancel context.CancelFunc
	wg     sync.WaitGroup
	conf   Config
	*MessageProtocol
	*Peer
}

// NewP2P creates a new P2P instance.
func NewP2P() *P2P {
	// TODO - get configuration from config file (GH issue #14)
	return &P2P{conf: Config{DummyConfigurationFeatureEnable: true}}
}

// Start function starts a P2P and all other related services.
func (p2p *P2P) Start(logger log.Logger) error {
	logger.Infof("Starting P2P module")
	ctx, cancel := context.WithCancel(context.Background())

	// TODO - get configuration from config file (GH issue #14)
	peer, err := NewPeer(ctx, logger, p2p.conf, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	if err != nil {
		cancel()
		return err
	}

	mp := NewMessageProtocol(ctx, peer)

	p2p.logger = logger
	p2p.cancel = cancel
	p2p.MessageProtocol = mp
	p2p.Peer = peer

	p2p.wg.Add(1)
	go natTraversalService(ctx, &p2p.wg, p2p.conf, mp)

	p2p.wg.Add(1)
	go p2pService(ctx, &p2p.wg, peer)

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

// p2pService handles P2P events.
func p2pService(ctx context.Context, wg *sync.WaitGroup, p *Peer) {
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

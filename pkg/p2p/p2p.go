// Package p2p implements P2P protocol using libp2p library.
package p2p

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/ipfs/kubo/core/bootstrap"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/LiskHQ/lisk-engine/pkg/engine/config"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	lps "github.com/LiskHQ/lisk-engine/pkg/p2p/pubsub"
)

const stopTimeout = time.Second * 5 // P2P service stop timeout in seconds.

// P2P type - a p2p service.
type P2P struct {
	logger     log.Logger
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	config     *config.NetworkConfig
	bootCloser io.Closer
	*MessageProtocol
	*Peer
	*GossipSub
}

// NewP2P creates a new P2P instance.
func NewP2P(config *config.NetworkConfig) *P2P {
	return &P2P{config: config, MessageProtocol: NewMessageProtocol(), GossipSub: NewGossipSub()}
}

// Start function starts a P2P and all other related services and handlers.
func (p2p *P2P) Start(logger log.Logger) error {
	logger.Infof("Starting P2P module")
	ctx, cancel := context.WithCancel(context.Background())

	peer, err := NewPeer(ctx, &p2p.wg, logger, *p2p.config)
	if err != nil {
		cancel()
		return err
	}
	peer.peerbook.init(logger)

	p2p.MessageProtocol.Start(ctx, logger, peer)

	sk := lps.NewScoreKeeper()
	err = p2p.GossipSub.Start(ctx, &p2p.wg, logger, peer, sk, *p2p.config)
	if err != nil {
		cancel()
		return err
	}

	// Start peer discovery bootstrap process.
	seedPeers, err := lps.ParseAddresses(p2p.config.SeedPeers)
	if err != nil {
		cancel()
		return err
	}
	cfgBootStrap := bootstrap.BootstrapConfigWithPeers(seedPeers)
	cfgBootStrap.MinPeerThreshold = p2p.config.MinNumOfConnections
	bootCloser, err := bootstrap.Bootstrap(peer.ID(), peer.host, nil, cfgBootStrap)
	if err != nil {
		cancel()
		return err
	}

	p2p.logger = logger
	p2p.cancel = cancel
	p2p.Peer = peer
	p2p.bootCloser = bootCloser

	p2p.wg.Add(1)
	go natTraversalService(ctx, &p2p.wg, *p2p.config, p2p.MessageProtocol)

	p2p.wg.Add(1)
	go p2pEventHandler(ctx, &p2p.wg, peer)

	p2p.wg.Add(1)
	go gossipSubEventHandler(ctx, &p2p.wg, peer, p2p.GossipSub)

	logger.Infof("P2P module successfully started")
	return nil
}

// Stop function stops a P2P.
func (p2p *P2P) Stop() error {
	p2p.cancel()
	p2p.bootCloser.Close()

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
func (p2p *P2P) ApplyPenalty(pid string, score int) error {
	return p2p.addPenalty(peer.ID(pid), score)
}

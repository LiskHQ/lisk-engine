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

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

const stopTimeout = time.Second * 5 // P2P service stop timeout in seconds.

// Connection - a connection to p2p network.
type Connection struct {
	logger     log.Logger
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	cfgNet     *Config
	bootCloser io.Closer
	*MessageProtocol
	*Peer
	*GossipSub
}

// NewConnection creates a new P2P instance.
func NewConnection(cfgNet *Config) *Connection {
	if err := cfgNet.insertDefault(); err != nil {
		// if there is an error on configuration, it should not start the package.
		panic(err)
	}
	return &Connection{
		cfgNet:          cfgNet,
		MessageProtocol: newMessageProtocol(cfgNet.ChainID, cfgNet.Version),
		GossipSub:       newGossipSub(cfgNet.ChainID, cfgNet.Version),
	}
}

// Start the P2P and all other related services and handlers.
func (conn *Connection) Start(logger log.Logger, seed []byte) error {
	logger.Infof("Starting P2P module")
	ctx, cancel := context.WithCancel(context.Background())

	peer, err := newPeer(ctx, &conn.wg, logger, seed, conn.cfgNet)
	if err != nil {
		cancel()
		return err
	}
	peer.peerbook.init(logger)

	conn.MessageProtocol.start(ctx, logger, peer)

	sk := newScoreKeeper()
	err = conn.GossipSub.start(ctx, &conn.wg, logger, peer, sk, conn.cfgNet)
	if err != nil {
		cancel()
		return err
	}

	// Start peer discovery bootstrap process.
	seedPeers, err := parseAddresses(conn.cfgNet.SeedPeers)
	if err != nil {
		cancel()
		return err
	}
	cfgBootStrap := bootstrap.BootstrapConfigWithPeers(seedPeers)
	cfgBootStrap.MinPeerThreshold = conn.cfgNet.MinNumOfConnections
	bootCloser, err := bootstrap.Bootstrap(peer.ID(), peer.host, nil, cfgBootStrap)
	if err != nil {
		cancel()
		return err
	}

	conn.logger = logger
	conn.cancel = cancel
	conn.Peer = peer
	conn.bootCloser = bootCloser

	conn.wg.Add(1)
	go natTraversalService(ctx, &conn.wg, conn.cfgNet, conn.MessageProtocol)

	conn.wg.Add(1)
	go connectionEventHandler(ctx, &conn.wg, peer)

	logger.Infof("P2P connection successfully started")
	return nil
}

// Stop the connection to the P2P network.
func (conn *Connection) Stop() error {
	conn.cancel()

	conn.bootCloser.Close()

	waitCh := make(chan struct{})
	go func() {
		conn.wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		// All services stopped successfully. Nothing to do.
	case <-time.After(stopTimeout):
		return errors.New("P2P connection failed to stop")
	}
	if err := conn.Peer.close(); err != nil {
		conn.logger.Error("Fail to close peer")
	}

	conn.logger.Infof("P2P connection successfully stopped")
	return nil
}

// connectionEventHandler handles connection events.
func connectionEventHandler(ctx context.Context, wg *sync.WaitGroup, p *Peer) {
	defer wg.Done()
	p.logger.Infof("Connection event handler started")

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
			p.logger.Infof("Connection event handler stopped")
			return
		}
	}
}

// ApplyPenalty updates the score of the given PeerID and blocks the peer if the
// score exceeded. Also disconnected the peer immediately.
func (conn *Connection) ApplyPenalty(pid string, score int) {
	if err := conn.addPenalty(peer.ID(pid), score); err != nil {
		conn.logger.Errorf("Failed to apply penalty to peer %s: %v", pid, err)
	}
}

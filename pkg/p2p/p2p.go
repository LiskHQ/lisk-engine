// Package p2p implements P2P protocol using libp2p library.
package p2p

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/ipfs/kubo/core/bootstrap"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	ma "github.com/multiformats/go-multiaddr"

	collection "github.com/LiskHQ/lisk-engine/pkg/collection"
	"github.com/LiskHQ/lisk-engine/pkg/log"
)

const stopTimeout = time.Second * 5      // P2P service stop timeout in seconds.
const dropConnTimeout = time.Minute * 30 // Randomly drop one connection after this timeout.

// Connection - a connection to p2p network.
type Connection struct {
	logger          log.Logger
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	cfg             *Config
	bootCloser      io.Closer
	dropConnTimeout time.Duration
	*MessageProtocol
	*Peer
	*GossipSub
}

// NewConnection creates a new P2P instance.
func NewConnection(cfg *Config) *Connection {
	if err := cfg.insertDefault(); err != nil {
		// if there is an error on configuration, it should not start the package.
		panic(err)
	}
	return &Connection{
		cfg:             cfg,
		dropConnTimeout: dropConnTimeout,
		MessageProtocol: newMessageProtocol(cfg.ChainID, cfg.Version),
		GossipSub:       newGossipSub(cfg.ChainID, cfg.Version),
	}
}

// Version returns network version set for the protocol.
func (conn *Connection) Version() string {
	return conn.cfg.Version
}

// Start the P2P and all other related services and handlers.
func (conn *Connection) Start(logger log.Logger, seed []byte) error {
	logger.Infof("Starting P2P module")
	ctx, cancel := context.WithCancel(context.Background())

	peer, err := newPeer(ctx, &conn.wg, logger, seed, conn.cfg)
	if err != nil {
		cancel()
		return err
	}
	peer.peerbook.init(logger)

	conn.MessageProtocol.start(ctx, logger, peer)

	sk := newScoreKeeper()
	err = conn.GossipSub.start(ctx, &conn.wg, logger, peer, sk, conn.cfg)
	if err != nil {
		cancel()
		return err
	}

	// Start peer discovery bootstrap process.
	seedPeers, err := parseAddresses(conn.cfg.SeedPeers)
	if err != nil {
		cancel()
		return err
	}
	cfgBootStrap := bootstrap.BootstrapConfigWithPeers(seedPeers)
	cfgBootStrap.MinPeerThreshold = conn.cfg.MinNumOfConnections
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
	go natTraversalService(ctx, &conn.wg, conn.cfg, conn.MessageProtocol)

	conn.wg.Add(1)
	go connectionEventHandler(ctx, &conn.wg, peer)

	conn.wg.Add(1)
	go connectionsHandler(ctx, &conn.wg, conn)

	conn.wg.Add(1)
	go rateLimiterHandler(ctx, &conn.wg, conn.MessageProtocol.rateLimit)

	addrs, err := conn.MultiAddress()
	if err != nil {
		return err
	}
	logger.Infof("P2P connection successfully started. Listening to: %v", addrs)
	return nil
}

// Stop the connection to the P2P network.
func (conn *Connection) Stop() error {
	// Cancel all subscriptions (to send graft messages (PX exchange) to peers).
	for _, s := range conn.subscriptions {
		s.Cancel()
	}

	// Close all topics.
	for _, t := range conn.topics {
		err := t.Close()
		if err != nil {
			conn.logger.Errorf("Failed to close topic: %v", err)
		}
	}

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

// ApplyPenalty updates the score of the given PeerID (all its IP addresses) and bans the peer if the
// score exceeded. Also disconnected the peer immediately.
func (conn *Connection) ApplyPenalty(pid PeerID, score int) {
	for _, c := range conn.Peer.host.Network().ConnsToPeer(pid) {
		addr := c.RemoteMultiaddr().String() + "/p2p/" + pid.String()
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			conn.logger.Errorf("Failed to create a new multiaddr: %s", err)
			return
		}
		if err := conn.addPenalty(maddr, score); err != nil {
			conn.logger.Errorf("Failed to apply penalty to peer %s: %v", pid, err)
		}
	}
}

// BanPeer bans the given PeerID (all its IP addresses) and disconnects the peer immediately.
func (conn *Connection) BanPeer(pid PeerID) {
	for _, c := range conn.Peer.host.Network().ConnsToPeer(pid) {
		addr := c.RemoteMultiaddr().String() + "/p2p/" + pid.String()
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			conn.logger.Errorf("Failed to create a new multiaddr: %s", err)
			return
		}
		if err := conn.Peer.banPeer(maddr); err != nil {
			conn.logger.Errorf("Failed to ban peer %s: %v", pid, err)
		}
	}
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

// connectionsHandler handles P2P connections (randomly drop a connection to a peer to make room for a new one).
func connectionsHandler(ctx context.Context, wg *sync.WaitGroup, conn *Connection) {
	defer wg.Done()
	conn.logger.Infof("Connections handler started")

	timerDrop := time.NewTicker(conn.dropConnTimeout)

	for {
		select {
		case <-timerDrop.C:
			allConns := conn.Peer.host.Network().Conns()

			// Remove fixed peers connections from the list.
			conns := make([]network.Conn, 0)
			for _, c := range allConns {
				index := collection.FindIndex(conn.cfg.FixedPeers, func(peer string) bool {
					return peer == c.RemoteMultiaddr().String()+"/p2p/"+c.RemotePeer().String()
				})

				if index == -1 {
					conns = append(conns, c)
				}
			}

			// Only drop a connection if we have more than the minimum number of connections.
			if len(allConns) > conn.cfg.MinNumOfConnections {
				conn.logger.Debugf("Dropping a random connection")
				rand.Seed(time.Now().UnixNano())
				rand.Shuffle(len(conns), func(i, j int) { conns[i], conns[j] = conns[j], conns[i] })
				if err := conn.Disconnect(conns[0].RemotePeer()); err != nil {
					conn.logger.Errorf("Failed to close connection to peer %s: %v", conns[0].RemotePeer(), err)
				}
			}
			timerDrop.Reset(conn.dropConnTimeout)
		case <-ctx.Done():
			conn.logger.Infof("Connections handler stopped")
			return
		}
	}
}

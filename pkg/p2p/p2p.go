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
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/LiskHQ/lisk-engine/pkg/engine/config"
	"github.com/LiskHQ/lisk-engine/pkg/log"
)

const stopTimeout = time.Second * 5       // P2P service stop timeout in seconds.
const dropConnTimeout = time.Minute * 30  // Randomly drop one connection after this timeout.
const manageConnTimeout = time.Minute * 5 // Manage number of connections periodically.

// Connection - a connection to p2p network.
type Connection struct {
	logger            log.Logger
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	cfg               *config.NetworkConfig
	bootCloser        io.Closer
	dropConnTimeout   time.Duration
	manageConnTimeout time.Duration
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
		cfg:               cfg,
		dropConnTimeout:   dropConnTimeout,
		manageConnTimeout: manageConnTimeout,
		MessageProtocol:   newMessageProtocol(cfgNet.ChainID, cfgNet.Version),
		GossipSub:         newGossipSub(cfgNet.ChainID, cfgNet.Version),
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

// ApplyPenalty updates the score of the given PeerID and blocks the peer if the
// score exceeded. Also disconnected the peer immediately.
func (conn *Connection) ApplyPenalty(pid string, score int) {
	if err := conn.addPenalty(peer.ID(pid), score); err != nil {
		conn.logger.Errorf("Failed to apply penalty to peer %s: %v", pid, err)
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
	timerConnect := time.NewTicker(conn.manageConnTimeout)

	for {
		select {
		case <-timerDrop.C:
			conns := conn.Peer.host.Network().Conns()
			// Only drop a connection if we have more than the minimum number of connections.
			if len(conns) > conn.cfgNet.MinNumOfConnections {
				conn.logger.Debugf("Dropping a random connection")
				rand.Seed(time.Now().UnixNano())
				rand.Shuffle(len(conns), func(i, j int) { conns[i], conns[j] = conns[j], conns[i] })
				if err := conn.Disconnect(conns[0].RemotePeer()); err != nil {
					conn.logger.Errorf("Failed to close connection to peer %s: %v", conns[0].RemotePeer(), err)
				}
			}
			timerDrop.Reset(conn.dropConnTimeout)
		case <-timerConnect.C:
			conns := conn.Peer.host.Network().Conns()
			// Only try to connect to known peers if we have less than two thirds of the maximum number of connections.
			if len(conns) < conn.cfgNet.MaxNumOfConnections*2/3 {
				conn.logger.Debugf("Trying to connect to a random peer")

				peers := conn.knownPeers()
				rand.Seed(time.Now().UnixNano())
				rand.Shuffle(len(peers), func(i, j int) { peers[i], peers[j] = peers[j], peers[i] })

				for _, peer := range peers {
					// Skip ourselves.
					if peer.ID == conn.Peer.host.ID() {
						continue
					}
					// Skip peers we are already connected to.
					if conn.Peer.host.Network().Connectedness(peer.ID) == network.Connected {
						continue
					}

					for _, addr := range peer.Addrs {
						addrInfo, err := AddrInfoFromMultiAddr(addr.String() + "/p2p/" + peer.ID.Pretty())
						if err != nil {
							continue
						}
						if err := conn.Connect(ctx, *addrInfo); err != nil {
							conn.logger.Debugf("Failed to connect to peer %s: %v", peer.ID.Pretty(), err)
							continue
						} else {
							break
						}
					}

					// Stop trying to connect to peers if we reached the maximum number of connections.
					if len(conn.Peer.host.Network().Conns()) >= conn.cfgNet.MaxNumOfConnections {
						break
					}
				}
			}

			timerConnect.Reset(conn.manageConnTimeout)
		case <-ctx.Done():
			conn.logger.Infof("Connections handler stopped")
			return
		}
	}
}

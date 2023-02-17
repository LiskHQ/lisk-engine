package p2p

import (
	"context"
	"fmt"

	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p/v1/addressbook"
	"github.com/LiskHQ/lisk-engine/pkg/p2p/v1/socket"
)

const (
	endpointGetNodeInfo = "getNodeInfo"
	endpointGetPeers    = "getPeers"
	endpointPing        = "ping"
)

// ConnectionConfig for a p2p network.
type ConnectionConfig struct {
	SeedAddresses         []*addressbook.Address
	FixedAddresses        []*addressbook.Address
	WhitelistedAddresses  []*addressbook.Address
	BannedIPs             []string
	MaxInboundConnection  int
	MaxOutboundConnection int
}

// Connection to a p2p network.
type Connection struct {
	active      bool
	logger      log.Logger
	cfg         *ConnectionConfig
	event       []chan *Event
	peerPool    *peerPool
	addressBook *addressbook.Book
	server      *server
}

// PeerInfo returns information about the peer.
type PeerInfo interface {
	ID() string
	IP() string
	Port() int
	Options() []byte
}

// AddressInfo returns exposable address info.
type AddressInfo interface {
	IP() string
	Port() int
}

func getSocketHandle(handler RPCHandler) socket.RPCHandler {
	return func(w socket.ResponseWriter, r *socket.Request) {
		req := Request(*r)
		handler(w, &req)
	}
}

// NewConnection creates new connection.
func NewConnection(cfg *ConnectionConfig) *Connection {
	conn := &Connection{
		cfg:   cfg,
		event: []chan *Event{},
	}
	seed := crypto.RandomBytes(4)
	conn.addressBook = addressbook.NewBook(seed, nil, cfg.SeedAddresses, cfg.FixedAddresses, cfg.WhitelistedAddresses)
	conn.peerPool = newPeerPool(&PeerPoolConfig{
		MaxOutboundConnection: cfg.MaxOutboundConnection,
		MaxInboundConnection:  cfg.MaxInboundConnection,
		Addressbook:           conn.addressBook,
	})
	conn.server = newServer(&serverConfig{
		pool: conn.peerPool,
	})
	rpcHandlers := map[string]socket.RPCHandler{}
	rpcHandlers[endpointGetPeers] = conn.peerPool.handleGetPeers()
	rpcHandlers[endpointGetNodeInfo] = conn.peerPool.handleNodeInfo()
	rpcHandlers[endpointPing] = conn.peerPool.handlePing()
	conn.peerPool.registerRPCHandlers(rpcHandlers)
	return conn
}

func (c *Connection) RegisterRPCHandler(endpoint string, handler RPCHandler) error {
	if c.active {
		return fmt.Errorf("p2P connection has already started. New endpoint %s cannot be added", endpoint)
	}
	return c.peerPool.registerRPCHandler(endpoint, getSocketHandle(handler))
}

func (c *Connection) RegisterEventHandler(name string, handler EventHandler) error {
	if c.active {
		return fmt.Errorf("p2P connection has already started. New handler %s cannot be added", name)
	}
	return c.peerPool.registerEventHandler(name, handler)
}

// Start P2P connection.
func (c *Connection) Start(logger log.Logger, nodeInfo *NodeInfo) error {
	c.active = true
	c.logger = logger
	c.server.nodeInfo = nodeInfo
	go c.peerPool.start(logger, nodeInfo)
	if c.cfg.MaxInboundConnection > 0 {
		c.logger.Infof("Starting P2P server at %d", nodeInfo.Port)
		go func() {
			if err := c.server.start(logger, nodeInfo); err != nil {
				c.logger.Errorf("Fail to start server with %w", err)
			}
		}()
	}
	return nil
}

// Stop P2P connection.
func (c *Connection) Stop() error {
	c.logger.Info("Stopping P2P connection")
	if err := c.server.stop(); err != nil {
		return err
	}
	c.peerPool.stop()
	c.active = false
	return nil
}

// PostNodeInfo sends current node information to a node.
func (c *Connection) PostNodeInfo(ctx context.Context, data []byte) {
	c.peerPool.postNodeInfo(ctx, data)
}

// Send data to partial set of peers.
func (c *Connection) Send(ctx context.Context, event string, data []byte) {
	c.peerPool.send(ctx, event, data)
}

// Broadcast data to all peers.
func (c *Connection) Broadcast(ctx context.Context, event string, data []byte) {
	c.peerPool.broadcast(ctx, event, data)
}

// Request data from random outbound peer.
func (c *Connection) Request(ctx context.Context, procedure string, data []byte) Response {
	return c.peerPool.request(ctx, procedure, data)
}

// RequestFrom data from a particular peer.
func (c *Connection) RequestFrom(ctx context.Context, peerID string, procedure string, data []byte) Response {
	return c.peerPool.requestFromPeer(ctx, peerID, procedure, data)
}

// ApplyPenalty to a particular peer.
func (c *Connection) ApplyPenalty(peerID string, score int) {
	c.peerPool.applyPenalty(peerID, score)
}

// ConnectedPeers returns all currently connected peers.
func (c *Connection) ConnectedPeers(onlyTrusted bool) []PeerInfo {
	connected := c.peerPool.getConnectedPeers(onlyTrusted)
	result := make([]PeerInfo, len(connected))
	for i, peer := range connected {
		result[i] = peer
	}
	return result
}

// DisconnectedAddresses returns all disconnected addresses.
func (c *Connection) DisconnectedAddresses() []AddressInfo {
	allAddresses := c.addressBook.All()
	result := []AddressInfo{}
	connectedIDs := c.peerPool.getConnectedIDs()
	for _, addr := range allAddresses {
		if _, exist := connectedIDs[getIDFromAddress(addr)]; !exist {
			result = append(result, addr)
		}
	}
	return result
}

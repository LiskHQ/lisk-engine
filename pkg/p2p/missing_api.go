package p2p

import (
	"context"
	"errors"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p/v1/socket"
)

/*************************************************************************************************************************
                                                        message.go
*************************************************************************************************************************/

// Request holds request from a peer.
type Request socket.Request

type ResponseOld socket.Response

/*************************************************************************************************************************
                                                        connection.go
*************************************************************************************************************************/

// ConnectionConfig for a p2p network.
type ConnectionConfig struct {
	SeedAddresses         []*Address
	FixedAddresses        []*Address
	WhitelistedAddresses  []*Address
	BannedIPs             []string
	MaxInboundConnection  int
	MaxOutboundConnection int
}

type Connection struct{}

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

// NewConnection creates new connection.
func NewConnection(cfg *ConnectionConfig) *Connection {
	conn := &Connection{}
	return conn
}

// Start P2P connection.
func (c *Connection) Start(logger log.Logger, nodeInfo *NodeInfo) error {
	return nil
}

// Stop P2P connection.
func (c *Connection) Stop() error {
	return nil
}

// PostNodeInfo sends current node information to a node.
func (c *Connection) PostNodeInfo(ctx context.Context, data []byte) {
}

// Send data to partial set of peers.
func (c *Connection) Send(ctx context.Context, event string, data []byte) {
}

// In old P2P eager/lazy push is handled in the application, but since now gossipsub maintains it,
// maybe we need to slightly modify an interface?
// Broadcast data to all peers.
func (c *Connection) Broadcast(ctx context.Context, event string, data []byte) {
}

// Request data from random outbound peer.
func (c *Connection) Request(ctx context.Context, procedure string, data []byte) ResponseOld {
	return socket.NewResponse(nil, errors.New("implementation is needed"))
}

// RequestFrom data from a particular peer.
func (c *Connection) RequestFrom(ctx context.Context, peerID string, procedure string, data []byte) ResponseOld {
	return socket.NewResponse(nil, errors.New("implementation is needed"))
}

// ApplyPenalty to a particular peer.
func (c *Connection) ApplyPenalty(peerID string, score int) {
}

// ConnectedPeers returns all currently connected peers.
func (c *Connection) ConnectedPeers(onlyTrusted bool) []PeerInfo {
	result := []PeerInfo{}
	return result
}

// DisconnectedAddresses returns all disconnected addresses.
func (c *Connection) DisconnectedAddresses() []AddressInfo {
	result := []AddressInfo{}
	return result
}

/*************************************************************************************************************************
                                                        node.go
*************************************************************************************************************************/

// In old P2P package NodeInfo (height/maxHeight, prevoted, etc) is handled in the P2P package, but this caused not to be able to
// fetch some information in the application layer. Therefore, we can remove this from P2P interface and handle it in the application.
// NodeInfo contains the information about the host node.
type NodeInfo struct {
	ChainID          string `json:"chainID" fieldNumber:"1"`
	NetworkVersion   string `json:"networkVersion"  fieldNumber:"2"`
	Nonce            string `json:"nonce"  fieldNumber:"3"`
	AdvertiseAddress bool   `json:"-"  fieldNumber:"4"`
	Port             int    `json:"port"`
	Options          []byte `json:"-" fieldNumber:"5"`
}

/*************************************************************************************************************************
                                                        addressbook/address.go
*************************************************************************************************************************/

// Address defines a content of the book.
type Address struct {
	ip        string `fieldNumber:"1"`
	port      uint32 `fieldNumber:"2"`
	advertise bool
	createdAt time.Time
}

// NewAddress creates new address set.
func NewAddress(ip string, port int) *Address {
	address := &Address{
		ip:        ip,
		port:      uint32(port),
		advertise: true,
		createdAt: time.Now(),
	}
	return address
}

package p2p

import (
	"context"
	"errors"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p/socket"
)

/*************************************************************************************************************************
                                                        message.go
*************************************************************************************************************************/

// Event holds event message from a peer
//
//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen
type Event struct {
	peerID string
	event  string
	data   []byte
}

// PeerID returns sender peer id.
func (e *Event) PeerID() string {
	return e.peerID
}

// Event returns event type.
func (e *Event) Event() string {
	return e.event
}

// Data returns event payload.
func (e *Event) Data() []byte {
	return e.data
}

// Request holds request from a peer.
type Request socket.Request

type Response socket.Response

// RPCHandler is a definition for accepting the RPC request from a peer.
type RPCHandler func(w ResponseWriter, r *Request)

type EventHandler func(event *Event)

// ResponseWriter is a interface for handler to write.
type ResponseWriter socket.ResponseWriter

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

func (c *Connection) RegisterRPCHandler(endpoint string, handler RPCHandler) error {
	return nil
}

func (c *Connection) RegisterEventHandler(name string, handler EventHandler) error {
	return nil
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

// Broadcast data to all peers.
func (c *Connection) Broadcast(ctx context.Context, event string, data []byte) {
}

// Request data from random outbound peer.
func (c *Connection) Request(ctx context.Context, procedure string, data []byte) Response {
	return socket.NewResponse(nil, errors.New("implementation is needed"))
}

// RequestFrom data from a particular peer.
func (c *Connection) RequestFrom(ctx context.Context, peerID string, procedure string, data []byte) Response {
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

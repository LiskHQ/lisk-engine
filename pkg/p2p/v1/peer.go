package p2p

import (
	"context"
	"fmt"
	"time"

	"github.com/gorilla/websocket"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p/v1/addressbook"
	"github.com/LiskHQ/lisk-engine/pkg/p2p/v1/socket"
)

const (
	eventPeerMessage = "eventPeerMessage"
	eventPeerBanned  = "eventPeerBanned"
	eventPeerClosed  = "eventPeerClosed"

	rpcGetPeers       = "getPeers"
	rpcGetNodeInfo    = "getNodeInfo"
	eventPostNodeInfo = "postNodeInfo"

	minPingInterval = 20
	maxPingInterval = 60
)

// Peer represents a connected Peer.
type Peer struct {
	ctx            context.Context
	logger         log.Logger
	ip             string
	port           uint32
	penalty        int
	eventCh        chan *peerEvent
	closeCh        chan bool
	socket         *socket.Socket
	nodeInfo       *NodeInfo
	netgroup       addressbook.Network
	latency        int64
	connectionTime int64
	latecyTicker   *bufferedTicker
	requestCount   int64
	responseCount  int64
}

// Peers holds multiple peers.
type Peers []*Peer

// Include checks if address list includes the ip and port.
func (p Peers) Include(id string) bool {
	for _, peer := range p {
		if peer.ID() == id {
			return true
		}
	}
	return false
}

// Filter address by input address, and return new list.
func (p Peers) Filter(excludes []*Peer) Peers {
	filtered := Peers{}
	for _, peer := range p {
		if !Peers(excludes).Include(peer.ID()) {
			filtered = append(filtered, peer)
		}
	}
	return filtered
}

// Unique returns unique address.
func (p Peers) Unique() Peers {
	uniqueMap := map[string]*Peer{}
	for _, peer := range p {
		if _, exist := uniqueMap[peer.ID()]; !exist {
			uniqueMap[peer.ID()] = peer
		}
	}

	filtered := make(Peers, len(uniqueMap))
	index := 0
	for _, addr := range uniqueMap {
		filtered[index] = addr
	}
	return filtered
}

type peerEvent struct {
	peerID string
	kind   string
	event  string
	data   []byte
}

// ID is identifier for peer.
func (p *Peer) ID() string {
	return fmt.Sprintf("%s:%d", p.ip, p.port)
}

// IP is getter for ip of the peer.
func (p *Peer) IP() string {
	return p.ip
}

// Port is getter for port of the peer.
func (p *Peer) Port() int {
	return int(p.port)
}

// Options is getter for options of the peer.
func (p *Peer) Options() []byte {
	if p.nodeInfo != nil {
		return p.nodeInfo.Options
	}
	return nil
}

func (p *Peer) responseRate() float64 {
	return float64(p.responseCount) / float64(p.requestCount)
}

func (p *Peer) send(ctx context.Context, event string, data []byte) {
	if err := p.socket.Send(ctx, event, data); err != nil {
		p.logger.Debug("Fail to send data with %w", err)
	}
}

func (p *Peer) request(ctx context.Context, method string, data []byte) Response {
	p.requestCount++
	resp := p.socket.Request(ctx, method, data)
	if resp.Err() == nil {
		p.responseCount++
	}
	return resp
}

func (p *Peer) getPeers(ctx context.Context) ([]*addressbook.Address, error) {
	p.logger.Debug("Requesting peers list")
	resp := p.socket.Request(ctx, rpcGetPeers, nil)
	if resp.Err() != nil {
		return nil, resp.Err()
	}
	body := &rpcGetPeersResponse{}
	if err := body.Decode(resp.Data()); err != nil {
		return nil, err
	}
	addresses := make([]*addressbook.Address, len(body.Peers))
	for i, peerBytes := range body.Peers {
		addr, err := addressbook.NewAddressFromBytes(peerBytes)
		if err != nil {
			return nil, err
		}
		addr.SetSource(p.ip)
		addresses[i] = addr
	}
	return addresses, nil
}

func (p *Peer) updateNodeInfo(ctx context.Context) error {
	p.logger.Debug("Requesting node info")
	resp := p.socket.Request(ctx, rpcGetNodeInfo, nil)
	if resp.Err() != nil {
		return resp.Err()
	}
	nodeInfo := &NodeInfo{}
	if err := nodeInfo.Decode(resp.Data()); err != nil {
		return err
	}
	p.nodeInfo = nodeInfo
	return nil
}

func (p *Peer) close() {
	p.logger.Debugf("Closing peer connection")
	p.socket.Close()
	close((p.closeCh))
	p.logger.Debugf("Closed peer connection")
}

func (p *Peer) bind() {
	// measure latency in different process
	go p.measureLatency()
	subscription := p.socket.Subscribe()
	for {
		select {
		case msg, ok := <-subscription:
			if !ok {
				close(p.eventCh)
				return
			}
			p.handleSocketMessage(msg)
		case <-p.closeCh:
			close(p.eventCh)
			return
		}
	}
}

func (p *Peer) measureLatency() {
	var cancel context.CancelFunc
	for {
		select {
		case <-p.closeCh:
			if cancel != nil {
				cancel()
			}
			p.latecyTicker.stop()
			return
		case <-p.latecyTicker.C:
			var ctx context.Context
			ctx, cancel = context.WithCancel(p.ctx)
			defer cancel()
			latency, err := p.socket.Ping(ctx)
			if err != nil {
				p.logger.Errorf("Fail to get latency with %v", err)
				continue
			}
			p.latency = latency
		}
	}
}

func (p *Peer) subscribe() <-chan *peerEvent {
	return p.eventCh
}

func (p *Peer) handleSocketMessage(msg *socket.Message) {
	switch msg.Kind() {
	case socket.EventMessageReceived:
		if msg.Event() == eventPostNodeInfo {
			p.handlePostNodeInfo(msg)
			return
		}
		p.eventCh <- &peerEvent{
			peerID: p.ID(),
			kind:   eventPeerMessage,
			event:  msg.Event(),
			data:   msg.Data(),
		}
	case socket.EventSocketClosed:
		p.logger.Debug("Close socket event received")
		p.eventCh <- &peerEvent{
			peerID: p.ID(),
			kind:   eventPeerClosed,
		}
	case socket.EventSocketError:
		p.logger.Errorf("Socket error received with %v", msg.Err())
	case socket.EventMessageLimitExceeded:
		p.logger.Info("Applying penalty for exceeding message rate limit")
		p.applyPenalty(10)
	case socket.EventGeneralMessageReceived:
		p.logger.Info("Applying penalty for accessing not supported message type")
		p.applyPenalty(10)
	}
}

func (p *Peer) applyPenalty(score int) {
	p.logger.Debugf("Applying penalty")
	p.penalty += score
	if p.penalty >= 100 {
		p.logger.Infof("Banning peer for exceeding max penalty")
		p.eventCh <- &peerEvent{
			peerID: p.ID(),
			kind:   eventPeerBanned,
		}
	}
}

func (p *Peer) handlePostNodeInfo(event *socket.Message) {
	info := &NodeInfo{}

	if err := info.Decode(event.Data()); err != nil {
		p.logger.Errorf("Fail to decode with %v", err)
		p.applyPenalty(100)
		return
	}
	p.nodeInfo = info
}

func (p *Peer) createHandshakeHandle() socket.HandshakeHandler {
	return func(conn *websocket.Conn) error {
		return nil
	}
}

func newInboundPeer(addr *addressbook.Address, conn *websocket.Conn, handlers map[string]socket.RPCHandler, logger log.Logger) *Peer {
	sock := socket.NewServerSocket(conn, logger)
	sock.RegisterRPCHandlers(handlers)
	network, _ := addressbook.GetNetwork(addr.IP())
	peer := &Peer{
		ip: addr.IP(),
		// TODO: for inbound peer, is this correct port?
		netgroup:       network,
		ctx:            context.Background(),
		connectionTime: time.Now().Unix(),
		port:           uint32(addr.Port()),
		eventCh:        make(chan *peerEvent),
		closeCh:        make(chan bool),
		latecyTicker:   newRandomBufferedTicker(minPingInterval*time.Second, maxPingInterval*time.Second),
		socket:         sock,
	}
	sock.RegisterHandshakeHandler(peer.createHandshakeHandle())
	go sock.StartServerProcess()
	peer.logger = logger.With("peerID", peer.ID())
	return peer
}

func newOutboundPeer(ctx context.Context, addr *addressbook.Address, conn *websocket.Conn, connector socket.Connector, handlers map[string]socket.RPCHandler, logger log.Logger) (*Peer, error) {
	sock, err := socket.NewClientSocket(ctx, conn, connector, logger)
	if err != nil {
		return nil, err
	}
	sock.RegisterRPCHandlers(handlers)
	network, err := addressbook.GetNetwork(addr.IP())
	if err != nil {
		return nil, err
	}
	peer := &Peer{
		ip:             addr.IP(),
		ctx:            ctx,
		netgroup:       network,
		connectionTime: time.Now().Unix(),
		port:           uint32(addr.Port()),
		eventCh:        make(chan *peerEvent),
		closeCh:        make(chan bool),
		latecyTicker:   newRandomBufferedTicker(minPingInterval*time.Second, maxPingInterval*time.Second),
		socket:         sock,
	}
	sock.RegisterHandshakeHandler(peer.createHandshakeHandle())
	peer.logger = logger.With("peerID", peer.ID())
	return peer, nil
}

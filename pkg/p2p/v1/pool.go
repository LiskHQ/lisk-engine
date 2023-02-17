package p2p

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/LiskHQ/lisk-engine/pkg/collection/ints"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p/v1/addressbook"
	"github.com/LiskHQ/lisk-engine/pkg/p2p/v1/socket"
)

const (
	populateInterval  = 10
	discoveryInterval = 30
	sendPeerLimit     = 16
	shuffleInterval   = 5 * 60
	connectionTimeout = 3
	// EventPeerMessage is for event message from a peer.
	EventPeerMessage = "EventPeerMessage"
)

// PeerPoolConfig for all the sttings of peer pool.
type PeerPoolConfig struct {
	MaxOutboundConnection int
	MaxInboundConnection  int
	RPCHandlers           map[string]socket.RPCHandler
	Addressbook           *addressbook.Book
}

// peerPool maintains all connected peers.
type peerPool struct {
	ctx                   context.Context
	logger                log.Logger
	active                bool
	mutex                 *sync.RWMutex
	addressbook           *addressbook.Book
	nodeInfo              *NodeInfo
	maxOutboundConnection int
	maxInboundConnection  int
	rpcHandlers           map[string]socket.RPCHandler
	eventHandlers         map[string]EventHandler
	rpcCounter            *rpcCounter

	eventCh         chan *Event
	outboundPeers   map[string]*Peer
	inboundPeers    map[string]*Peer
	populateTicker  *bufferedTicker
	shuffleTicker   *bufferedTicker
	discoveryTicker *bufferedTicker
	close           chan bool
}

type rpcCounter struct {
	mutex   *sync.RWMutex
	counter map[string]map[string]int
}

func newRPCCounter() *rpcCounter {
	return &rpcCounter{
		mutex:   new(sync.RWMutex),
		counter: map[string]map[string]int{},
	}
}

func (c *rpcCounter) increment(peerID string, procedure string) int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	_, exist := c.counter[peerID]
	if !exist {
		c.counter[peerID] = map[string]int{}
	}
	val, exist := c.counter[peerID][procedure]
	if exist {
		val++
	} else {
		val = 1
	}
	c.counter[peerID][procedure] = val
	return val
}

// newPeerPool returns a peer pool with the configuration.
func newPeerPool(cfg *PeerPoolConfig) *peerPool {
	return &peerPool{
		maxOutboundConnection: cfg.MaxOutboundConnection,
		maxInboundConnection:  cfg.MaxInboundConnection,
		addressbook:           cfg.Addressbook,
		ctx:                   context.Background(),
		mutex:                 new(sync.RWMutex),
		outboundPeers:         make(map[string]*Peer),
		inboundPeers:          make(map[string]*Peer),
		rpcCounter:            newRPCCounter(),
		eventCh:               make(chan *Event, (cfg.MaxInboundConnection+cfg.MaxInboundConnection)*5),
		eventHandlers:         make(map[string]EventHandler),
		rpcHandlers:           make(map[string]socket.RPCHandler),
		close:                 make(chan bool),
	}
}

// start the peer pool.
func (p *peerPool) start(logger log.Logger, nodeInfo *NodeInfo) {
	p.logger = logger
	p.nodeInfo = nodeInfo
	p.populateTicker = newBufferedTicker(populateInterval * time.Second)
	p.discoveryTicker = newBufferedTicker(discoveryInterval * time.Second)
	p.shuffleTicker = newBufferedTicker(shuffleInterval * time.Second)
	p.active = true
	go p.discovery()
	go p.populateWorker(p.populateTicker.C)
	go p.discoveryWorker(p.discoveryTicker.C)
	go p.shuffleWorker(p.discoveryTicker.C)
	for range p.close {
		p.drainEvent()
		close(p.eventCh)
		return
	}
}

func (p *peerPool) registerRPCHandlers(handlers map[string]socket.RPCHandler) {
	p.rpcHandlers = handlers
}

func (p *peerPool) registerRPCHandler(endpoint string, handler socket.RPCHandler) error {
	_, exist := p.rpcHandlers[endpoint]
	if exist {
		return fmt.Errorf("endpoint %s is already registered", endpoint)
	}
	p.rpcHandlers[endpoint] = handler
	return nil
}

func (p *peerPool) registerEventHandler(endpoint string, handler EventHandler) error {
	_, exist := p.eventHandlers[endpoint]
	if exist {
		return fmt.Errorf("eventHandler %s is already registered", endpoint)
	}
	p.eventHandlers[endpoint] = handler
	return nil
}

func (p *peerPool) drainEvent() {
	for {
		select {
		case <-p.eventCh:
		default:
			return
		}
	}
}

// stop the peer pool.
func (p *peerPool) stop() {
	p.populateTicker.stop()
	p.shuffleTicker.stop()
	p.discoveryTicker.stop()
	for _, peer := range p.getOutboundPeers() {
		p.disconnectAndRemove(peer)
	}
	for _, peer := range p.getInboundPeers() {
		p.disconnectAndRemove(peer)
	}
	close(p.close)
	p.logger.Debug("Stopped peer pool")
}

func (p *peerPool) populateWorker(notifier <-chan time.Time) {
	for range notifier {
		p.populate()
	}
}

func (p *peerPool) discoveryWorker(notifier <-chan time.Time) {
	for range notifier {
		p.discovery()
	}
}

func (p *peerPool) shuffleWorker(notifier <-chan time.Time) {
	for range notifier {
		p.shuffle()
	}
}

// addConnection adds a connection as incoming connection.
func (p *peerPool) addConnection(conn *websocket.Conn, addr *addressbook.Address) {
	if err := p.addressbook.AddNewAddress(addr); err != nil {
		p.logger.Errorf("Fail to add address to the book with %v", err)
		return
	}
	if len(p.inboundPeers) >= p.maxInboundConnection {
		p.evictInbound()
	}
	// add to connection
	peer := newInboundPeer(addr, conn, p.rpcHandlers, p.logger)
	p.addInboundPeer(peer)
	go peer.bind()
	go p.bind(peer)
}

// request information from random outbound peer.
func (p *peerPool) request(ctx context.Context, procedure string, data []byte) socket.Response {
	outboundSize := len(p.outboundPeers)
	if outboundSize == 0 {
		return socket.NewResponse(nil, errors.New("no peer to request"))
	}
	candidates := p.getOutboundPeers()
	index := rand.Intn(outboundSize)
	// select random outbound peer
	return candidates[index].request(ctx, procedure, data)
}

// requestFromPeer information from a peer.
func (p *peerPool) requestFromPeer(ctx context.Context, peerID string, procedure string, data []byte) socket.Response {
	peer, exist := p.getPeer(peerID)
	if !exist {
		return socket.NewResponse(nil, fmt.Errorf("peerID %s do not exist to make request", peerID))
	}
	return peer.request(ctx, procedure, data)
}

// requestFromPeer information from a peer.
func (p *peerPool) postNodeInfo(ctx context.Context, data []byte) {
	p.nodeInfo.Options = data
	encodedNodeInfo, err := p.nodeInfo.Encode()
	if err != nil {
		p.logger.Errorf("Node info failed to be encoded with error %v", err)
		return
	}
	p.broadcast(ctx, eventPostNodeInfo, encodedNodeInfo)
}

func (p *peerPool) handleNodeInfo() socket.RPCHandler {
	return func(w socket.ResponseWriter, r *socket.Request) {
		peer, exist := p.getPeer(r.PeerID)
		if exist {
			val := p.rpcCounter.increment(peer.ID(), endpointGetNodeInfo)
			if val > 1 {
				peer.applyPenalty(10)
			}
		}
		encodedNodeInfo, err := p.nodeInfo.Encode()
		if err != nil {
			p.logger.Errorf("Node info failed to be encoded with error %v", err)
			w.Error(err)
			return
		}
		w.Write(encodedNodeInfo)
	}
}

func (p *peerPool) handlePing() socket.RPCHandler {
	return func(w socket.ResponseWriter, r *socket.Request) {
		w.Write([]byte("pong"))
	}
}

func (p *peerPool) handleGetPeers() socket.RPCHandler {
	return func(w socket.ResponseWriter, r *socket.Request) {
		peer, exist := p.getPeer(r.PeerID)
		if exist {
			val := p.rpcCounter.increment(peer.ID(), endpointGetPeers)
			if val > 1 {
				peer.applyPenalty(10)
			}
		}

		addresses := p.addressbook.GetAddresses(1000)
		newAddresses := &addressbook.Addresses{Addresses: addresses}
		encodedAddresses, err := newAddresses.Encode()
		if err != nil {
			p.logger.Errorf("Node info failed to be encoded with error %v", err)
			w.Error(err)
			return
		}
		w.Write(encodedAddresses)
	}
}

// broadcast information to all connected peers.
func (p *peerPool) broadcast(ctx context.Context, event string, data []byte) {
	peers := p.getConnectedPeers(false)
	for _, peer := range peers {
		peer.send(ctx, event, data)
	}
}

// send information to subset of peers.
func (p *peerPool) send(ctx context.Context, event string, data []byte) {
	sendList := p.selectForSend()
	for _, peer := range sendList {
		peer.send(ctx, event, data)
	}
}

func (p *peerPool) applyPenalty(peerID string, score int) {
	peer, exist := p.getPeer(peerID)
	if exist {
		peer.applyPenalty(score)
	}
}

func (p *peerPool) populate() {
	p.logger.Debug("Start populating the pool")
	// calculate insert not connected fixed peers
	notConnectedFixedAddresses := []*addressbook.Address{}
	fixedAddresses := p.addressbook.FixedAddresses()
	for _, fixedAddr := range fixedAddresses {
		if !p.connected(fixedAddr) {
			notConnectedFixedAddresses = append(notConnectedFixedAddresses, fixedAddr)
		}
	}
	// Enough connection exist
	openSlot := ints.Min(p.maxOutboundConnection-len(p.outboundPeers)-len(notConnectedFixedAddresses), p.addressbook.Size()+len(fixedAddresses))
	if openSlot <= 0 {
		// disconnect from seed peers
		p.logger.Debug("Populate: there are no open slot", p.maxOutboundConnection-len(p.outboundPeers), p.addressbook.Size())
		if err := p.disconnectSeedPeers(); err != nil {
			p.logger.Errorf("Fail to disconnect from seed peer with %v", err)
		}
		return
	}
	connectedIPs := make([]string, len(p.inboundPeers)+len(p.outboundPeers))
	index := 0
	for _, v := range p.inboundPeers {
		connectedIPs[index] = v.ip
		index++
	}
	for _, v := range p.outboundPeers {
		connectedIPs[index] = v.ip
		index++
	}

	p.logger.Debug("Populating for open slot", openSlot)
	addresses := p.addressbook.GetAddress(openSlot, connectedIPs)
	// Random address and fixed address
	addresses = append(addresses, fixedAddresses...)
	wg := new(sync.WaitGroup)
	wg.Add(len(addresses))
	for _, address := range addresses {
		go func(address *addressbook.Address) {
			defer wg.Done()
			p.logger.Debug("Connecting to ", address.IP())
			if _, err := p.createConnection(p.ctx, address); err != nil {
				p.logger.Errorf("Fail to create connection to %s with error %v", address.IP(), err)
			}
		}(address)
	}
	wg.Wait()
	p.logger.Debug("Finish populating the pool")
}

func (p *peerPool) discovery() {
	p.logger.Debug("Start discovery")
	openSlot := p.maxOutboundConnection - len(p.outboundPeers)
	if openSlot <= 0 {
		p.logger.Debug("Discovery: there are no open slot")
		return
	}
	seedAddresses := p.addressbook.SeedAddresses()
	if len(seedAddresses) == 0 {
		p.logger.Debug("Discovery: there are no seed addresses")
		return
	}
	connected := []*Peer{}
	notConnected := []*addressbook.Address{}
	for _, seed := range seedAddresses {
		seedPeer, exist := p.getOutboundPeer(getIDFromAddress(seed))
		if exist {
			connected = append(connected, seedPeer)
			continue
		}
		notConnected = append(notConnected, seed)
	}
	// out of connected peer, check
	for _, peer := range connected {
		addresses, err := peer.getPeers(p.ctx)
		if err != nil {
			p.logger.Debugf("Fail to get peers by %v", err)
		}
		for _, addr := range addresses {
			p.logger.Debug("Adding new address", addr.IP())
			if err := p.addressbook.AddNewAddress(addr); err != nil {
				p.logger.Errorf("Fail to add new address %s by %v", addr, err)
			}
		}
	}
	for _, addr := range notConnected {
		if _, err := p.createConnection(p.ctx, addr); err != nil {
			p.logger.Errorf("Fail to connect to new address %s by %v", addr, err)
		}
	}
}

func (p *peerPool) shuffle() {
	p.logger.Debug("Start shuffle")
	defer func() { p.logger.Debug("Finish shuffle") }()
	if len(p.outboundPeers) == 0 {
		return
	}
	candidates := []*Peer{}
	fixedPeerAddrs := p.addressbook.FixedAddresses()
	for _, peer := range p.getConnectedPeers(true) {
		isFixed := false
		for _, fixed := range fixedPeerAddrs {
			if peer.ip == fixed.IP() {
				isFixed = true
				break
			}
		}
		if !isFixed {
			candidates = append(candidates, peer)
		}
	}
	if len(candidates) == 0 {
		return
	}
	index := rand.Intn(len(candidates))
	closingPeer := candidates[index]
	p.disconnectAndRemove(closingPeer)
}

func (p *peerPool) createConnection(ctx context.Context, address *addressbook.Address) (bool, error) {
	p.logger.Debugf("Creating connection to %s", address.IP())
	connector := createConnector(ctx, address, p.nodeInfo)
	conn, err := connector()
	if err != nil {
		return false, err
	}
	peer, err := newOutboundPeer(ctx, address, conn, connector, p.rpcHandlers, p.logger)
	if err != nil {
		return false, err
	}
	go peer.bind()
	go p.bind(peer)
	if err := peer.updateNodeInfo(ctx); err != nil {
		return false, err
	}
	address.SetAdvertise(peer.nodeInfo.AdvertiseAddress)
	otherPeers, err := peer.getPeers(ctx)
	if err != nil {
		return false, err
	}
	for _, addr := range otherPeers {
		if err := p.addressbook.AddNewAddress(addr); err != nil {
			return false, err
		}
	}
	p.addOutboundPeer(peer)
	if err := p.addressbook.MoveNewAddressToTried(address); err != nil {
		return false, err
	}
	p.logger.Infof("Created connection to %s", peer.ID())
	return true, nil
}

func (p *peerPool) bind(peer *Peer) {
	for msg := range peer.subscribe() {
		switch msg.kind {
		case eventPeerClosed:
			p.disconnectAndRemove(peer)
		case eventPeerMessage:
			event := newEvent(msg.peerID, msg.event, msg.data)
			handler, exist := p.eventHandlers[msg.event]
			if !exist {
				p.logger.Warningf("Received unregistered event %s from %s. Applying penalty", msg.event, msg.peerID)
				peer.applyPenalty(100)
				continue
			}
			handler(event)
		case eventPeerBanned:
			p.disconnectAndRemove(peer)
			banningPeer := addressbook.NewAddress(peer.IP(), peer.Port())
			p.addressbook.BanAddress(banningPeer)
		default:
			p.logger.Errorf("No matching kind of message to %s", msg.kind)
		}
	}
}

func (p *peerPool) connected(addr *addressbook.Address) bool {
	p.mutex.RLocker().Lock()
	defer p.mutex.RLocker().Unlock()
	if _, exist := p.outboundPeers[getIDFromAddress(addr)]; exist {
		return true
	}
	if _, exist := p.inboundPeers[getIDFromAddress(addr)]; exist {
		return true
	}
	return false
}

func (p *peerPool) evictInbound() {
	inboundPeers := p.getInboundPeers()
	unprotectedPeers := []*Peer{}
	protectedAddresses := append(p.addressbook.FixedAddresses(), p.addressbook.WhitelistedAddresses()...)
	for _, peer := range inboundPeers {
		if !addressbook.AddressList(protectedAddresses).Include(peer.ip, int(peer.port)) {
			unprotectedPeers = append(unprotectedPeers, peer)
		}
	}
	netgroupProtected := filterByNetgroup(unprotectedPeers)
	latencyProtected := filterByLatency(unprotectedPeers)
	reposnseTimeProtected := filterByResponseRate(unprotectedPeers)
	// combine all protected peers
	allProtected := append(netgroupProtected, latencyProtected...) //nolint: gocritic // intending to combine 2 slices and create new slice
	allProtected = append(allProtected, reposnseTimeProtected...)
	// Update protected to be unique
	allProtected = Peers(allProtected).Unique()
	// additionally check for connection time
	connectionTimeProtected := filterByConnectionTime(allProtected)
	allProtected = Peers(allProtected).Filter(connectionTimeProtected)
	// remove all protected
	evictingPeers := Peers(unprotectedPeers).Filter(allProtected)
	if len(evictingPeers) == 0 {
		p.logger.Debug("eviction did not choose any address")
		return
	}
	index := rand.Intn(len(evictingPeers))
	p.disconnectAndRemove(evictingPeers[index])
}

func (p *peerPool) disconnectSeedPeers() error {
	seedAddresses := p.addressbook.SeedAddresses()
	for _, seed := range seedAddresses {
		if seedPeer, exist := p.getOutboundPeer(getIDFromAddress(seed)); exist {
			p.disconnectAndRemove(seedPeer)
		}
	}
	return nil
}

func (p *peerPool) addOutboundPeer(peer *Peer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.outboundPeers[peer.ID()] = peer
}

func (p *peerPool) addInboundPeer(peer *Peer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.inboundPeers[peer.ID()] = peer
}

func (p *peerPool) getInboundPeers() []*Peer {
	p.mutex.RLocker().Lock()
	defer p.mutex.RLocker().Unlock()
	peers := make([]*Peer, len(p.inboundPeers))
	index := 0
	for _, peer := range p.outboundPeers {
		peers[index] = peer
		index++
	}
	return peers
}

func (p *peerPool) getOutboundPeers() []*Peer {
	p.mutex.RLocker().Lock()
	defer p.mutex.RLocker().Unlock()
	peers := make([]*Peer, len(p.outboundPeers))
	index := 0
	for _, peer := range p.outboundPeers {
		peers[index] = peer
		index++
	}
	return peers
}

// getConnectedIDs returns a map with key as peerID if connected.
func (p *peerPool) getConnectedIDs() map[string]bool {
	p.mutex.RLocker().Lock()
	defer p.mutex.RLocker().Unlock()
	result := map[string]bool{}
	for _, peer := range p.outboundPeers {
		result[peer.ID()] = true
	}
	for _, peer := range p.inboundPeers {
		result[peer.ID()] = true
	}
	return result
}

func (p *peerPool) getOutboundPeer(peerID string) (*Peer, bool) {
	p.mutex.RLocker().Lock()
	defer p.mutex.RLocker().Unlock()
	peer, exist := p.outboundPeers[peerID]
	if exist {
		return peer, true
	}
	return nil, false
}

func (p *peerPool) getPeer(peerID string) (*Peer, bool) {
	p.mutex.RLocker().Lock()
	defer p.mutex.RLocker().Unlock()
	peer, exist := p.outboundPeers[peerID]
	if exist {
		return peer, true
	}
	peer, exist = p.inboundPeers[peerID]
	if exist {
		return peer, true
	}
	return nil, false
}

// addConnection adds a connection as incoming connection.
func (p *peerPool) getConnectedPeers(onlyOutbound bool) []*Peer {
	p.mutex.RLocker().Lock()
	defer p.mutex.RLocker().Unlock()
	result := []*Peer{}
	for _, peer := range p.outboundPeers {
		result = append(result, peer)
	}
	if onlyOutbound {
		return result
	}
	for _, peer := range p.inboundPeers {
		result = append(result, peer)
	}
	return result
}

func (p *peerPool) disconnectAndRemove(peer *Peer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	peer.close()
	delete(p.outboundPeers, peer.ID())
	delete(p.inboundPeers, peer.ID())
}

func (p *peerPool) selectForSend() []*Peer {
	p.mutex.RLocker().Lock()
	defer p.mutex.RLocker().Unlock()
	// check the size of the inbound connection max
	halfLimit := sendPeerLimit / 2
	// evict if it needs space
	outPeers := make([]*Peer, len(p.outboundPeers))
	index := 0
	for _, peer := range p.outboundPeers {
		outPeers[index] = peer
	}
	inPeers := make([]*Peer, len(p.inboundPeers))
	index = 0
	for _, peer := range p.inboundPeers {
		inPeers[index] = peer
	}
	if len(outPeers) == 0 && len(inPeers) == 0 {
		return []*Peer{}
	}
	var shortList []*Peer
	var longList []*Peer
	if len(outPeers) > len(inPeers) {
		shortList, longList = inPeers, outPeers
	} else {
		shortList, longList = outPeers, inPeers
	}
	shortHalfLimit := ints.Min(halfLimit, len(shortList))
	sendList := shortList[:shortHalfLimit]
	longHalfLimit := ints.Min(sendPeerLimit-len(sendList), len(longList))
	sendList = append(sendList, longList[:longHalfLimit]...)
	return sendList
}

func createConnector(ctx context.Context, address *addressbook.Address, nodeInfo *NodeInfo) func() (*websocket.Conn, error) {
	return func() (*websocket.Conn, error) {
		u := url.URL{
			Scheme:   "ws",
			Host:     fmt.Sprintf("%s:%d", address.IP(), address.Port()),
			Path:     "/rpc/",
			RawQuery: nodeInfo.Query(),
		}
		ctx, cancel := context.WithTimeout(ctx, time.Second*connectionTimeout)
		defer cancel()
		conn, _, err := websocket.DefaultDialer.DialContext(ctx, u.String(), nil)
		return conn, err
	}
}

func getIDFromAddress(address *addressbook.Address) string {
	return fmt.Sprintf("%s:%d", address.IP(), address.Port())
}

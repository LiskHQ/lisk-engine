package p2p

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/control"
	"github.com/libp2p/go-libp2p/core/network"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

const (
	MaxPenaltyScore = 100 // When a peer exceeded the MaxPenaltyScore, it should be blocked
)

var (
	errInvalidDuration       = errors.New("invalid duration")
	errConnGaterIsNotrunning = errors.New("failed to add new peer, start needs to be called first")
)

// peerInfo keeps information of each peer ID in the peerScore of the connectionGater.
type peerInfo struct {
	score      int
	expiration int64
}

func newPeerInfo(expiration int64, score int) *peerInfo {
	return &peerInfo{
		expiration: expiration,
		score:      score,
	}
}

// connectionGater extendis the BasicConnectionGater of the libp2p to use expire
// time to remove peer ID from peerScore.
type connectionGater struct {
	mutex *sync.RWMutex

	peerScore    map[PeerID]*peerInfo
	blockedAddrs map[string]struct{}

	logger        log.Logger
	expiration    time.Duration
	intervalCheck time.Duration
	isStarted     bool
}

// newConnGater returns a new connectionGater.
func newConnGater(logger log.Logger, expiration, itervalCheck time.Duration) (*connectionGater, error) {
	if expiration <= 0 || itervalCheck <= 0 {
		return nil, errInvalidDuration
	}

	return &connectionGater{
		mutex:  new(sync.RWMutex),
		logger: logger,

		peerScore:    make(map[PeerID]*peerInfo),
		blockedAddrs: make(map[string]struct{}),

		expiration:    expiration,
		intervalCheck: itervalCheck,
	}, nil
}

// addPenalty will update the score of given peer ID.
func (cg *connectionGater) addPenalty(pid PeerID, score int) (int, error) {
	if !cg.isStarted {
		return 0, errConnGaterIsNotrunning
	}

	cg.mutex.Lock()
	defer cg.mutex.Unlock()

	newScore := score
	if info, ok := cg.peerScore[pid]; ok {
		newScore = info.score + score
		info.score = newScore
	} else {
		cg.peerScore[pid] = newPeerInfo(0, newScore)
	}

	if newScore >= MaxPenaltyScore {
		exTime := time.Now().Unix() + int64(cg.expiration.Seconds())
		if info, ok := cg.peerScore[pid]; ok {
			info.expiration = exTime
		} else {
			cg.peerScore[pid] = newPeerInfo(exTime, 0)
		}
		return newScore, nil
	}

	return newScore, nil
}

// listBlockedPeers return a list of blocked peers.
func (cg *connectionGater) listBlockedPeers() []PeerID {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	result := make([]PeerID, 0, len(cg.peerScore))
	for p, info := range cg.peerScore {
		if info.expiration != 0 {
			result = append(result, p)
		}
	}

	return result
}

// start runs a new goroutine to check the expiration time based on
// intervaliCheck, it will be run automatically.
func (cg *connectionGater) start(ctx context.Context, wg *sync.WaitGroup) {
	if !cg.isStarted {
		t := time.NewTicker(cg.intervalCheck)
		cg.logger.Infof("ConnectionGater is started")
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-t.C:
					for p, info := range cg.peerScore {
						if time.Now().Unix() > info.expiration {
							cg.mutex.Lock()
							delete(cg.peerScore, p)
							cg.mutex.Unlock()
						}
					}
				case <-ctx.Done():
					return
				}
			}
		}()
		cg.isStarted = true
	}
}

// blockAddr adds an IP address to the set of blocked addresses.
// Note: active connections to the IP address are not automatically closed.
func (cg *connectionGater) blockAddr(ip net.IP) {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	cg.blockedAddrs[ip.String()] = struct{}{}
}

// unblockAddr removes an IP address from the set of blocked addresses.
func (cg *connectionGater) unblockAddr(ip net.IP) {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	delete(cg.blockedAddrs, ip.String())
}

// listBlockedAddrs return a list of blocked IP addresses.
func (cg *connectionGater) listBlockedAddrs() []net.IP {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	result := make([]net.IP, 0, len(cg.blockedAddrs))
	for ipStr := range cg.blockedAddrs {
		ip := net.ParseIP(ipStr)
		result = append(result, ip)
	}

	return result
}

// optionWithBlacklist returns the ConnectionGater option of the libp2p which
// is usable to reject incoming and outgoing connections based on blacklists.
// Locally verifying the blacklist in unit test is not feasible because it needs multiple IPs.
// It is tested in the network.
func (cg *connectionGater) optionWithBlacklist(bl []string) (libp2p.Option, error) {
	if len(bl) > 0 {
		blIPs := []net.IP{}
		for _, ipStr := range bl {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				return nil, fmt.Errorf("IP %s is invalid", ipStr)
			}
			blIPs = append(blIPs, ip)
		}

		for _, adr := range blIPs {
			cg.blockAddr(adr)
		}
	}

	return libp2p.ConnectionGater(cg), nil
}

// InterceptPeerDial tests whether we're permitted to Dial the specified peer.
//
// This is called by the network.Network implementation when dialling a peer.
func (cg *connectionGater) InterceptPeerDial(pid PeerID) (allow bool) {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	info, block := cg.peerScore[pid]
	return !(block && info.expiration != 0)
}

// InterceptAddrDial tests whether we're permitted to dial the specified
// multiaddr for the given peer.
//
// This is called by the network.Network implementation after it has
// resolved the peer's addrs, and prior to dialling each.
func (cg *connectionGater) InterceptAddrDial(pid PeerID, addr ma.Multiaddr) bool {
	// we have already filtered blocked peers in InterceptPeerDial, so we just check the IP
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	ip, err := manet.ToIP(addr)
	if err != nil {
		cg.logger.Warningf("error converting multiaddr to IP addr: %s", err)
		return true
	}

	_, block := cg.blockedAddrs[ip.String()]
	return !block
}

// InterceptAccept tests whether an incipient inbound connection is allowed.
//
// This is called by the upgrader, or by the transport directly (e.g. QUIC,
// Bluetooth), straight after it has accepted a connection from its socket.
func (cg *connectionGater) InterceptAccept(cma network.ConnMultiaddrs) bool {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	a := cma.RemoteMultiaddr()

	ip, err := manet.ToIP(a)
	if err != nil {
		cg.logger.Warningf("error converting multiaddr to IP addr: %s", err)
		return true
	}

	_, block := cg.blockedAddrs[ip.String()]
	return !block
}

// InterceptSecured tests whether a given connection, now authenticated,
// is allowed.
//
// This is called by the upgrader, after it has performed the security
// handshake, and before it negotiates the muxer, or by the directly by the
// transport, at the exact same checkpoint.
func (cg *connectionGater) InterceptSecured(dir network.Direction, p PeerID, cma network.ConnMultiaddrs) bool {
	if dir == network.DirOutbound {
		// we have already filtered those in InterceptPeerDial/InterceptAddrDial
		return true
	}

	// we have already filtered addrs in InterceptAccept, so we just check the peer ID
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	info, block := cg.peerScore[p]
	return !(block && info.expiration != 0)
}

// InterceptUpgraded tests whether a fully capable connection is allowed.
//
// At this point, the connection a multiplexer has been selected.
// When rejecting a connection, the gater can return a DisconnectReason.
// Refer to the godoc on the ConnectionGater type for more information.
//
// NOTE: the go-libp2p implementation currently IGNORES the disconnect reason.
func (cg *connectionGater) InterceptUpgraded(network.Conn) (allow bool, reason control.DisconnectReason) {
	return true, 0
}

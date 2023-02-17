package p2p

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/control"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

var (
	errInvalidDuration       = errors.New("the value of duration is invalid")
	errConnGaterIsNotrunning = errors.New("to be able add new peer, please call start function")
	maxScore                 = 100
)

type peerInfo struct {
	score      int
	expiration int64
}

// connectionGater extendis the BasicConnectionGater of the libp2p to use expire
// time to remove peer ID from blockedPeers.
type connectionGater struct {
	sync.RWMutex

	blockedPeers map[peer.ID]peerInfo
	blockedAddrs map[string]struct{}

	logger        log.Logger
	expiration    time.Duration
	intervalCheck time.Duration
	isStarted     bool
}

// newConnGater returns a new connectionGater.
func newConnGater(ex, iCheck time.Duration) (*connectionGater, error) {
	if ex <= 0 || iCheck <= 0 {
		return nil, errInvalidDuration
	}

	return &connectionGater{
		blockedPeers:  make(map[peer.ID]peerInfo),
		blockedAddrs:  make(map[string]struct{}),
		expiration:    ex,
		intervalCheck: iCheck,
	}, nil
}

// addPenalty will update the score of given peer ID.
func (cg *connectionGater) addPenalty(pid peer.ID, score int) error {
	if !cg.isStarted {
		return errConnGaterIsNotrunning
	}

	cg.Lock()
	if info, ok := cg.blockedPeers[pid]; ok {
		score = info.score + score
		cg.blockedPeers[pid] = peerInfo{
			score:      score,
			expiration: info.expiration,
		}
	} else {
		cg.blockedPeers[pid] = peerInfo{
			score: score,
		}
	}

	if score >= maxScore {
		cg.Unlock()
		return cg.blockPeer(pid)
	}

	cg.Unlock()
	return nil
}

// blockPeer blocks the given peer ID.
func (cg *connectionGater) blockPeer(pid peer.ID) error {
	if !cg.isStarted {
		return errConnGaterIsNotrunning
	}

	cg.Lock()
	defer cg.Unlock()
	exTime := time.Now().Unix() + int64(cg.expiration.Seconds())
	if info, ok := cg.blockedPeers[pid]; ok {
		info.expiration = exTime
		cg.blockedPeers[pid] = info
	} else {
		cg.blockedPeers[pid] = peerInfo{
			expiration: exTime,
		}
	}

	return nil
}

// listBlockedPeers return a list of blocked peers.
func (cg *connectionGater) listBlockedPeers() []peer.ID {
	cg.RLock()
	defer cg.RUnlock()

	result := make([]peer.ID, 0, len(cg.blockedPeers))
	for p, info := range cg.blockedPeers {
		if info.expiration != 0 {
			result = append(result, p)
		}
	}

	return result
}

// start runs a new goroutine to check the expiration time based on
// intervaliCheck, it will be run automatically.
func (cg *connectionGater) start(ctx context.Context) {
	if !cg.isStarted {
		t := time.NewTicker(cg.intervalCheck)
		go func() {
			for {
				select {
				case <-t.C:
					for p, info := range cg.blockedPeers {
						if time.Now().Unix() > info.expiration {
							cg.Lock()
							delete(cg.blockedPeers, p)
							cg.Unlock()
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
	cg.Lock()
	defer cg.Unlock()

	cg.blockedAddrs[ip.String()] = struct{}{}
}

// unblockAddr removes an IP address from the set of blocked addresses.
func (cg *connectionGater) unblockAddr(ip net.IP) {
	cg.Lock()
	defer cg.Unlock()

	delete(cg.blockedAddrs, ip.String())
}

// listBlockedAddrs return a list of blocked IP addresses.
func (cg *connectionGater) listBlockedAddrs() []net.IP {
	cg.RLock()
	defer cg.RUnlock()

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

// ConnectionGater interface.
var _ connmgr.ConnectionGater = (*conngater.BasicConnectionGater)(nil)

func (cg *connectionGater) InterceptPeerDial(p peer.ID) (allow bool) {
	cg.RLock()
	defer cg.RUnlock()

	info, block := cg.blockedPeers[p]
	return !(block && info.expiration != 0)
}

func (cg *connectionGater) InterceptAddrDial(p peer.ID, a ma.Multiaddr) (allow bool) {
	// we have already filtered blocked peers in InterceptPeerDial, so we just check the IP
	cg.RLock()
	defer cg.RUnlock()

	ip, err := manet.ToIP(a)
	if err != nil {
		cg.logger.Warningf("error converting multiaddr to IP addr: %s", err)
		return true
	}

	_, block := cg.blockedAddrs[ip.String()]
	return !block
}

func (cg *connectionGater) InterceptAccept(cma network.ConnMultiaddrs) (allow bool) {
	cg.RLock()
	defer cg.RUnlock()

	a := cma.RemoteMultiaddr()

	ip, err := manet.ToIP(a)
	if err != nil {
		cg.logger.Warningf("error converting multiaddr to IP addr: %s", err)
		return true
	}

	_, block := cg.blockedAddrs[ip.String()]
	return !block
}

func (cg *connectionGater) InterceptSecured(dir network.Direction, p peer.ID, cma network.ConnMultiaddrs) (allow bool) {
	if dir == network.DirOutbound {
		// we have already filtered those in InterceptPeerDial/InterceptAddrDial
		return true
	}

	// we have already filtered addrs in InterceptAccept, so we just check the peer ID
	cg.RLock()
	defer cg.RUnlock()

	info, block := cg.blockedPeers[p]
	return !(block && info.expiration != 0)
}

func (cg *connectionGater) InterceptUpgraded(network.Conn) (allow bool, reason control.DisconnectReason) {
	return true, 0
}

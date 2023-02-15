package p2p

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	collection "github.com/LiskHQ/lisk-engine/pkg/collection"
	log "github.com/LiskHQ/lisk-engine/pkg/log"
	lps "github.com/LiskHQ/lisk-engine/pkg/p2p/v2/pubsub"
)

const peerbookUpdateTimeout = time.Second * 5                      // Peerbook update timeout in nanoseconds.
const banTimeout int64 = int64((time.Hour * 24) / time.Nanosecond) // Ban timeout in seconds (24 hours).

// Peerbook keeps track of different lists of peers.
type Peerbook struct {
	logger         log.Logger
	mutex          sync.Mutex
	seedPeers      []*peer.AddrInfo
	fixedPeers     []*peer.AddrInfo
	blacklistedIPs []string
	bannedIPs      []*BannedIP
}

// BannedIP represents a banned IP and its timestamp.
type BannedIP struct {
	ip        string
	timestamp int64
}

// NewPeerbook returns a new Peerbook.
func NewPeerbook(seedPeers []string, fixedPeers []string, blacklistedIPs []string) (*Peerbook, error) {
	seedPeersAddrInfo := make([]*peer.AddrInfo, len(seedPeers))
	for i, seedPeer := range seedPeers {
		addrInfo, err := PeerInfoFromMultiAddr(seedPeer)
		if err != nil {
			return nil, err
		}
		seedPeersAddrInfo[i] = addrInfo
	}

	fixedPeersAddrInfo := make([]*peer.AddrInfo, len(fixedPeers))
	for i, fixedPeer := range fixedPeers {
		addrInfo, err := PeerInfoFromMultiAddr(fixedPeer)
		if err != nil {
			return nil, err
		}
		fixedPeersAddrInfo[i] = addrInfo
	}

	peerbook := &Peerbook{mutex: sync.Mutex{}, seedPeers: seedPeersAddrInfo, fixedPeers: fixedPeersAddrInfo, blacklistedIPs: blacklistedIPs, bannedIPs: []*BannedIP{}}
	return peerbook, nil
}

// init initializes the Peerbook.
func (pb *Peerbook) init(logger log.Logger) {
	pb.logger = logger

	for _, ip := range pb.blacklistedIPs {
		// Only warn if the blacklisted IP is present in seed peers or fixed peers.
		if pb.isIPInSeedPeers(ip) {
			pb.logger.Errorf("Blacklisted IP %s is present in seed peers", ip)
		}
		if pb.isIPInFixedPeers(ip) {
			pb.logger.Errorf("Blacklisted IP %s is present in fixed peers", ip)
		}
	}
}

// SeedPeers returns seed peers.
func (pb *Peerbook) SeedPeers() []*peer.AddrInfo {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	return pb.seedPeers
}

// FixedPeers returns fixed peers.
func (pb *Peerbook) FixedPeers() []*peer.AddrInfo {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	return pb.fixedPeers
}

// BlacklistedIPs returns blacklisted IPs.
func (pb *Peerbook) BlacklistedIPs() []string {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	return pb.blacklistedIPs
}

// BannedIPs returns banned peers.
func (pb *Peerbook) BannedIPs() []*BannedIP {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	return pb.bannedIPs
}

// BanIP bans an IP address.
func (pb *Peerbook) BanIP(ip string) {
	pb.mutex.Lock()
	index := collection.FindIndex(pb.bannedIPs, func(val *BannedIP) bool {
		return val.ip == ip
	})
	pb.mutex.Unlock()

	if index > -1 {
		pb.mutex.Lock()
		pb.bannedIPs[index].timestamp = time.Now().Unix()
		pb.mutex.Unlock()
		return
	}

	// Warn if the IP is present in seed peers or fixed peers and do not ban it.
	if pb.isIPInSeedPeers(ip) {
		pb.logger.Warningf("IP %s is present in seed peers, will not ban it", ip)
		return
	}
	if pb.isIPInFixedPeers(ip) {
		pb.logger.Warningf("IP %s is present in fixed peers, will not ban it", ip)
		return
	}

	pb.mutex.Lock()
	pb.bannedIPs = append(pb.bannedIPs, &BannedIP{ip: ip, timestamp: time.Now().Unix()})
	pb.mutex.Unlock()
}

// isIPInSeedPeers returns true if the IP is in the list of seed peers.
func (pb *Peerbook) isIPInSeedPeers(ip string) bool {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	for _, seedPeer := range pb.seedPeers {
		for _, addr := range seedPeer.Addrs {
			if ip == lps.ExtractIP(addr) {
				return true
			}
		}
	}

	return false
}

// isIPInFixedPeers returns true if the IP is in the list of fixed peers.
func (pb *Peerbook) isIPInFixedPeers(ip string) bool {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	for _, fixedPeer := range pb.fixedPeers {
		for _, addr := range fixedPeer.Addrs {
			if ip == lps.ExtractIP(addr) {
				return true
			}
		}
	}

	return false
}

// isIPBlacklisted returns true if the IP is blacklisted.
func (pb *Peerbook) isIPBlacklisted(ip string) bool {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	index := collection.FindIndex(pb.blacklistedIPs, func(val string) bool {
		return val == ip
	})

	return index != -1
}

// isInSeedPeers returns true if the peer is in the list of seed peers.
func (pb *Peerbook) isInSeedPeers(peerID peer.ID) bool {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	index := collection.FindIndex(pb.seedPeers, func(val *peer.AddrInfo) bool {
		return peer.ID(val.ID.String()) == peerID
	})

	return index != -1
}

// isInFixedPeers returns true if the peer is in the list of fixed peers.
func (pb *Peerbook) isInFixedPeers(peerID peer.ID) bool {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	index := collection.FindIndex(pb.fixedPeers, func(val *peer.AddrInfo) bool {
		return peer.ID(val.ID.String()) == peerID
	})

	return index != -1
}

// isIPBanned returns true if the IP is banned.
func (pb *Peerbook) isIPBanned(ip string) bool {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	index := collection.FindIndex(pb.bannedIPs, func(val *BannedIP) bool {
		return val.ip == ip
	})

	return index != -1
}

// removeIPFromBannedIPs removes an IP from the list of banned IPs.
func (pb *Peerbook) removeIPFromBannedIPs(ip string) {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	index := collection.FindIndex(pb.bannedIPs, func(val *BannedIP) bool {
		return val.ip == ip
	})

	if index != -1 {
		pb.bannedIPs = append(pb.bannedIPs[:index], pb.bannedIPs[index+1:]...)
	}
}

// peerBookService handles are related jobs (update known peers list, manage banning/unbanning peers) for peerbook on a predefined interval.
func (pb *Peerbook) peerBookService(ctx context.Context, wg *sync.WaitGroup, p *Peer) {
	defer wg.Done()
	pb.logger.Infof("Peerbook service started")

	// Set up a ticker to update the peerbook on a predefined interval.
	t := time.NewTicker(peerbookUpdateTimeout)

	for {
		select {
		case <-t.C:
			pb.logger.Debugf("List of connected peers: %v", p.ConnectedPeers())
			pb.logger.Debugf("List of known peers: %v", p.KnownPeers())

			pb.logger.Debugf("List of banned IPs: %v", pb.BannedIPs())
			for _, ip := range pb.BannedIPs() {
				if ip.timestamp+banTimeout < time.Now().Unix() {
					pb.removeIPFromBannedIPs(ip.ip)
				}
			}

			t.Reset(peerbookUpdateTimeout)
		case <-ctx.Done():
			pb.logger.Infof("Peerbook service stopped")
			return
		}
	}
}

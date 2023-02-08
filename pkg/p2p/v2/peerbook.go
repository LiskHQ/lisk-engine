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
	seedPeers      []peer.AddrInfo
	fixedPeers     []peer.AddrInfo
	blacklistedIPs []string
	bannedIPs      []BannedIP
	knownPeers     []peer.AddrInfo
}

// BannedIP represents a banned IP and its timestamp.
type BannedIP struct {
	ip        string
	timestamp int64
}

// NewPeerbook returns a new Peerbook.
func NewPeerbook(seedPeers []string, fixedPeers []string, blacklistedIPs []string, knownPeers []AddressInfo2) (*Peerbook, error) {
	seedPeersAddrInfo := make([]peer.AddrInfo, len(seedPeers))
	for i, seedPeer := range seedPeers {
		addrInfo, err := PeerInfoFromMultiAddr(seedPeer)
		if err != nil {
			return nil, err
		}
		seedPeersAddrInfo[i] = *addrInfo
	}

	fixedPeersAddrInfo := make([]peer.AddrInfo, len(fixedPeers))
	for i, fixedPeer := range fixedPeers {
		addrInfo, err := PeerInfoFromMultiAddr(fixedPeer)
		if err != nil {
			return nil, err
		}
		fixedPeersAddrInfo[i] = *addrInfo
	}

	knownPeersAddrInfo := make([]peer.AddrInfo, len(knownPeers))
	for i, knownPeer := range knownPeers {
		addrInfo := peer.AddrInfo{ID: knownPeer.ID, Addrs: knownPeer.Addrs}
		knownPeersAddrInfo[i] = addrInfo
	}

	peerbook := &Peerbook{mutex: sync.Mutex{}, seedPeers: seedPeersAddrInfo, fixedPeers: fixedPeersAddrInfo, blacklistedIPs: blacklistedIPs, bannedIPs: []BannedIP{}, knownPeers: knownPeersAddrInfo}
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
func (pb *Peerbook) SeedPeers() []peer.AddrInfo {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	return pb.seedPeers
}

// FixedPeers returns fixed peers.
func (pb *Peerbook) FixedPeers() []peer.AddrInfo {
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
func (pb *Peerbook) BannedIPs() []BannedIP {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	return pb.bannedIPs
}

// KnownPeers returns known peers.
func (pb *Peerbook) KnownPeers() []peer.AddrInfo {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	return pb.knownPeers
}

// BanIP bans an IP address.
func (pb *Peerbook) BanIP(ip string) {
	pb.mutex.Lock()
	index := collection.FindIndex(pb.bannedIPs, func(val BannedIP) bool {
		return val.ip == ip
	})
	pb.mutex.Unlock()

	if index == -1 {
		// Warn if the IP is present in seed peers or fixed peers and do not ban it.
		if pb.isIPInSeedPeers(ip) {
			pb.logger.Warningf("IP %s is present in seed peers, will not ban it", ip)
			return
		}
		if pb.isIPInFixedPeers(ip) {
			pb.logger.Warningf("IP %s is present in fixed peers, will not ban it", ip)
			return
		}

		// Remove a peer from known peers if it has the same IP address.
		for _, knownPeer := range pb.knownPeers {
			for _, addr := range knownPeer.Addrs {
				if ip == lps.ExtractIP(addr) {
					pb.removePeerFromKnownPeers(knownPeer.ID)
				}
			}
		}

		pb.bannedIPs = append(pb.bannedIPs, BannedIP{ip: ip, timestamp: time.Now().Unix()})
	} else {
		pb.bannedIPs[index].timestamp = time.Now().Unix()
	}
}

// addPeerToKnownPeers adds a peer to the list of known peers and saves it to non-volatile storage (database).
func (pb *Peerbook) addPeerToKnownPeers(newPeer peer.AddrInfo) {
	pb.mutex.Lock()
	index := collection.FindIndex(pb.knownPeers, func(val peer.AddrInfo) bool {
		return val.ID == newPeer.ID
	})
	pb.mutex.Unlock()

	if index == -1 {
		for _, addr := range newPeer.Addrs {
			ip := lps.ExtractIP(addr)

			// If the peer is in seed peers, we won't add it to the list of known peers.
			if pb.isInSeedPeers(newPeer.ID) {
				return
			}

			// If the peer is in fixed peers, we won't add it to the list of known peers.
			if pb.isInFixedPeers(newPeer.ID) {
				return
			}

			// If the peer has an IP address that is blacklisted, we won't add it to the list of known peers.
			if pb.isIPBlacklisted(ip) {
				return
			}

			// If the peer has an IP address that is banned, we won't add it to the list of known peers.
			if pb.isIPBanned(ip) {
				return
			}
		}

		pb.mutex.Lock()
		pb.knownPeers = append(pb.knownPeers, newPeer)
		pb.mutex.Unlock()
	}
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

	index := collection.FindIndex(pb.seedPeers, func(val peer.AddrInfo) bool {
		return peer.ID(val.ID.String()) == peerID
	})

	return index != -1
}

// isInFixedPeers returns true if the peer is in the list of fixed peers.
func (pb *Peerbook) isInFixedPeers(peerID peer.ID) bool {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	index := collection.FindIndex(pb.fixedPeers, func(val peer.AddrInfo) bool {
		return peer.ID(val.ID.String()) == peerID
	})

	return index != -1
}

// isIPBanned returns true if the IP is banned.
func (pb *Peerbook) isIPBanned(ip string) bool {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	index := collection.FindIndex(pb.bannedIPs, func(val BannedIP) bool {
		return val.ip == ip
	})

	return index != -1
}

// removeIPFromBannedIPs removes an IP from the list of banned IPs.
func (pb *Peerbook) removeIPFromBannedIPs(ip string) {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	index := collection.FindIndex(pb.bannedIPs, func(val BannedIP) bool {
		return val.ip == ip
	})

	if index != -1 {
		pb.bannedIPs = append(pb.bannedIPs[:index], pb.bannedIPs[index+1:]...)
	}
}

// removePeerFromKnownPeers removes a peer from the list of known peers and saves it to non-volatile storage (database).
func (pb *Peerbook) removePeerFromKnownPeers(peerID peer.ID) {
	pb.mutex.Lock()
	index := collection.FindIndex(pb.knownPeers, func(val peer.AddrInfo) bool {
		return val.ID == peerID
	})
	pb.mutex.Unlock()

	if index != -1 {
		pb.mutex.Lock()
		pb.knownPeers = append(pb.knownPeers[:index], pb.knownPeers[index+1:]...)
		pb.mutex.Unlock()
	}
}

// peerBookService handles are related jobs (update known peers list, save data to database) for peerbook on a predefined interval.
func (pb *Peerbook) peerBookService(ctx context.Context, wg *sync.WaitGroup, p *Peer) {
	defer wg.Done()
	pb.logger.Infof("Peerbook service started")

	// Set up a ticker to update the peerbook on a predefined interval.
	t := time.NewTicker(peerbookUpdateTimeout)

	for {
		select {
		case <-t.C:
			pb.logger.Debugf("List of connected peers: %v", p.ConnectedPeers())
			pb.logger.Debugf("List of known peers: %v", pb.KnownPeers())

			for _, connPeer := range p.ConnectedPeers() {
				pb.addPeerToKnownPeers(p.host.Peerstore().PeerInfo(connPeer))
			}

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

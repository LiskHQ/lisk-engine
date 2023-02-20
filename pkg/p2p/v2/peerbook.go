package p2p

import (
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"

	collection "github.com/LiskHQ/lisk-engine/pkg/collection"
	log "github.com/LiskHQ/lisk-engine/pkg/log"
	lps "github.com/LiskHQ/lisk-engine/pkg/p2p/v2/pubsub"
)

// Peerbook keeps track of different lists of peers.
type Peerbook struct {
	logger                    log.Logger
	mutex                     sync.Mutex
	seedPeers                 []*peer.AddrInfo
	fixedPeers                []*peer.AddrInfo
	permanentlyBlacklistedIPs []string
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

	peerbook := &Peerbook{mutex: sync.Mutex{}, seedPeers: seedPeersAddrInfo, fixedPeers: fixedPeersAddrInfo, permanentlyBlacklistedIPs: blacklistedIPs}
	return peerbook, nil
}

// init initializes the Peerbook.
func (pb *Peerbook) init(logger log.Logger) {
	pb.logger = logger

	for _, ip := range pb.permanentlyBlacklistedIPs {
		// Only warn if the permanently blacklisted IP is present in seed peers or fixed peers.
		if pb.isIPInSeedPeers(ip) {
			pb.logger.Errorf("Permanently blacklisted IP %s is present in seed peers", ip)
		}
		if pb.isIPInFixedPeers(ip) {
			pb.logger.Errorf("Permanently blacklisted IP %s is present in fixed peers", ip)
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

// PermanentlyBlacklistedIPs returns permanently blacklisted IPs.
func (pb *Peerbook) PermanentlyBlacklistedIPs() []string {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()
	return pb.permanentlyBlacklistedIPs
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

// isIPPermanentlyBlacklisted returns true if the IP is permanently blacklisted.
func (pb *Peerbook) isIPPermanentlyBlacklisted(ip string) bool {
	pb.mutex.Lock()
	defer pb.mutex.Unlock()

	index := collection.FindIndex(pb.permanentlyBlacklistedIPs, func(val string) bool {
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

package p2p

import (
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

// PeerScore keeps a map based on peer ID and its score
type PeerScore struct {
	peers map[peer.ID]int
	mutex sync.Mutex
}

// NewPeerSocre returns a new PeerScore
func NewPeerScore() *PeerScore {
	return &PeerScore{
		peers: make(map[peer.ID]int),
		mutex: sync.Mutex{},
	}
}

// addPenalty will update the score of given peer ID
func (ps *PeerScore) addPenalty(pid peer.ID, score int) int {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	if peer, ok := ps.peers[pid]; ok {
		peer += score
		return peer
	}

	ps.peers[pid] = score
	return ps.peers[pid]
}

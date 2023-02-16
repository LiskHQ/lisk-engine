package p2p

import (
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
)

// peerScore keeps a map based on peer ID and its score.
type peerScore struct {
	peers map[peer.ID]int
	mutex sync.Mutex
}

// newPeerScore returns a new peerScore.
func newPeerScore() *peerScore {
	return &peerScore{
		peers: make(map[peer.ID]int),
		mutex: sync.Mutex{},
	}
}

// addPenalty will update the score of given peer ID.
func (ps *peerScore) addPenalty(pid peer.ID, score int) int {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	if peer, ok := ps.peers[pid]; ok {
		peer += score
		return peer
	}

	ps.peers[pid] = score
	return ps.peers[pid]
}

// deletePeer removes the given peer ID from Peers.
func (ps *peerScore) deletePeer(pid peer.ID) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	delete(ps.peers, pid)
}

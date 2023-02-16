package p2p

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

func Test_PeerScore(t *testing.T) {
	assert := assert.New(t)

	pidA := peer.ID("A")
	pidB := peer.ID("B")
	ps := newPeerScore()
	socre := ps.addPenalty(pidA, 0)
	assert.Equal(socre, 0)
	ps.addPenalty(pidB, 0)
	socre = ps.addPenalty(pidA, 10)
	assert.Equal(socre, 10)

	assert.Equal(len(ps.peers), 2)
	ps.deletePeer(pidB)
	assert.Equal(len(ps.peers), 1)
}

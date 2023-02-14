package p2p

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

func Test_PeerScore(t *testing.T) {
	assert := assert.New(t)

	pid := peer.ID("A")
	ps := NewPeerScore()
	socre := ps.addPenalty(pid, 0)
	assert.Equal(socre, 0)
	socre = ps.addPenalty(pid, 10)
	assert.Equal(socre, 10)
}

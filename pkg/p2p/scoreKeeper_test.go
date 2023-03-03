package p2p

import (
	"testing"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

func TestScoreKeeper(t *testing.T) {
	assert := assert.New(t)

	sk := newScoreKeeper()
	oldSK := sk.Get()
	newSK := make(map[peer.ID]*pubsub.PeerScoreSnapshot)
	assert.Equal(oldSK, newSK)

	oldSK = sk.Get()
	newSK["test"] = &pubsub.PeerScoreSnapshot{}
	assert.NotEqual(oldSK, newSK)
	sk.Update(newSK)
	oldSK = sk.Get()
	assert.Equal(oldSK, newSK)
}

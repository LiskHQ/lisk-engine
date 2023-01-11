package pubsub

import (
	"testing"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	peer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

func TestScoreKeeper(t *testing.T) {
	sk := NewScoreKeeper()
	oldSK := sk.Get()
	newSK := make(map[peer.ID]*pubsub.PeerScoreSnapshot)
	assert.Equal(t, oldSK, newSK)

	oldSK = sk.Get()
	newSK["test"] = &pubsub.PeerScoreSnapshot{}
	assert.NotEqual(t, oldSK, newSK)
	sk.Update(newSK)
	oldSK = sk.Get()
	assert.Equal(t, oldSK, newSK)
}

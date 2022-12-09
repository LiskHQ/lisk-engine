package p2p

import (
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	peer "github.com/libp2p/go-libp2p/core/peer"
)

type ScoreKeeper struct {
	lk     sync.RWMutex
	scores map[peer.ID]*pubsub.PeerScoreSnapshot
}

func NewScoreKeeper() *ScoreKeeper {
	return &ScoreKeeper{
		scores: make(map[peer.ID]*pubsub.PeerScoreSnapshot),
	}
}

func (sk *ScoreKeeper) Update(scores map[peer.ID]*pubsub.PeerScoreSnapshot) {
	sk.scores = scores
}

func (sk *ScoreKeeper) Get() map[peer.ID]*pubsub.PeerScoreSnapshot {
	sk.lk.RLock()
	defer sk.lk.RUnlock()
	return sk.scores
}

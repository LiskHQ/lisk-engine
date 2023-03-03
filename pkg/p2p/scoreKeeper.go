package p2p

import (
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

type scoreKeeper struct {
	lk     sync.Mutex
	scores map[peer.ID]*pubsub.PeerScoreSnapshot
}

func newScoreKeeper() *scoreKeeper {
	return &scoreKeeper{
		scores: make(map[peer.ID]*pubsub.PeerScoreSnapshot),
	}
}

func (sk *scoreKeeper) Update(scores map[peer.ID]*pubsub.PeerScoreSnapshot) {
	sk.lk.Lock()
	defer sk.lk.Unlock()
	sk.scores = scores
}

func (sk *scoreKeeper) Get() map[peer.ID]*pubsub.PeerScoreSnapshot {
	sk.lk.Lock()
	defer sk.lk.Unlock()
	return sk.scores
}

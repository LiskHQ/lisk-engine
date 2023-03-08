package p2p

import (
	"context"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

type Discovery struct {
	peer *Peer
}

func (d Discovery) Advertise(ctx context.Context, ns string, opts ...discovery.Option) (time.Duration, error) {
	// We currently don't advertise anything.
	return time.Hour * 24, nil // Set next advertisement to 24 hours from now.
}

func (d Discovery) FindPeers(ctx context.Context, ns string, opts ...discovery.Option) (<-chan peer.AddrInfo, error) {
	ch := make(chan peer.AddrInfo, 1)

	go func() {
		defer close(ch)

		peers := d.peer.knownPeers()

		// Shuffle known peers to avoid always returning the same peers.
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(peers), func(i, j int) { peers[i], peers[j] = peers[j], peers[i] })

		numPeers := len(peers)

		// Skip providing unnecessary peers.
		for i := 0; i < numPeers; i++ {
			// Skip ourselves.
			if peers[i].ID == d.peer.host.ID() {
				numPeers--
				continue
			}
			// Skip peers we are already connected to.
			if d.peer.host.Network().Connectedness(peers[i].ID) == network.Connected {
				numPeers--
				continue
			}
			break
		}

		if numPeers == 0 {
			return
		}

		for {
			select {
			case ch <- peers[numPeers-1]:
				numPeers--

				// Skip providing unnecessary peers.
				for i := 0; i < numPeers; i++ {
					// Skip ourselves.
					if peers[i].ID == d.peer.host.ID() {
						numPeers--
						continue
					}
					// Skip peers we are already connected to.
					if d.peer.host.Network().Connectedness(peers[i].ID) == network.Connected {
						numPeers--
						continue
					}
					break
				}

				if numPeers == 0 {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch, nil
}

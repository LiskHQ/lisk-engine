package p2p

import (
	"math"
	"sort"
)

const (
	protectionForNetgroup   = 0.034
	protectionForLatency    = 0.068
	protectionForUsefulness = 0.068
	protectionForLongevity  = 0.5
)

func filterByNetgroup(addresses []*Peer) []*Peer {
	if len(addresses) <= 1 {
		return addresses
	}
	copied := addresses
	numberOfProtectedPeer := math.Ceil(float64(len(addresses)) * protectionForNetgroup)
	sort.Slice(copied, func(i, j int) bool {
		return copied[i].netgroup > copied[j].netgroup
	})

	return copied[:int(numberOfProtectedPeer)]
}

func filterByLatency(addresses []*Peer) []*Peer {
	if len(addresses) <= 1 {
		return addresses
	}
	copied := addresses
	numberOfProtectedPeer := math.Ceil(float64(len(addresses)) * protectionForLatency)
	sort.Slice(copied, func(i, j int) bool {
		return copied[i].latency < copied[j].latency
	})

	return copied[:int(numberOfProtectedPeer)]
}

func filterByResponseRate(addresses []*Peer) []*Peer {
	if len(addresses) <= 1 {
		return addresses
	}
	copied := addresses
	numberOfProtectedPeer := math.Ceil(float64(len(addresses)) * protectionForUsefulness)
	sort.Slice(copied, func(i, j int) bool {
		return copied[i].responseRate() > copied[j].responseRate()
	})

	return copied[:int(numberOfProtectedPeer)]
}

func filterByConnectionTime(addresses []*Peer) []*Peer {
	if len(addresses) <= 1 {
		return addresses
	}
	copied := addresses
	numberOfProtectedPeer := math.Ceil(float64(len(addresses)) * protectionForLongevity)
	sort.Slice(copied, func(i, j int) bool {
		return copied[i].connectionTime < copied[j].connectionTime
	})

	return copied[:int(numberOfProtectedPeer)]
}

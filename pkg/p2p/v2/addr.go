package p2p

import (
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// PeerInfoFromMultiAddr returns a peer info from multi address as string.
func PeerInfoFromMultiAddr(s string) (*peer.AddrInfo, error) {
	addr, err := ma.NewMultiaddr(s)
	if err != nil {
		return &peer.AddrInfo{}, err
	}
	return peer.AddrInfoFromP2pAddr(addr)
}

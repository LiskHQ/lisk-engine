package p2p

import (
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// AddrInfoFromMultiAddr returns a peer info from multi address as string.
func AddrInfoFromMultiAddr(s string) (*AddrInfo, error) {
	addr, err := ma.NewMultiaddr(s)
	if err != nil {
		return &peer.AddrInfo{}, err
	}
	return peer.AddrInfoFromP2pAddr(addr)
}

// ParseAddresses returns an array of AddrInfo based on the array of the string.
func parseAddresses(addrs []string) ([]peer.AddrInfo, error) {
	var maddrs []ma.Multiaddr
	for _, addr := range addrs {
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return nil, err
		}

		if _, last := ma.SplitLast(maddr); last.Protocol().Code == ma.P_P2P {
			maddrs = append(maddrs, maddr)
		}
	}

	return peer.AddrInfosFromP2pAddrs(maddrs...)
}

// ExtractIP returns the IP address from the multiaddress.
func extractIP(address ma.Multiaddr) string {
	component, _ := ma.SplitFirst(address)
	return component.Value()
}

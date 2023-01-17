package p2p

import (
	"fmt"
	"net"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"
)

// ConnectionGaterOption returns the ConnectionGater option of the libp2p which
// is usable to reject incoming and outgoing connections based on blacklists.
// Locally verifying the blacklist in unit test is not feasible because it needs multiple IPs.
// It is tested in the network.
func ConnectionGaterOption(bl []string) (libp2p.Option, error) {
	blIPs := []net.IP{}
	for _, ipStr := range bl {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return nil, fmt.Errorf("IP %s is invalid", ipStr)
		}
		blIPs = append(blIPs, ip)
	}

	cg, err := conngater.NewBasicConnectionGater(nil)
	if err != nil {
		return nil, err
	}
	for _, adr := range blIPs {
		err := cg.BlockAddr(adr)
		if err != nil {
			return nil, err
		}
	}

	return libp2p.ConnectionGater(cg), nil
}

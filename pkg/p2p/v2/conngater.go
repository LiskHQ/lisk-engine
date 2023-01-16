package p2p

import (
	"net"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"
)

// Blacklists contains three different types of blocking which are used in ConnectionGater.
type Blacklists struct {
	peers   []PeerID
	addrs   []net.IP
	subnets []*net.IPNet
}

// NewBlackLisrs returns Blacklists based on three input arguments.
func NewBlacklists(peers []PeerID, addrs []net.IP, subnets []*net.IPNet) Blacklists {
	return Blacklists{
		peers:   peers,
		addrs:   addrs,
		subnets: subnets,
	}
}

func NewBlacklistWithPeer(peers []PeerID) Blacklists {
	return Blacklists{
		peers: peers,
	}
}

func NewBlacklistWithAddress(addrs []net.IP) Blacklists {
	return Blacklists{
		addrs: addrs,
	}
}

func NewBlacklistWithSubnet(subnets []*net.IPNet) Blacklists {
	return Blacklists{
		subnets: subnets,
	}
}

// ConnectionGaterOption returns the ConnectionGater option of the libp2p which
// is usable to reject incoming and outgoing connections based on blacklists.
func ConnectionGaterOption(bl Blacklists) (libp2p.Option, error) {
	cg, err := conngater.NewBasicConnectionGater(nil)
	if err != nil {
		return nil, err
	}
	for _, id := range bl.peers {
		err := cg.BlockPeer(peer.ID(id))
		if err != nil {
			return nil, err
		}
	}
	for _, adr := range bl.addrs {
		err := cg.BlockAddr(adr)
		if err != nil {
			return nil, err
		}
	}
	for _, sub := range bl.subnets {
		err := cg.BlockSubnet(sub)
		if err != nil {
			return nil, err
		}
	}

	return libp2p.ConnectionGater(cg), nil
}

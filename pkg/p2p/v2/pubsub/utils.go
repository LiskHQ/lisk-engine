package pubsub

import (
	"context"
	"fmt"

	pubsub_pb "github.com/libp2p/go-libp2p-pubsub/pb"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

// MessageTopic returns the network pubsub topic.
func MessageTopic(networkName string) string {
	return fmt.Sprintf("/lsk/%s", networkName)
}

func HashMsgID(m *pubsub_pb.Message) string {
	hash := crypto.Hash(m.Data)
	return string(hash)
}

// ParseAddresses returns an array of AddrInfo based on the array of the string.
func ParseAddresses(ctx context.Context, addrs []string) ([]peer.AddrInfo, error) {
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

// P2pAddrs returns all listen address of GossipSub based on host.
func P2pAddrs(h host.Host) ([]ma.Multiaddr, error) {
	peerInfo := peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}

	return peer.AddrInfoToP2pAddrs(&peerInfo)
}

package p2p

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	"github.com/stretchr/testify/assert"
)

func TestBlacklistPeer(t *testing.T) {
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := []libp2p.Option{}
	adr := []string{"/ip4/127.0.0.1/tcp/0"}
	opts = append(opts, libp2p.ListenAddrStrings(adr...))

	blockedHost, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	assert.Nil(err)
	acceptedHost, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	assert.Nil(err)

	var blockedPeers []PeerID
	for _, id := range blockedHost.Peerstore().Peers() {
		blockedPeers = append(blockedPeers, PeerID(id))
	}
	bl := NewBlacklistWithPeer(blockedPeers)
	conngr, err := ConnectionGaterOption(bl)
	assert.Nil(err)
	opts = append(opts, conngr)
	host, err := libp2p.New(opts...)
	assert.Nil(err)

	hostPeer := host.Peerstore().PeerInfo(host.Peerstore().Peers()[0])
	blockedPeer := blockedHost.Peerstore().PeerInfo(blockedHost.Peerstore().Peers()[0])
	assert.ErrorContains(host.Connect(ctx, blockedPeer), swarm.ErrGaterDisallowedConnection.Error())
	assert.ErrorContains(blockedHost.Connect(ctx, hostPeer), swarm.ErrNoAddresses.Error())

	acceptedPeer := acceptedHost.Peerstore().PeerInfo(acceptedHost.Peerstore().Peers()[0])
	assert.Nil(host.Connect(ctx, acceptedPeer))
	assert.Nil(acceptedHost.Connect(ctx, hostPeer))
}
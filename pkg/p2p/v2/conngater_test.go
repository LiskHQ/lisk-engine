package p2p

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"

	ma "github.com/multiformats/go-multiaddr"
)

func TestConnGater_BlacklistIPs(t *testing.T) {
	assert := assert.New(t)

	cg, err := newConnGater(10, 5)
	assert.Nil(err)

	containsInvalidIPs := []string{
		"128.20.12.11",
		"1.222.2222.12.12",
		"12.30.28.110",
	}
	_, err = cg.optionWithBlacklist(containsInvalidIPs)
	assert.ErrorContains(err, "is invalid")

	validIPs := []string{
		"127.0.0.1",
		"192.168.2.30",
	}
	_, err = cg.optionWithBlacklist(validIPs)
	assert.Nil(err)
	has := func(netIP net.IP) bool {
		for _, ip := range validIPs {
			if ip == netIP.String() {
				return true
			}
		}
		return false
	}

	for _, ip := range cg.listBlockedAddrs() {
		assert.True(has(ip))
	}
}

func TestConnGater_Errors(t *testing.T) {
	assert := assert.New(t)

	_, err := newConnGater(0, 0)
	assert.Equal(errInvalidDuration, err)

	_, err = newConnGater(10, -1)
	assert.Equal(errInvalidDuration, err)

	cg, err := newConnGater(1, 1)
	assert.Nil(err)

	assert.Equal(errConnGaterIsNotrunning, cg.blockPeer(peer.ID("A")))
}

func TestConnGater_ExpireTime(t *testing.T) {
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cg, err := newConnGater(time.Second*4, time.Second*2)
	assert.Nil(err)
	cg.start(ctx)

	pidA := peer.ID("A")
	assert.Nil(cg.blockPeer(pidA))

	time.Sleep(time.Second)
	pidB := peer.ID("B")
	cg.blockPeer(pidB)
	pidC := peer.ID("C")
	cg.blockPeer(pidC)

	assert.Equal(3, len(cg.listBlockedPeers()))
	time.Sleep(time.Second)
	pidD := peer.ID("D")
	cg.blockPeer(pidD)

	time.Sleep(time.Second * 3)
	assert.Equal(4, len(cg.listBlockedPeers()))
	time.Sleep(time.Second * 2)
	assert.Equal(1, len(cg.listBlockedPeers()))
	assert.Equal(pidD, cg.listBlockedPeers()[0])
	time.Sleep(time.Second)
	assert.Equal(0, len(cg.listBlockedPeers()))
}

func TestConnectionGater(t *testing.T) {
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	peerA := peer.ID("A")
	peerB := peer.ID("B")

	cg, err := newConnGater(time.Second*2, time.Second*1)
	assert.Nil(err)
	cg.start(ctx)

	// test peer blocking
	assert.Truef(cg.InterceptPeerDial(peerA), "expected gater to allow peerA")
	assert.Truef(cg.InterceptPeerDial(peerB), "expected gater to allow peerB")

	allow := cg.InterceptSecured(network.DirInbound, peerA, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.Truef(allow, "expected gater to allow peerA")

	allow = cg.InterceptSecured(network.DirInbound, peerB, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.Truef(allow, "expected gater to allow peerB")

	assert.Nil(cg.blockPeer(peerA))
	assert.Falsef(cg.InterceptPeerDial(peerA), "expected gater to deny peerA")
	assert.Truef(cg.InterceptPeerDial(peerB), "expected gater to allow peerB")

	allow = cg.InterceptSecured(network.DirInbound, peerA, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.Falsef(allow, "expected gater to allow peerA")

	allow = cg.InterceptSecured(network.DirInbound, peerB, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.Truef(allow, "expected gater to allow peerB")

	ip1 := net.ParseIP("1.2.3.4")
	cg.blockAddr(ip1)

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/1.2.3.4/tcp/1234"))
	assert.Falsef(allow, "expected gater to deny peerB in 1.2.3.4")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/1.2.3.4/tcp/1234")})
	assert.Falsef(allow, " expected gater to deny peerB in 1.2.3.4")

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/1.2.3.5/tcp/1234"))
	assert.Truef(allow, "expected gater to allow peerB in 1.2.3.5")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/1.2.3.5/tcp/1234")})
	assert.Truef(allow, "expected gater to allow peerB in 1.2.3.5")

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/2.3.4.5/tcp/1234"))
	assert.Truef(allow, "expected gater to allow peerB in 2.3.4.5")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/2.3.4.5/tcp/1234")})
	assert.Truef(allow, "expected gater to allow peerB in 2.3.4.5")

	// undo the blocks to ensure that we can unblock stuff
	cg.unblockAddr(ip1)
	// peers should be remove after expire time
	time.Sleep(time.Millisecond * 3100)
	assert.Equal(0, len(cg.listBlockedPeers()))

	assert.Truef(cg.InterceptPeerDial(peerA), "expected gater to allow peerA")
	assert.Truef(cg.InterceptPeerDial(peerB), "expected gater to allow peerB")

	allow = cg.InterceptSecured(network.DirInbound, peerA, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.Truef(allow, "expected gater to allow peerA")

	allow = cg.InterceptSecured(network.DirInbound, peerB, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.Truef(allow, "expected gater to allow peerB")

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/1.2.3.4/tcp/1234"))
	assert.Truef(allow, "expected gater to allow peerB in 1.2.3.4")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/1.2.3.4/tcp/1234")})
	assert.Truef(allow, "expected gater to allow peerB in 1.2.3.4")

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/1.2.3.5/tcp/1234"))
	assert.Truef(allow, "expected gater to allow peerB in 1.2.3.5")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/1.2.3.5/tcp/1234")})
	assert.Truef(allow, "expected gater to allow peerB in 1.2.3.5")

	allow = cg.InterceptAddrDial(peerB, ma.StringCast("/ip4/2.3.4.5/tcp/1234"))
	assert.Truef(allow, "expected gater to allow peerB in 1.2.3.5")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/2.3.4.5/tcp/1234")})
	assert.Truef(allow, "expected gater to allow peerB in 1.2.3.5")
}

type mockConnMultiaddrs struct {
	local, remote ma.Multiaddr
}

func (cma *mockConnMultiaddrs) LocalMultiaddr() ma.Multiaddr {
	return cma.local
}

func (cma *mockConnMultiaddrs) RemoteMultiaddr() ma.Multiaddr {
	return cma.remote
}

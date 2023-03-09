package p2p

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

func TestConnGater_BlacklistIPs(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()

	cg, err := newConnGater(logger, 10, 5)
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

	logger, _ := log.NewDefaultProductionLogger()

	_, err := newConnGater(logger, 0, 0)
	assert.Equal(errInvalidDuration, err)

	_, err = newConnGater(logger, 10, -1)
	assert.Equal(errInvalidDuration, err)

	cg, err := newConnGater(logger, 1, 1)
	assert.Nil(err)

	_, err = cg.addPenalty(peer.ID("A"), MaxPenaltyScore)
	assert.Equal(errConnGaterIsNotrunning, err)
}

func TestConnGater_ExpireTime(t *testing.T) {
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cg, err := newConnGater(logger, time.Second*4, time.Second*2)
	assert.Nil(err)
	wg := &sync.WaitGroup{}
	cg.start(ctx, wg)

	pidA := peer.ID("A")
	_, err = cg.addPenalty(pidA, MaxPenaltyScore)
	assert.Nil(err)

	time.Sleep(time.Second)
	pidB := peer.ID("B")
	cg.addPenalty(pidB, MaxPenaltyScore)
	pidC := peer.ID("C")
	cg.addPenalty(pidC, MaxPenaltyScore)

	assert.Equal(3, len(cg.listBlockedPeers()))
	time.Sleep(time.Second)
	pidD := peer.ID("D")
	cg.addPenalty(pidD, MaxPenaltyScore)

	waitForTestCondition(t, func() int { return len(cg.listBlockedPeers()) }, 4)
	assert.Equal(4, len(cg.listBlockedPeers()))

	waitForTestCondition(t, func() int { return len(cg.listBlockedPeers()) }, 1)
	assert.Equal(1, len(cg.listBlockedPeers()))
	assert.Equal(pidD, cg.listBlockedPeers()[0])

	waitForTestCondition(t, func() int { return len(cg.listBlockedPeers()) }, 0)
	assert.Equal(0, len(cg.listBlockedPeers()))
}

func TestConneGater(t *testing.T) {
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cg, err := newConnGater(logger, time.Second*2, time.Second*1)
	assert.Nil(err)
	wg := &sync.WaitGroup{}
	cg.start(ctx, wg)

	pidA := peer.ID("A")
	pidB := peer.ID("B")

	// test peer blocking
	assert.Truef(cg.InterceptPeerDial(pidA), "expected gater to allow peer A")
	assert.Truef(cg.InterceptPeerDial(pidB), "expected gater to allow peer B")

	allow := cg.InterceptSecured(network.DirInbound, pidA, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.Truef(allow, "expected gater to allow peer A")

	allow = cg.InterceptSecured(network.DirInbound, pidB, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.Truef(allow, "expected gater to allow peer B")

	_, err = cg.addPenalty(pidA, MaxPenaltyScore)
	assert.Nil(err)
	assert.Falsef(cg.InterceptPeerDial(pidA), "expected gater to deny peer A")
	assert.Truef(cg.InterceptPeerDial(pidB), "expected gater to allow peer B")

	allow = cg.InterceptSecured(network.DirInbound, pidA, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.Falsef(allow, "expected gater to deny peer A")

	allow = cg.InterceptSecured(network.DirInbound, pidB, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.Truef(allow, "expected gater to allow peer B")

	ip1 := net.ParseIP("1.2.3.4")
	cg.blockAddr(ip1)

	allow = cg.InterceptAddrDial(pidB, ma.StringCast("/ip4/1.2.3.4/tcp/1234"))
	assert.Falsef(allow, "expected gater to deny peer B in 1.2.3.4")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/1.2.3.4/tcp/1234")})
	assert.Falsef(allow, " expected gater to deny peer B in 1.2.3.4")

	allow = cg.InterceptAddrDial(pidB, ma.StringCast("/ip4/1.2.3.5/tcp/1234"))
	assert.Truef(allow, "expected gater to allow peer B in 1.2.3.5")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/1.2.3.5/tcp/1234")})
	assert.Truef(allow, "expected gater to allow peer B in 1.2.3.5")

	allow = cg.InterceptAddrDial(pidB, ma.StringCast("/ip4/2.3.4.5/tcp/1234"))
	assert.Truef(allow, "expected gater to allow peer B in 2.3.4.5")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/2.3.4.5/tcp/1234")})
	assert.Truef(allow, "expected gater to allow peer B in 2.3.4.5")

	// undo the blocks to ensure that we can unblock stuff
	cg.unblockAddr(ip1)
	// peers should be remove after expire time
	waitForTestCondition(t, func() int { return len(cg.listBlockedPeers()) }, 0)
	assert.Equal(0, len(cg.listBlockedPeers()))

	assert.Truef(cg.InterceptPeerDial(pidA), "expected gater to allow peer A")
	assert.Truef(cg.InterceptPeerDial(pidB), "expected gater to allow peer B")

	allow = cg.InterceptSecured(network.DirInbound, pidA, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.Truef(allow, "expected gater to allow peer A")

	allow = cg.InterceptSecured(network.DirInbound, pidB, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.Truef(allow, "expected gater to allow peer B")

	allow = cg.InterceptAddrDial(pidB, ma.StringCast("/ip4/1.2.3.4/tcp/1234"))
	assert.Truef(allow, "expected gater to allow peer B in 1.2.3.4")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/1.2.3.4/tcp/1234")})
	assert.Truef(allow, "expected gater to allow peer B in 1.2.3.4")

	allow = cg.InterceptAddrDial(pidB, ma.StringCast("/ip4/1.2.3.5/tcp/1234"))
	assert.Truef(allow, "expected gater to allow peer B in 1.2.3.5")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/1.2.3.5/tcp/1234")})
	assert.Truef(allow, "expected gater to allow peer B in 1.2.3.5")

	allow = cg.InterceptAddrDial(pidB, ma.StringCast("/ip4/2.3.4.5/tcp/1234"))
	assert.Truef(allow, "expected gater to allow peer B in 1.2.3.5")

	allow = cg.InterceptAccept(&mockConnMultiaddrs{local: nil, remote: ma.StringCast("/ip4/2.3.4.5/tcp/1234")})
	assert.Truef(allow, "expected gater to allow peer B in 1.2.3.5")
}

func TestConnGater_Score(t *testing.T) {
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cg, err := newConnGater(logger, time.Second*2, time.Second*1)
	assert.Nil(err)
	pidA := peer.ID("A")
	_, err = cg.addPenalty(pidA, 0)
	assert.Equal(errConnGaterIsNotrunning, err)

	wg := &sync.WaitGroup{}
	cg.start(ctx, wg)
	cg.addPenalty(pidA, 0)
	pidB := peer.ID("B")
	cg.addPenalty(pidB, 0)
	assert.Equal(0, len(cg.listBlockedPeers()))
	newScore, _ := cg.addPenalty(pidA, 10)
	assert.Equal(10, newScore)
	assert.Equal(0, len(cg.listBlockedPeers()))
	newScore, _ = cg.addPenalty(pidA, 100)
	assert.Equal(110, newScore)
	assert.Equal(1, len(cg.listBlockedPeers()))

	assert.Falsef(cg.InterceptPeerDial(pidA), "expected gater to deny peer A")
	assert.Truef(cg.InterceptPeerDial(pidB), "expected gater to allow peer B")
	allow := cg.InterceptSecured(network.DirInbound, pidA, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.Falsef(allow, "expected gater to deny peer A")

	allow = cg.InterceptSecured(network.DirInbound, pidB, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.Truef(allow, "expected gater to allow peer B")

	waitForTestCondition(t, func() int { return len(cg.listBlockedPeers()) }, 0)
	assert.Equal(0, len(cg.listBlockedPeers()))
	assert.Truef(cg.InterceptPeerDial(pidA), "expected gater to deny peer A")
	allow = cg.InterceptSecured(network.DirInbound, pidA, &mockConnMultiaddrs{local: nil, remote: nil})
	assert.Truef(allow, "expected gater to deny peer A")

	assert.Equal(0, len(cg.listBlockedPeers()))
	pidC := peer.ID("C")
	cg.addPenalty(pidC, 100)
	assert.Equal(pidC, cg.listBlockedPeers()[0])
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

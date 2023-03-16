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
	manet "github.com/multiformats/go-multiaddr/net"
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

	addr := ma.StringCast("/ip4/7.7.7.7/tcp/4242/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N")
	_, err = cg.addPenalty(addr, MaxPenaltyScore)
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

	addrA := ma.StringCast("/ip4/6.6.6.6/tcp/3131/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5N")
	_, err = cg.addPenalty(addrA, MaxPenaltyScore)
	assert.Nil(err)

	time.Sleep(time.Second)
	addrB := ma.StringCast("/ip4/5.5.5.5/tcp/5454/p2p/12D3KooWGQTQuV6JfgpKpc847NMsxaFnKXDEpkN5kbeH1REW41BR")
	cg.addPenalty(addrB, MaxPenaltyScore)
	addrC := ma.StringCast("/ip4/4.4.4.4/tcp/6262/p2p/12D3KooWGQTQuV6JfgpKpc847NMsxaFnKXDEpkN5kbeH1SDFW34S")
	cg.addPenalty(addrC, MaxPenaltyScore)

	assert.Equal(3, len(cg.listBannedPeers()))
	time.Sleep(time.Second)
	addrD := ma.StringCast("/ip4/3.3.3.3/tcp/7676/p2p/12D3KooWB4J4mraN1nAB9Ge5w8JrzGRKDQ3iGyDyHZdbMWDtYg3R")
	cg.addPenalty(addrD, MaxPenaltyScore)

	time.Sleep(time.Second * 3) // We must really wait for 3 seconds to make sure the peer is removed from the list
	assert.Equal(4, len(cg.listBannedPeers()))

	time.Sleep(time.Second * 2) // We must really wait for 2 seconds to make sure the peer is removed from the list
	assert.Equal(1, len(cg.listBannedPeers()))
	ipD, err := manet.ToIP(addrD)
	assert.Nil(err)
	assert.Equal(ipD.String(), cg.listBannedPeers()[0].String())

	waitForTestCondition(t, func() int { return len(cg.listBannedPeers()) }, 0, testTimeout)
	assert.Equal(0, len(cg.listBannedPeers()))
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

	addrA := ma.StringCast("/ip4/1.2.3.4/tcp/3131/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5M")
	addrB := ma.StringCast("/ip4/1.2.3.5/tcp/3131/p2p/12D3KooWGQTQuV6JfgpKpc847NMsxaFnKXDEpkN5kbeH1REW41BR")
	pidA := peer.ID("A")
	pidB := peer.ID("B")

	// test peer blocking

	allow := cg.InterceptSecured(network.DirInbound, pidA, &mockConnMultiaddrs{local: nil, remote: addrA})
	assert.Truef(allow, "expected gater to allow peer A")

	allow = cg.InterceptSecured(network.DirInbound, pidB, &mockConnMultiaddrs{local: nil, remote: addrB})
	assert.Truef(allow, "expected gater to allow peer B")

	_, err = cg.addPenalty(addrA, MaxPenaltyScore)
	assert.Nil(err)

	allow = cg.InterceptSecured(network.DirInbound, pidA, &mockConnMultiaddrs{local: nil, remote: addrA})
	assert.Falsef(allow, "expected gater to deny peer A")

	allow = cg.InterceptSecured(network.DirInbound, pidB, &mockConnMultiaddrs{local: nil, remote: addrB})
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
	waitForTestCondition(t, func() int { return len(cg.listBannedPeers()) }, 0, time.Millisecond*3100)
	assert.Equal(0, len(cg.listBannedPeers()))

	allow = cg.InterceptSecured(network.DirInbound, pidA, &mockConnMultiaddrs{local: nil, remote: addrA})
	assert.Truef(allow, "expected gater to allow peer A")

	allow = cg.InterceptSecured(network.DirInbound, pidB, &mockConnMultiaddrs{local: nil, remote: addrB})
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
	addrA := ma.StringCast("/ip4/6.6.6.6/tcp/3131/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5M")
	pidA := peer.ID("A")
	_, err = cg.addPenalty(addrA, 0)
	assert.Equal(errConnGaterIsNotrunning, err)

	wg := &sync.WaitGroup{}
	cg.start(ctx, wg)
	cg.addPenalty(addrA, 0)
	addrB := ma.StringCast("/ip4/5.5.5.5/tcp/5454/p2p/12D3KooWGQTQuV6JfgpKpc847NMsxaFnKXDEpkN5kbeH1REW41BR")
	pidB := peer.ID("B")
	cg.addPenalty(addrB, 0)
	assert.Equal(0, len(cg.listBannedPeers()))
	newScore, _ := cg.addPenalty(addrA, 10)
	assert.Equal(10, newScore)
	assert.Equal(0, len(cg.listBannedPeers()))
	newScore, _ = cg.addPenalty(addrA, 100)
	assert.Equal(110, newScore)
	assert.Equal(1, len(cg.listBannedPeers()))

	allow := cg.InterceptSecured(network.DirInbound, pidA, &mockConnMultiaddrs{local: nil, remote: addrA})
	assert.Falsef(allow, "expected gater to deny peer A")

	allow = cg.InterceptSecured(network.DirInbound, pidB, &mockConnMultiaddrs{local: nil, remote: addrB})
	assert.Truef(allow, "expected gater to allow peer B")

	waitForTestCondition(t, func() int { return len(cg.listBannedPeers()) }, 0, time.Millisecond*3100)
	assert.Equal(0, len(cg.listBannedPeers()))
	allow = cg.InterceptSecured(network.DirInbound, pidA, &mockConnMultiaddrs{local: nil, remote: addrA})
	assert.Truef(allow, "expected gater to deny peer A")

	assert.Equal(0, len(cg.listBannedPeers()))
	addrC := ma.StringCast("/ip4/4.4.4.4/tcp/6262/p2p/QmYyQSo1c1Ym7orWxLYvCrM2EmxFTANf8wXmmE7DWjhx5P")
	cg.addPenalty(addrC, 100)
	ipC, err := manet.ToIP(addrC)
	assert.Nil(err)
	assert.Equal(ipC.String(), cg.listBannedPeers()[0].String())
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

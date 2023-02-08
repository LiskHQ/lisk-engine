package p2p

import (
	"strings"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

func TestPeerbook_NewPeerbook(t *testing.T) {
	assert := assert.New(t)

	seedPeers := []string{"/ip4/1.1.1.1/tcp/10/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", "/ip4/2.2.2.2/tcp/20/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB"}
	fixedPeers := []string{"/ip4/3.3.3.3/tcp/30/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", "/ip4/4.4.4.4/tcp/40/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD"}
	blacklistedIPs := []string{"5.5.5.5", "6.6.6.6"}

	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/80")
	addr2, _ := ma.NewMultiaddr("/ip4/5.6.7.8/udp/90")

	knownPeers := []AddressInfo2{
		AddressInfo2{ID: "11111", Addrs: []ma.Multiaddr{addr1}},
		AddressInfo2{ID: "22222", Addrs: []ma.Multiaddr{addr2}},
	}

	pb, err := NewPeerbook(seedPeers, fixedPeers, blacklistedIPs, knownPeers)
	assert.Nil(err)
	assert.NotNil(pb)

	// seed peers
	assert.Equal(2, len(pb.seedPeers))
	assert.Equal("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", pb.seedPeers[0].ID.String())
	assert.Equal("/ip4/1.1.1.1/tcp/10", pb.seedPeers[0].Addrs[0].String())
	assert.Equal("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB", pb.seedPeers[1].ID.String())
	assert.Equal("/ip4/2.2.2.2/tcp/20", pb.seedPeers[1].Addrs[0].String())

	// fixed peers
	assert.Equal(2, len(pb.fixedPeers))
	assert.Equal("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", pb.fixedPeers[0].ID.String())
	assert.Equal("/ip4/3.3.3.3/tcp/30", pb.fixedPeers[0].Addrs[0].String())
	assert.Equal("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD", pb.fixedPeers[1].ID.String())
	assert.Equal("/ip4/4.4.4.4/tcp/40", pb.fixedPeers[1].Addrs[0].String())

	// blacklisted IPs
	assert.Equal(2, len(pb.blacklistedIPs))
	assert.Equal("5.5.5.5", pb.blacklistedIPs[0])
	assert.Equal("6.6.6.6", pb.blacklistedIPs[1])

	// known peers
	assert.Equal(2, len(pb.knownPeers))
	assert.Equal(peer.ID("11111"), pb.knownPeers[0].ID)
	assert.Equal("/ip4/1.2.3.4/tcp/80", pb.knownPeers[0].Addrs[0].String())
	assert.Equal(peer.ID("22222"), pb.knownPeers[1].ID)
	assert.Equal("/ip4/5.6.7.8/udp/90", pb.knownPeers[1].Addrs[0].String())
}

func TestPeerbook_Init(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()

	seedPeers := []string{"/ip4/1.1.1.1/tcp/10/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", "/ip4/2.2.2.2/tcp/20/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB"}
	fixedPeers := []string{"/ip4/3.3.3.3/tcp/30/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", "/ip4/4.4.4.4/tcp/40/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD"}
	blacklistedIPs := []string{"5.5.5.5", "6.6.6.6"}

	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/80")
	addr2, _ := ma.NewMultiaddr("/ip4/5.6.7.8/udp/90")

	knownPeers := []AddressInfo2{
		AddressInfo2{ID: "11111", Addrs: []ma.Multiaddr{addr1}},
		AddressInfo2{ID: "22222", Addrs: []ma.Multiaddr{addr2}},
	}

	pb, _ := NewPeerbook(seedPeers, fixedPeers, blacklistedIPs, knownPeers)

	assert.Nil(pb.logger)
	err := pb.init(logger)
	assert.Nil(err)
	assert.NotNil(pb.logger)
}

func TestPeerbook_InitBlacklistedIPInSeedPeers(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	loggerTest := testLogger{Logger: logger}

	seedPeers := []string{"/ip4/5.5.5.5/tcp/10/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", "/ip4/2.2.2.2/tcp/20/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB"}
	fixedPeers := []string{"/ip4/3.3.3.3/tcp/30/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", "/ip4/4.4.4.4/tcp/40/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD"}
	blacklistedIPs := []string{"5.5.5.5", "6.6.6.6"}

	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/80")
	addr2, _ := ma.NewMultiaddr("/ip4/5.6.7.8/udp/90")

	knownPeers := []AddressInfo2{
		AddressInfo2{ID: "11111", Addrs: []ma.Multiaddr{addr1}},
		AddressInfo2{ID: "22222", Addrs: []ma.Multiaddr{addr2}},
	}

	pb, _ := NewPeerbook(seedPeers, fixedPeers, blacklistedIPs, knownPeers)

	_ = pb.init(&loggerTest)
	idx := slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, "Blacklisted IP %s is present in seed peers") })
	assert.NotEqual(-1, idx)
}

func TestPeerbook_InitBlacklistedIPInFixedPeers(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	loggerTest := testLogger{Logger: logger}

	seedPeers := []string{"/ip4/1.1.1.1/tcp/10/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", "/ip4/2.2.2.2/tcp/20/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB"}
	fixedPeers := []string{"/ip4/3.3.3.3/tcp/30/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", "/ip4/6.6.6.6/tcp/40/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD"}
	blacklistedIPs := []string{"5.5.5.5", "6.6.6.6"}

	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/80")
	addr2, _ := ma.NewMultiaddr("/ip4/5.6.7.8/udp/90")

	knownPeers := []AddressInfo2{
		AddressInfo2{ID: "11111", Addrs: []ma.Multiaddr{addr1}},
		AddressInfo2{ID: "22222", Addrs: []ma.Multiaddr{addr2}},
	}

	pb, _ := NewPeerbook(seedPeers, fixedPeers, blacklistedIPs, knownPeers)

	_ = pb.init(&loggerTest)
	idx := slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, "Blacklisted IP %s is present in fixed peers") })
	assert.NotEqual(-1, idx)
}

func TestPeerbook_SeedPeers(t *testing.T) {
	assert := assert.New(t)

	seedPeers := []string{"/ip4/1.1.1.1/tcp/10/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", "/ip4/2.2.2.2/tcp/20/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB"}
	pb, _ := NewPeerbook(seedPeers, []string{}, []string{}, []AddressInfo2{})

	peers := pb.SeedPeers()
	assert.Equal(2, len(peers))
	assert.Equal("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", peers[0].ID.String())
	assert.Equal("/ip4/1.1.1.1/tcp/10", peers[0].Addrs[0].String())
	assert.Equal("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB", peers[1].ID.String())
	assert.Equal("/ip4/2.2.2.2/tcp/20", peers[1].Addrs[0].String())
}

func TestPeerbook_FixedPeers(t *testing.T) {
	assert := assert.New(t)

	fixedPeers := []string{"/ip4/3.3.3.3/tcp/30/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", "/ip4/4.4.4.4/tcp/40/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD"}
	pb, _ := NewPeerbook([]string{}, fixedPeers, []string{}, []AddressInfo2{})

	peers := pb.FixedPeers()
	assert.Equal(2, len(peers))
	assert.Equal("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", peers[0].ID.String())
	assert.Equal("/ip4/3.3.3.3/tcp/30", peers[0].Addrs[0].String())
	assert.Equal("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD", peers[1].ID.String())
	assert.Equal("/ip4/4.4.4.4/tcp/40", peers[1].Addrs[0].String())
}

func TestPeerbook_BlacklistedIPs(t *testing.T) {
	assert := assert.New(t)

	blacklistedIPs := []string{"5.5.5.5", "6.6.6.6"}
	pb, _ := NewPeerbook([]string{}, []string{}, blacklistedIPs, []AddressInfo2{})

	peers := pb.BlacklistedIPs()
	assert.Equal(2, len(peers))
	assert.Equal("5.5.5.5", peers[0])
	assert.Equal("6.6.6.6", peers[1])
}

func TestPeerbook_BannedIPs(t *testing.T) {
	assert := assert.New(t)

	pb, _ := NewPeerbook([]string{}, []string{}, []string{}, []AddressInfo2{})
	pb.bannedIPs = []BannedIP{BannedIP{ip: "7.7.7.7", timestamp: 12345}, BannedIP{ip: "8.8.8.8", timestamp: 6789}}

	peers := pb.BannedIPs()
	assert.Equal(2, len(peers))
	assert.Equal("7.7.7.7", peers[0].ip)
	assert.Equal(int64(12345), peers[0].timestamp)
	assert.Equal("8.8.8.8", peers[1].ip)
	assert.Equal(int64(6789), peers[1].timestamp)
}

func TestPeerbook_KnownPeers(t *testing.T) {
	assert := assert.New(t)

	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/80")
	addr2, _ := ma.NewMultiaddr("/ip4/5.6.7.8/udp/90")

	knownPeers := []AddressInfo2{
		AddressInfo2{ID: "11111", Addrs: []ma.Multiaddr{addr1}},
		AddressInfo2{ID: "22222", Addrs: []ma.Multiaddr{addr2}},
	}
	pb, _ := NewPeerbook([]string{}, []string{}, []string{}, knownPeers)

	peers := pb.KnownPeers()
	assert.Equal(2, len(peers))
	assert.Equal(peer.ID("11111"), peers[0].ID)
	assert.Equal("/ip4/1.2.3.4/tcp/80", peers[0].Addrs[0].String())
	assert.Equal(peer.ID("22222"), peers[1].ID)
	assert.Equal("/ip4/5.6.7.8/udp/90", peers[1].Addrs[0].String())
}

func TestPeerbook_BanIP(t *testing.T) {
	assert := assert.New(t)

	pb, _ := NewPeerbook([]string{}, []string{}, []string{}, []AddressInfo2{})

	err := pb.BanIP("1.2.3.4")
	assert.Nil(err)
	assert.Equal(1, len(pb.bannedIPs))
	assert.Equal("1.2.3.4", pb.bannedIPs[0].ip)
	assert.True(pb.bannedIPs[0].timestamp > 0)
}

func TestPeerbook_BanIPAlreadyBanned(t *testing.T) {
	assert := assert.New(t)

	pb, _ := NewPeerbook([]string{}, []string{}, []string{}, []AddressInfo2{})

	timestamp := time.Now().Unix() - 10
	pb.bannedIPs = []BannedIP{BannedIP{ip: "1.2.3.4", timestamp: timestamp}}

	err := pb.BanIP("1.2.3.4")
	assert.Nil(err)
	assert.Equal("1.2.3.4", pb.bannedIPs[0].ip)
	assert.True(pb.bannedIPs[0].timestamp > timestamp)
}

func TestPeerbook_BanIPSeedPeer(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	loggerTest := testLogger{Logger: logger}

	seedPeers := []string{"/ip4/1.1.1.1/tcp/10/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", "/ip4/2.2.2.2/tcp/20/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB"}
	pb, _ := NewPeerbook(seedPeers, []string{}, []string{}, []AddressInfo2{})
	_ = pb.init(&loggerTest)

	err := pb.BanIP("1.1.1.1")
	assert.Nil(err)
	assert.Equal(0, len(pb.bannedIPs))
	idx := slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, "IP %s is present in seed peers, will not ban it") })
	assert.NotEqual(-1, idx)
}

func TestPeerbook_BanIPFixedPeer(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	loggerTest := testLogger{Logger: logger}

	fixedPeers := []string{"/ip4/3.3.3.3/tcp/30/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", "/ip4/6.6.6.6/tcp/40/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD"}
	pb, _ := NewPeerbook([]string{}, fixedPeers, []string{}, []AddressInfo2{})
	_ = pb.init(&loggerTest)

	err := pb.BanIP("6.6.6.6")
	assert.Nil(err)
	assert.Equal(0, len(pb.bannedIPs))
	idx := slices.IndexFunc(loggerTest.logs, func(s string) bool { return strings.Contains(s, "IP %s is present in fixed peers, will not ban it") })
	assert.NotEqual(-1, idx)
}

func TestPeerbook_BanIPKnownPeer(t *testing.T) {
	assert := assert.New(t)

	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/80")
	addr2, _ := ma.NewMultiaddr("/ip4/5.6.7.8/udp/90")

	knownPeers := []AddressInfo2{
		AddressInfo2{ID: "11111", Addrs: []ma.Multiaddr{addr1}},
		AddressInfo2{ID: "22222", Addrs: []ma.Multiaddr{addr2}},
	}
	pb, _ := NewPeerbook([]string{}, []string{}, []string{}, knownPeers)

	assert.Equal(2, len(pb.knownPeers))
	err := pb.BanIP("5.6.7.8")
	assert.Nil(err)
	assert.Equal(1, len(pb.bannedIPs))
	assert.Equal("5.6.7.8", pb.bannedIPs[0].ip)
	assert.True(pb.bannedIPs[0].timestamp > 0)
	assert.Equal(1, len(pb.knownPeers))
	assert.Equal(peer.ID("11111"), pb.knownPeers[0].ID)
	assert.Equal("/ip4/1.2.3.4/tcp/80", pb.knownPeers[0].Addrs[0].String())
}

func TestPeerbook_AddPeerToKnownPeers(t *testing.T) {
	assert := assert.New(t)

	pb, _ := NewPeerbook([]string{}, []string{}, []string{}, []AddressInfo2{})

	addr, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/80")
	addrInfo := peer.AddrInfo{ID: "11111", Addrs: []ma.Multiaddr{addr}}

	err := pb.addPeerToKnownPeers(addrInfo)
	assert.Nil(err)
	assert.Equal(1, len(pb.knownPeers))
}

func TestPeerbook_AddPeerToKnownPeersAlreadyInList(t *testing.T) {
	assert := assert.New(t)

	pb, _ := NewPeerbook([]string{}, []string{}, []string{}, []AddressInfo2{})

	addr, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/80")
	addrInfo := peer.AddrInfo{ID: "11111", Addrs: []ma.Multiaddr{addr}}

	// 1st time
	err := pb.addPeerToKnownPeers(addrInfo)
	assert.Nil(err)
	assert.Equal(1, len(pb.knownPeers))

	// 2nd time
	err = pb.addPeerToKnownPeers(addrInfo)
	assert.Nil(err)
	assert.Equal(1, len(pb.knownPeers))
}

func TestPeerbook_AddPeerToKnownPeersSeedPeer(t *testing.T) {
	assert := assert.New(t)

	seedPeers := []string{"/ip4/1.1.1.1/tcp/10/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", "/ip4/2.2.2.2/tcp/20/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB"}
	pb, _ := NewPeerbook(seedPeers, []string{}, []string{}, []AddressInfo2{})

	addr, _ := ma.NewMultiaddr("/ip4/1.1.1.1/tcp/10")
	addrInfo := peer.AddrInfo{ID: "12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", Addrs: []ma.Multiaddr{addr}}

	err := pb.addPeerToKnownPeers(addrInfo)
	assert.Nil(err)
	assert.Equal(0, len(pb.knownPeers))
}

func TestPeerbook_AddPeerToKnownPeersFixedPeer(t *testing.T) {
	assert := assert.New(t)

	fixedPeers := []string{"/ip4/3.3.3.3/tcp/30/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", "/ip4/6.6.6.6/tcp/40/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD"}
	pb, _ := NewPeerbook([]string{}, fixedPeers, []string{}, []AddressInfo2{})

	addr, _ := ma.NewMultiaddr("/ip4/3.3.3.3/tcp/30")
	addrInfo := peer.AddrInfo{ID: "12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", Addrs: []ma.Multiaddr{addr}}

	err := pb.addPeerToKnownPeers(addrInfo)
	assert.Nil(err)
	assert.Equal(0, len(pb.knownPeers))
}

func TestPeerbook_AddPeerToKnownPeersBlacklistedIP(t *testing.T) {
	assert := assert.New(t)

	blacklistedIPs := []string{"5.5.5.5", "6.6.6.6"}
	pb, _ := NewPeerbook([]string{}, []string{}, blacklistedIPs, []AddressInfo2{})

	addr, _ := ma.NewMultiaddr("/ip4/6.6.6.6/tcp/30")
	addrInfo := peer.AddrInfo{ID: "12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", Addrs: []ma.Multiaddr{addr}}

	err := pb.addPeerToKnownPeers(addrInfo)
	assert.Nil(err)
	assert.Equal(0, len(pb.knownPeers))
}

func TestPeerbook_AddPeerToKnownPeersBannedIP(t *testing.T) {
	assert := assert.New(t)

	pb, _ := NewPeerbook([]string{}, []string{}, []string{}, []AddressInfo2{})
	pb.bannedIPs = []BannedIP{BannedIP{ip: "7.7.7.7", timestamp: 12345}, BannedIP{ip: "8.8.8.8", timestamp: 6789}}

	addr, _ := ma.NewMultiaddr("/ip4/7.7.7.7/tcp/30")
	addrInfo := peer.AddrInfo{ID: "12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", Addrs: []ma.Multiaddr{addr}}

	err := pb.addPeerToKnownPeers(addrInfo)
	assert.Nil(err)
	assert.Equal(0, len(pb.knownPeers))
}

func TestPeerbook_IsIPInSeedPeers(t *testing.T) {
	assert := assert.New(t)

	seedPeers := []string{"/ip4/1.1.1.1/tcp/10/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", "/ip4/2.2.2.2/tcp/20/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB"}
	pb, _ := NewPeerbook(seedPeers, []string{}, []string{}, []AddressInfo2{})

	result := pb.isIPInSeedPeers("1.2.3.4")
	assert.False(result)

	result = pb.isIPInSeedPeers("2.2.2.2")
	assert.True(result)
}

func TestPeerbook_IsIPInFixedPeers(t *testing.T) {
	assert := assert.New(t)

	fixedPeers := []string{"/ip4/3.3.3.3/tcp/30/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", "/ip4/6.6.6.6/tcp/40/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD"}
	pb, _ := NewPeerbook([]string{}, fixedPeers, []string{}, []AddressInfo2{})

	result := pb.isIPInFixedPeers("1.2.3.4")
	assert.False(result)

	result = pb.isIPInFixedPeers("3.3.3.3")
	assert.True(result)
}

func TestPeerbook_IsIPBlacklisted(t *testing.T) {
	assert := assert.New(t)

	blacklistedIPs := []string{"5.5.5.5", "6.6.6.6"}
	pb, _ := NewPeerbook([]string{}, []string{}, blacklistedIPs, []AddressInfo2{})

	result := pb.isIPBlacklisted("1.2.3.4")
	assert.False(result)

	result = pb.isIPBlacklisted("6.6.6.6")
	assert.True(result)
}

func TestPeerbook_IsInSeedPeers(t *testing.T) {
	assert := assert.New(t)

	seedPeers := []string{"/ip4/1.1.1.1/tcp/10/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", "/ip4/2.2.2.2/tcp/20/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB"}
	pb, _ := NewPeerbook(seedPeers, []string{}, []string{}, []AddressInfo2{})

	result := pb.isInSeedPeers("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXF")
	assert.False(result)

	result = pb.isInSeedPeers("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA")
	assert.True(result)
}

func TestPeerbook_IsInFixedPeers(t *testing.T) {
	assert := assert.New(t)

	fixedPeers := []string{"/ip4/3.3.3.3/tcp/30/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", "/ip4/6.6.6.6/tcp/40/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD"}
	pb, _ := NewPeerbook([]string{}, fixedPeers, []string{}, []AddressInfo2{})

	result := pb.isInFixedPeers("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA")
	assert.False(result)

	result = pb.isInFixedPeers("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC")
	assert.True(result)
}

func TestPeerbook_IsIPBanned(t *testing.T) {
	assert := assert.New(t)

	pb, _ := NewPeerbook([]string{}, []string{}, []string{}, []AddressInfo2{})
	pb.bannedIPs = []BannedIP{BannedIP{ip: "7.7.7.7", timestamp: 12345}, BannedIP{ip: "8.8.8.8", timestamp: 6789}}

	result := pb.isIPBanned("1.2.3.4")
	assert.False(result)

	result = pb.isIPBanned("8.8.8.8")
	assert.True(result)
}

func TestPeerbook_RemoveIPFromBannedIPs(t *testing.T) {
	assert := assert.New(t)

	pb, _ := NewPeerbook([]string{}, []string{}, []string{}, []AddressInfo2{})
	pb.bannedIPs = []BannedIP{BannedIP{ip: "7.7.7.7", timestamp: 12345}, BannedIP{ip: "8.8.8.8", timestamp: 6789}}

	assert.Equal(2, len(pb.bannedIPs))
	err := pb.removeIPFromBannedIPs("1.2.3.4")
	assert.Nil(err)
	assert.Equal(2, len(pb.bannedIPs))

	err = pb.removeIPFromBannedIPs("8.8.8.8")
	assert.Nil(err)
	assert.Equal(1, len(pb.bannedIPs))
	assert.Equal("7.7.7.7", pb.bannedIPs[0].ip)
}

func TestPeerbook_RemovePeerFromKnownPeers(t *testing.T) {
	assert := assert.New(t)

	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/80")
	addr2, _ := ma.NewMultiaddr("/ip4/5.6.7.8/udp/90")

	knownPeers := []AddressInfo2{
		AddressInfo2{ID: "11111", Addrs: []ma.Multiaddr{addr1}},
		AddressInfo2{ID: "22222", Addrs: []ma.Multiaddr{addr2}},
	}
	pb, _ := NewPeerbook([]string{}, []string{}, []string{}, knownPeers)

	assert.Equal(2, len(pb.knownPeers))
	err := pb.removePeerFromKnownPeers("33333")
	assert.Nil(err)
	assert.Equal(2, len(pb.knownPeers))

	err = pb.removePeerFromKnownPeers("11111")
	assert.Nil(err)
	assert.Equal(1, len(pb.knownPeers))
	assert.Equal(peer.ID("22222"), pb.knownPeers[0].ID)
	assert.Equal("/ip4/5.6.7.8/udp/90", pb.knownPeers[0].Addrs[0].String())
}

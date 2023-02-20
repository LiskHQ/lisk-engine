package p2p

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

func TestPeerbook_NewPeerbook(t *testing.T) {
	assert := assert.New(t)

	seedPeers := []string{"/ip4/1.1.1.1/tcp/10/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", "/ip4/2.2.2.2/tcp/20/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB"}
	fixedPeers := []string{"/ip4/3.3.3.3/tcp/30/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", "/ip4/4.4.4.4/tcp/40/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD"}
	permanentlyBlacklistedIPs := []string{"5.5.5.5", "6.6.6.6"}

	pb, err := NewPeerbook(seedPeers, fixedPeers, permanentlyBlacklistedIPs)
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

	// permanently blacklisted IPs
	assert.Equal(2, len(pb.permanentlyBlacklistedIPs))
	assert.Equal("5.5.5.5", pb.permanentlyBlacklistedIPs[0])
	assert.Equal("6.6.6.6", pb.permanentlyBlacklistedIPs[1])
}

func TestPeerbook_Init(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()

	seedPeers := []string{"/ip4/1.1.1.1/tcp/10/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", "/ip4/2.2.2.2/tcp/20/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB"}
	fixedPeers := []string{"/ip4/3.3.3.3/tcp/30/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", "/ip4/4.4.4.4/tcp/40/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD"}
	permanentlyBlacklistedIPs := []string{"5.5.5.5", "6.6.6.6"}

	pb, _ := NewPeerbook(seedPeers, fixedPeers, permanentlyBlacklistedIPs)

	assert.Nil(pb.logger)
	pb.init(logger)
	assert.NotNil(pb.logger)
}

func TestPeerbook_InitPermanentlyBlacklistedIPInSeedPeers(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	loggerTest := testLogger{Logger: logger}

	seedPeers := []string{"/ip4/5.5.5.5/tcp/10/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", "/ip4/2.2.2.2/tcp/20/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB"}
	fixedPeers := []string{"/ip4/3.3.3.3/tcp/30/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", "/ip4/4.4.4.4/tcp/40/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD"}
	permanentlyBlacklistedIPs := []string{"5.5.5.5", "6.6.6.6"}

	pb, _ := NewPeerbook(seedPeers, fixedPeers, permanentlyBlacklistedIPs)

	pb.init(&loggerTest)
	idx := slices.IndexFunc(loggerTest.logs, func(s string) bool {
		return strings.Contains(s, "Permanently blacklisted IP %s is present in seed peers")
	})
	assert.NotEqual(-1, idx)
}

func TestPeerbook_InitPermanentlyBlacklistedIPInFixedPeers(t *testing.T) {
	assert := assert.New(t)

	logger, _ := log.NewDefaultProductionLogger()
	loggerTest := testLogger{Logger: logger}

	seedPeers := []string{"/ip4/1.1.1.1/tcp/10/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", "/ip4/2.2.2.2/tcp/20/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB"}
	fixedPeers := []string{"/ip4/3.3.3.3/tcp/30/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", "/ip4/6.6.6.6/tcp/40/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD"}
	permanentlyBlacklistedIPs := []string{"5.5.5.5", "6.6.6.6"}

	pb, _ := NewPeerbook(seedPeers, fixedPeers, permanentlyBlacklistedIPs)

	pb.init(&loggerTest)
	idx := slices.IndexFunc(loggerTest.logs, func(s string) bool {
		return strings.Contains(s, "Permanently blacklisted IP %s is present in fixed peers")
	})
	assert.NotEqual(-1, idx)
}

func TestPeerbook_SeedPeers(t *testing.T) {
	assert := assert.New(t)

	seedPeers := []string{"/ip4/1.1.1.1/tcp/10/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", "/ip4/2.2.2.2/tcp/20/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB"}
	pb, _ := NewPeerbook(seedPeers, []string{}, []string{})

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
	pb, _ := NewPeerbook([]string{}, fixedPeers, []string{})

	peers := pb.FixedPeers()
	assert.Equal(2, len(peers))
	assert.Equal("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", peers[0].ID.String())
	assert.Equal("/ip4/3.3.3.3/tcp/30", peers[0].Addrs[0].String())
	assert.Equal("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD", peers[1].ID.String())
	assert.Equal("/ip4/4.4.4.4/tcp/40", peers[1].Addrs[0].String())
}

func TestPeerbook_PermanentlyBlacklistedIPs(t *testing.T) {
	assert := assert.New(t)

	permanentlyBlacklistedIPs := []string{"5.5.5.5", "6.6.6.6"}
	pb, _ := NewPeerbook([]string{}, []string{}, permanentlyBlacklistedIPs)

	peers := pb.PermanentlyBlacklistedIPs()
	assert.Equal(2, len(peers))
	assert.Equal("5.5.5.5", peers[0])
	assert.Equal("6.6.6.6", peers[1])
}

func TestPeerbook_IsIPInSeedPeers(t *testing.T) {
	assert := assert.New(t)

	seedPeers := []string{"/ip4/1.1.1.1/tcp/10/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", "/ip4/2.2.2.2/tcp/20/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB"}
	pb, _ := NewPeerbook(seedPeers, []string{}, []string{})

	result := pb.isIPInSeedPeers("1.2.3.4")
	assert.False(result)

	result = pb.isIPInSeedPeers("2.2.2.2")
	assert.True(result)
}

func TestPeerbook_IsIPInFixedPeers(t *testing.T) {
	assert := assert.New(t)

	fixedPeers := []string{"/ip4/3.3.3.3/tcp/30/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", "/ip4/6.6.6.6/tcp/40/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD"}
	pb, _ := NewPeerbook([]string{}, fixedPeers, []string{})

	result := pb.isIPInFixedPeers("1.2.3.4")
	assert.False(result)

	result = pb.isIPInFixedPeers("3.3.3.3")
	assert.True(result)
}

func TestPeerbook_IsIPPermanentlyBlacklisted(t *testing.T) {
	assert := assert.New(t)

	permanentlyBlacklistedIPs := []string{"5.5.5.5", "6.6.6.6"}
	pb, _ := NewPeerbook([]string{}, []string{}, permanentlyBlacklistedIPs)

	result := pb.isIPPermanentlyBlacklisted("1.2.3.4")
	assert.False(result)

	result = pb.isIPPermanentlyBlacklisted("6.6.6.6")
	assert.True(result)
}

func TestPeerbook_IsInSeedPeers(t *testing.T) {
	assert := assert.New(t)

	seedPeers := []string{"/ip4/1.1.1.1/tcp/10/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA", "/ip4/2.2.2.2/tcp/20/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXB"}
	pb, _ := NewPeerbook(seedPeers, []string{}, []string{})

	result := pb.isInSeedPeers("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXF")
	assert.False(result)

	result = pb.isInSeedPeers("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA")
	assert.True(result)
}

func TestPeerbook_IsInFixedPeers(t *testing.T) {
	assert := assert.New(t)

	fixedPeers := []string{"/ip4/3.3.3.3/tcp/30/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC", "/ip4/6.6.6.6/tcp/40/p2p/12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXD"}
	pb, _ := NewPeerbook([]string{}, fixedPeers, []string{})

	result := pb.isInFixedPeers("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXA")
	assert.False(result)

	result = pb.isInFixedPeers("12D3KooWNLGFBbaLyFtMzXAPmD7xL63xjXoC4Bg1cW8zoD8jJdXC")
	assert.True(result)
}

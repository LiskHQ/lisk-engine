package p2p

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	logger "github.com/LiskHQ/lisk-engine/pkg/log"
)

func TestP2P_NewP2P(t *testing.T) {
	assert := assert.New(t)

	config := Config{}
	err := config.InsertDefault()
	assert.Nil(err)
	p2p := NewP2P(config)
	assert.NotNil(p2p)
	assert.Equal("1.0", p2p.config.Version)
	assert.Equal([]string{testIPv4TCP, testIPv4UDP}, p2p.config.Addresses)
	assert.Equal(false, p2p.config.AllowIncomingConnections)
	assert.Equal(false, p2p.config.EnableNATService)
	assert.Equal(false, p2p.config.EnableUsingRelayService)
	assert.Equal(false, p2p.config.EnableRelayService)
	assert.Equal(false, p2p.config.EnableHolePunching)
	assert.Equal([]string{}, p2p.config.SeedPeers)
	assert.Equal([]string{}, p2p.config.FixedPeers)
	assert.Equal([]string{}, p2p.config.BlacklistedIPs)
	assert.Equal([]AddressInfo2{}, p2p.config.KnownPeers)
	assert.Equal(100, p2p.config.MaxInboundConnections)
	assert.Equal(20, p2p.config.MaxOutboundConnections)
	assert.Equal(false, p2p.config.IsSeedNode)
	assert.Equal("lisk-test", p2p.config.NetworkName)
	assert.NotNil(p2p.GossipSub)
}

func TestP2P_Start(t *testing.T) {
	assert := assert.New(t)

	config := Config{}
	_ = config.InsertDefault()
	p2p := NewP2P(config)
	logger, _ := logger.NewDefaultProductionLogger()
	err := p2p.Start(logger)
	assert.Nil(err)
	assert.Equal(logger, p2p.logger)
	assert.NotNil(p2p.Peer)
	assert.NotNil(p2p.host)
	assert.NotNil(p2p.MessageProtocol)
	assert.NotNil(p2p.Peer.peerbook)
}

func TestP2P_Stop(t *testing.T) {
	assert := assert.New(t)

	config := Config{}
	_ = config.InsertDefault()
	p2p := NewP2P(config)
	logger, _ := logger.NewDefaultProductionLogger()
	_ = p2p.Start(logger)

	ch := make(chan struct{})
	defer close(ch)

	go func() {
		err := p2p.Stop()
		assert.Nil(err)
		ch <- struct{}{}
	}()

	select {
	case <-ch:
		break
	case <-time.After(time.Second):
		t.Fatalf("timeout occurs, P2P stop is not working")
	}
}

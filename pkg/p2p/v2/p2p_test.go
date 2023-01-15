package p2p

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"

	logger "github.com/LiskHQ/lisk-engine/pkg/log"
)

func TestP2P_NewP2P(t *testing.T) {
	config := Config{}
	err := config.InsertDefault()
	assert.Nil(t, err)
	p2p := NewP2P(config)
	assert.NotNil(t, p2p)
	assert.Equal(t, "1.0", p2p.config.Version)
	assert.Equal(t, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, p2p.config.Addresses)
	assert.Equal(t, false, p2p.config.AllowIncomingConnections)
	assert.Equal(t, false, p2p.config.EnableNATService)
	assert.Equal(t, false, p2p.config.EnableUsingRelayService)
	assert.Equal(t, false, p2p.config.EnableRelayService)
	assert.Equal(t, false, p2p.config.EnableHolePunching)
	assert.Equal(t, []peer.ID{}, p2p.config.SeedPeers)
	assert.Equal(t, []peer.ID{}, p2p.config.FixedPeers)
	assert.Equal(t, []peer.ID{}, p2p.config.BlackListedPeers)
	assert.Equal(t, 100, p2p.config.MaxInboundConnections)
	assert.Equal(t, 20, p2p.config.MaxOutboundConnections)
	assert.Equal(t, false, p2p.config.IsSeedNode)
	assert.Equal(t, "lisk-test", p2p.config.NetworkName)
	assert.Equal(t, []string{}, p2p.config.SeedNodes)
}

func TestP2P_Start(t *testing.T) {
	config := Config{}
	_ = config.InsertDefault()
	p2p := NewP2P(config)
	logger, _ := logger.NewDefaultProductionLogger()
	err := p2p.Start(logger)
	assert.Nil(t, err)
	assert.Equal(t, logger, p2p.logger)
	assert.NotNil(t, p2p.host)
}

func TestP2P_Stop(t *testing.T) {
	config := Config{}
	_ = config.InsertDefault()
	p2p := NewP2P(config)
	logger, _ := logger.NewDefaultProductionLogger()
	_ = p2p.Start(logger)

	ch := make(chan struct{})
	defer close(ch)

	go func() {
		err := p2p.Stop()
		assert.Nil(t, err)
		ch <- struct{}{}
	}()

	select {
	case <-ch:
		break
	case <-time.After(time.Second):
		t.Fatalf("timeout occurs, P2P stop is not working")
	}
}

package p2p

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"
	"github.com/stretchr/testify/assert"
)

func TestBlacklistIPs(t *testing.T) {
	assert := assert.New(t)

	cg, err := conngater.NewBasicConnectionGater(nil)
	assert.Nil(err)

	containsInvalidIPs := []string{
		"128.20.12.11",
		"1.222.2222.12.12",
		"12.30.28.110",
	}
	_, err = ConnectionGaterOption(cg, containsInvalidIPs)
	assert.ErrorContains(err, "is invalid")

	validIPs := []string{
		"127.0.0.1",
		"192.168.2.30",
	}
	_, err = ConnectionGaterOption(cg, validIPs)
	assert.Nil(err)
}

func TestConnGater_ExpireTime(t *testing.T) {
	assert := assert.New(t)

	g, err := newGater(time.Second*4, time.Second*2)
	assert.Nil(err)
	g.start()

	pidA := peer.ID("A")
	g.addPeer(pidA)

	time.Sleep(time.Second)
	pidB := peer.ID("B")
	g.addPeer(pidB)
	pidC := peer.ID("C")
	g.addPeer(pidC)

	gater := g.connectionGater()
	assert.Equal(3, len(gater.ListBlockedPeers()))
	time.Sleep(time.Second)
	pidD := peer.ID("D")
	g.addPeer(pidD)

	time.Sleep(time.Second * 3)
	assert.Equal(4, len(gater.ListBlockedPeers()))
	time.Sleep(time.Second * 2)
	assert.Equal(1, len(gater.ListBlockedPeers()))
	time.Sleep(time.Second)
	assert.Equal(0, len(gater.ListBlockedPeers()))
}

package p2p

import (
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
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
	cg.blockPeer(pidA)

	time.Sleep(time.Second)
	pidB := peer.ID("B")
	cg.blockPeer(pidB)
	pidC := peer.ID("C")
	cg.blockPeer(pidC)

	assert.Equal(3, len(cg.connGater.ListBlockedPeers()))
	time.Sleep(time.Second)
	pidD := peer.ID("D")
	cg.blockPeer(pidD)

	time.Sleep(time.Second * 3)
	assert.Equal(4, len(cg.connGater.ListBlockedPeers()))
	time.Sleep(time.Second * 2)
	assert.Equal(1, len(cg.connGater.ListBlockedPeers()))
	assert.Equal(pidD, cg.connGater.ListBlockedPeers()[0])
	time.Sleep(time.Second)
	assert.Equal(0, len(cg.connGater.ListBlockedPeers()))
}

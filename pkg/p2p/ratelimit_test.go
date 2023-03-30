package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/stretchr/testify/assert"
)

func TestRateLimit_RPCMessageCounter_BanPeer(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := log.NewDefaultProductionLogger()
	cfg := &Config{Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()

	node1 := NewConnection(cfg)
	testHandler := func(w ResponseWriter, req *Request) {}
	node1.RegisterRPCHandler(testRPC, testHandler, WithRPCMessageCounter(5, 25))
	node1.MessageProtocol.rateLimit.interval = time.Millisecond * 100 // Decrease the interval to speed up the test
	node1.MessageProtocol.timeout = time.Millisecond * 10             // Decrease the interval to speed up the test
	err := node1.Start(logger, []byte{})
	assert.Nil(err)
	node1Addrs, _ := node1.MultiAddress()
	node1AddrInfo, _ := AddrInfoFromMultiAddr(node1Addrs[0])
	node1.connGater.intervalCheck = time.Millisecond * 25 // Decrease the interval to speed up the test

	node2 := NewConnection(cfg)
	node2.RegisterRPCHandler(testRPC, func(w ResponseWriter, req *Request) {
		w.Write([]byte("Average RTT with you:"))
	}, WithRPCMessageCounter(5, 25))
	err = node2.Start(logger, []byte{})
	assert.Nil(err)
	node2Addrs, _ := node2.MultiAddress()
	node2AddrInfo, _ := AddrInfoFromMultiAddr(node2Addrs[0])

	err = node1.Connect(ctx, *node2AddrInfo)
	assert.NoError(err)

	// Wait 4 times for the rate limiter to apply the penalty to node2 (4 * 25 = 100)
	for i := 0; i < 4; i++ {
		// Send 6 requests to node1 (1 too many) that a penalty will be applied
		for j := 0; j < 6; j++ {
			_, err := node2.request(ctx, node1.ID(), testRPC, []byte(testRequestData))
			// Last 24 request should fail because of the penalty reaches 100 and a peer is banned
			if i == 3 && j == 5 {
				assert.NotNil(err)
			} else {
				assert.Nil(err)
			}
		}
	}

	// Check that a peer is banned
	assert.Equal(0, len(node1.ConnectedPeers()))

	// Try to connect node1 with node2 (it should not be possible because node2 is banned)
	err = node1.Connect(ctx, *node2AddrInfo)
	assert.NotNil(err)

	// Try to connect node2 with node1 and send a request (it should not be possible because node2 is banned)
	err = node2.Connect(ctx, *node1AddrInfo)
	assert.Nil(err)
	_, err = node2.request(ctx, node1.ID(), testRPC, []byte(testRequestData))
	assert.NotNil(err)

	node1.Stop()
	node2.Stop()
}

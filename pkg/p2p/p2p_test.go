package p2p

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	logger "github.com/LiskHQ/lisk-engine/pkg/log"
)

const (
	testTimestamp    = int64(123456789)
	testPeerID       = PeerID("testPeerID")
	testEvent        = "testEvent"
	testProcedure    = "testProcedure"
	testRPC          = "testRPC"
	testData         = "testData"
	testReqMsgID     = "123456789"
	testRequestData  = "testRequestData"
	testResponseData = "testResponseData"
	testVersion      = "2.0"

	testIPv4TCP = "/ip4/127.0.0.1/tcp/0"
	testIPv4UDP = "/ip4/127.0.0.1/udp/0/quic"

	testTopic1 = "testTopic1"
	testTopic2 = "testTopic2"
	testTopic3 = "testTopic3"

	testError = "testError"

	testTimeout = time.Second * 3
)

var (
	testChainID = []byte{1, 2, 0, 0}
	testMV      = func(ctx context.Context, msg *Message) ValidationResult {
		if bytes.Contains(msg.Data, []byte("Invalid")) {
			return ValidationReject
		} else {
			return ValidationAccept
		}
	}
)

func waitForTestCondition(t *testing.T, condition func() int, expected int, timeout time.Duration) {
	timeoutCondition := time.After(timeout)

	for {
		if condition() == expected {
			break
		}

		select {
		case <-timeoutCondition:
			t.Fatalf("timeout occurs, unable to meet condition")
		case <-time.After(time.Millisecond * 100):
			// check if above condition is true
		}
	}
}

type testLogger struct {
	logger.Logger
	logs []string
}

func (l *testLogger) Debugf(msg string, others ...interface{}) {
	l.logs = append(l.logs, msg)
}

func (l *testLogger) Warningf(msg string, others ...interface{}) {
	l.logs = append(l.logs, msg)
}

func (l *testLogger) Errorf(msg string, others ...interface{}) {
	l.logs = append(l.logs, msg)
}

func TestP2P_NewP2P(t *testing.T) {
	assert := assert.New(t)

	cfg := &Config{}
	err := cfg.insertDefault()
	assert.Nil(err)
	p2p := NewConnection(logger.DefaultLogger, cfg)
	assert.NotNil(p2p)
	assert.Equal("1.0", p2p.cfg.Version)
	assert.Equal([]string{}, p2p.cfg.Addresses)
	assert.Equal(false, p2p.cfg.EnableNATService)
	assert.Equal(false, p2p.cfg.EnableUsingRelayService)
	assert.Equal(false, p2p.cfg.EnableRelayService)
	assert.Equal(false, p2p.cfg.EnableHolePunching)
	assert.Equal([]string{}, p2p.cfg.SeedPeers)
	assert.Equal([]string{}, p2p.cfg.FixedPeers)
	assert.Equal([]string{}, p2p.cfg.BlacklistedIPs)
	assert.Equal(20, p2p.cfg.MinNumOfConnections)
	assert.Equal(100, p2p.cfg.MaxNumOfConnections)
	assert.Equal(false, p2p.cfg.IsSeedPeer)
	assert.NotNil(p2p.GossipSub)
}

func TestP2P_Start(t *testing.T) {
	assert := assert.New(t)

	cfg := &Config{}
	_ = cfg.insertDefault()
	logger, _ := logger.NewDefaultProductionLogger()
	p2p := NewConnection(logger, cfg)
	err := p2p.Start([]byte{})
	assert.Nil(err)
	assert.Equal(logger, p2p.logger)
	assert.NotNil(p2p.Peer)
	assert.NotNil(p2p.host)
	assert.NotNil(p2p.MessageProtocol)
	assert.NotNil(p2p.Peer.peerbook)
	assert.NotNil(p2p.bootCloser)

	p2p.Stop()
}

func TestP2P_AddPenalty(t *testing.T) {
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &Config{Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()
	logger, _ := logger.NewDefaultProductionLogger()
	node1 := NewConnection(logger, cfg)
	node2 := NewConnection(logger, cfg)
	node1.RegisterEventHandler(testTopic1, func(event *Event) {}, nil)
	node2.RegisterEventHandler(testTopic1, func(event *Event) {}, nil)
	err := node1.Start([]byte{})
	assert.Nil(err)
	err = node2.Start([]byte{})
	assert.Nil(err)

	err = node2.Publish(ctx, testTopic1, testMessageData)
	assert.Nil(err)
	p2Addrs, err := node2.MultiAddress()
	assert.Nil(err)
	p2AddrInfo, err := AddrInfoFromMultiAddr(p2Addrs[0])
	assert.Nil(err)
	err = node1.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)

	node1.ApplyPenalty(p2AddrInfo.ID, 10)
	assert.Equal(node2.ID(), node1.ConnectedPeers()[0])
	node1.ApplyPenalty(p2AddrInfo.ID, MaxPenaltyScore)
	assert.Equal(len(node1.ConnectedPeers()), 0)

	err = node1.Connect(ctx, *p2AddrInfo)
	assert.Containsf(err.Error(), "no good addresses", "Connection should be rejected by ConnectionGater")

	node1.Stop()
	node2.Stop()
}

func TestP2P_BanPeer(t *testing.T) {
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &Config{Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()
	logger, _ := logger.NewDefaultProductionLogger()
	node1 := NewConnection(logger, cfg)
	node2 := NewConnection(logger, cfg)
	node1.RegisterEventHandler(testTopic1, func(event *Event) {}, nil)
	node2.RegisterEventHandler(testTopic1, func(event *Event) {}, nil)
	err := node1.Start([]byte{})
	assert.Nil(err)
	err = node2.Start([]byte{})
	assert.Nil(err)

	err = node2.Publish(ctx, testTopic1, testMessageData)
	assert.Nil(err)
	p2Addrs, err := node2.MultiAddress()
	assert.Nil(err)
	p2AddrInfo, err := AddrInfoFromMultiAddr(p2Addrs[0])
	assert.Nil(err)
	err = node1.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)

	node1.BanPeer(p2AddrInfo.ID)
	assert.Equal(len(node1.ConnectedPeers()), 0)

	err = node1.Connect(ctx, *p2AddrInfo)
	assert.Containsf(err.Error(), "no good addresses", "Connection should be rejected by ConnectionGater")

	node1.Stop()
	node2.Stop()
}

func TestP2P_Stop(t *testing.T) {
	assert := assert.New(t)

	cfg := &Config{}
	_ = cfg.insertDefault()
	logger, _ := logger.NewDefaultProductionLogger()
	p2p := NewConnection(logger, cfg)
	_ = p2p.Start([]byte{})

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

	p2p.Stop()
}

func TestP2P_ConnectionsHandler_DropRandomPeer(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := Config{Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg.insertDefault()
	logger, _ := logger.NewDefaultProductionLogger()
	p2p := NewConnection(logger, &cfg)
	p2p.dropConnTimeout = 1 * time.Second // Set drop random connection timeout to 1s to speed up the test
	p2p.cfg.MinNumOfConnections = 1
	err := p2p.Start([]byte{})
	assert.Nil(err)

	// Create two new peers and connect them to our p2p node
	wg := &sync.WaitGroup{}
	p1, _ := newPeer(ctx, wg, logger, []byte{}, &cfg)
	p2, _ := newPeer(ctx, wg, logger, []byte{}, &cfg)
	p1Addrs, _ := p1.MultiAddress()
	p1AddrInfo, _ := AddrInfoFromMultiAddr(p1Addrs[0])
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

	err = p2p.Connect(ctx, *p1AddrInfo)
	assert.Nil(err)
	err = p2p.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)

	// Check if the number of connected peers is the same as the one we have connected
	assert.Equal(2, len(p2p.ConnectedPeers()))

	// Wait for the connections handler to finish
	waitForTestCondition(t, func() int { return len(p2p.ConnectedPeers()) }, 1, testTimeout)

	// Check if the number of connected peers is lower than the one we have connected
	assert.Equal(1, len(p2p.ConnectedPeers()))

	p1.close()
	p2.close()
	p2p.Stop()
}

func TestP2P_ConnectionsHandler_DropRandomPeerFixedPeer(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := logger.NewDefaultProductionLogger()

	// Create two new peers
	cfg1 := Config{Addresses: []string{testIPv4TCP, testIPv4UDP}}
	_ = cfg1.insertDefault()

	wg := &sync.WaitGroup{}
	p1, _ := newPeer(ctx, wg, logger, []byte{1, 2, 3}, &cfg1)
	p2, _ := newPeer(ctx, wg, logger, []byte{4, 5, 6}, &cfg1)
	p1Addrs, _ := p1.MultiAddress()
	p1AddrInfo, _ := AddrInfoFromMultiAddr(p1Addrs[0])
	p2Addrs, _ := p2.MultiAddress()
	p2AddrInfo, _ := AddrInfoFromMultiAddr(p2Addrs[0])

	// Create a p2p node with fixed peers
	cfg2 := Config{
		Addresses:  []string{testIPv4TCP, testIPv4UDP},
		FixedPeers: []string{p1Addrs[0]},
	}

	_ = cfg2.insertDefault()
	p2p := NewConnection(logger, &cfg2)
	p2p.dropConnTimeout = 1 * time.Second // Set drop random connection timeout to 1s to speed up the test
	p2p.cfg.MinNumOfConnections = 1

	err := p2p.Start([]byte{})
	assert.Nil(err)

	// Connect to our p2p node the two peers we created
	err = p2p.Connect(ctx, *p1AddrInfo)
	assert.Nil(err)
	err = p2p.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)

	// Check if the number of connected peers is the same as the one we have connected
	assert.Equal(2, len(p2p.ConnectedPeers()))

	// Wait for the connections handler to finish
	waitForTestCondition(t, func() int { return len(p2p.ConnectedPeers()) }, 1, testTimeout)

	// Check if the number of connected peers is lower than the one we have connected
	assert.Equal(1, len(p2p.ConnectedPeers()))
	// And that the remaining peer is the one we have fixed
	assert.Equal(p1.ID(), p2p.ConnectedPeers()[0])

	p1.close()
	p2.close()
	p2p.Stop()
}

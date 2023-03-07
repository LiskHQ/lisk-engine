package p2p

import (
	"bytes"
	"context"
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
	p2p := NewConnection(cfg)
	assert.NotNil(p2p)
	assert.Equal("1.0", p2p.cfg.Version)
	assert.Equal([]string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, p2p.cfg.Addresses)
	assert.Equal(false, p2p.cfg.AllowIncomingConnections)
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
	p2p := NewConnection(cfg)
	logger, _ := logger.NewDefaultProductionLogger()
	err := p2p.Start(logger, []byte{})
	assert.Nil(err)
	assert.Equal(logger, p2p.logger)
	assert.NotNil(p2p.Peer)
	assert.NotNil(p2p.host)
	assert.NotNil(p2p.MessageProtocol)
	assert.NotNil(p2p.Peer.peerbook)
	assert.NotNil(p2p.bootCloser)
}

func TestP2P_AddPenalty(t *testing.T) {
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &Config{
		AllowIncomingConnections: true,
		Addresses:                []string{testIPv4TCP, testIPv4UDP},
	}
	_ = cfg.insertDefault()
	node1 := NewConnection(cfg)
	node2 := NewConnection(cfg)
	logger, _ := logger.NewDefaultProductionLogger()
	node1.RegisterEventHandler(testTopic1, func(event *Event) {}, nil)
	node2.RegisterEventHandler(testTopic1, func(event *Event) {}, nil)
	err := node1.Start(logger, []byte{})
	assert.Nil(err)
	err = node2.Start(logger, []byte{})
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
	assert.Containsf(err.Error(), "gater disallow", "Connection should be rejected by ConnectionGater")
}

func TestP2P_Stop(t *testing.T) {
	assert := assert.New(t)

	cfg := &Config{}
	_ = cfg.insertDefault()
	p2p := NewConnection(cfg)
	logger, _ := logger.NewDefaultProductionLogger()
	_ = p2p.Start(logger, []byte{})

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

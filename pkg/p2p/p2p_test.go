package p2p

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	cfg "github.com/LiskHQ/lisk-engine/pkg/engine/config"
	logger "github.com/LiskHQ/lisk-engine/pkg/log"
)

const (
	testTimestamp    = int64(123456789)
	testPeerID       = "testPeerID"
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

	cfgNet := cfg.NetworkConfig{}
	err := cfgNet.InsertDefault()
	assert.Nil(err)
	p2p := NewP2P(&cfgNet)
	assert.NotNil(p2p)
	assert.Equal("1.0", p2p.cfgNet.Version)
	assert.Equal([]string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, p2p.cfgNet.Addresses)
	assert.True(p2p.cfgNet.AdvertiseAddresses)
	assert.Equal(false, p2p.cfgNet.AllowIncomingConnections)
	assert.Equal(false, p2p.cfgNet.EnableNATService)
	assert.Equal(false, p2p.cfgNet.EnableUsingRelayService)
	assert.Equal(false, p2p.cfgNet.EnableRelayService)
	assert.Equal(false, p2p.cfgNet.EnableHolePunching)
	assert.Equal([]string{}, p2p.cfgNet.SeedPeers)
	assert.Equal([]string{}, p2p.cfgNet.FixedPeers)
	assert.Equal([]string{}, p2p.cfgNet.BlacklistedIPs)
	assert.Equal(20, p2p.cfgNet.MinNumOfConnections)
	assert.Equal(100, p2p.cfgNet.MaxNumOfConnections)
	assert.Equal(false, p2p.cfgNet.IsSeedNode)
	assert.NotNil(p2p.GossipSub)
}

func TestP2P_Start(t *testing.T) {
	assert := assert.New(t)

	cfgNet := cfg.NetworkConfig{}
	_ = cfgNet.InsertDefault()
	p2p := NewP2P(&cfgNet)
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

	cfgNet := cfg.NetworkConfig{
		AllowIncomingConnections: true,
		Addresses:                []string{testIPv4TCP, testIPv4UDP},
	}
	_ = cfgNet.InsertDefault()
	node1 := NewP2P(&cfgNet)
	node2 := NewP2P(&cfgNet)
	logger, _ := logger.NewDefaultProductionLogger()
	node1.RegisterEventHandler(testTopic1, func(event *Event) {}, nil)
	node2.RegisterEventHandler(testTopic1, func(event *Event) {}, nil)
	err := node1.Start(logger, []byte{})
	assert.Nil(err)
	err = node2.Start(logger, []byte{})
	assert.Nil(err)

	err = node2.Publish(ctx, testTopic1, testMessageData)
	assert.Nil(err)
	p2Addrs, err := node2.P2PAddrs()
	assert.Nil(err)
	p2AddrInfo, err := PeerInfoFromMultiAddr(p2Addrs[0].String())
	assert.Nil(err)
	err = node1.Connect(ctx, *p2AddrInfo)
	assert.Nil(err)

	node1.ApplyPenalty(string(p2AddrInfo.ID), 10)
	assert.Equal(node2.ID(), node1.ConnectedPeers()[0])
	node1.ApplyPenalty(string(p2AddrInfo.ID), MaxScore)
	assert.Equal(len(node1.ConnectedPeers()), 0)

	err = node1.Connect(ctx, *p2AddrInfo)
	assert.Containsf(err.Error(), "gater disallow", "Connection should be rejected by ConnectionGater")
}

func TestP2P_Stop(t *testing.T) {
	assert := assert.New(t)

	cfgNet := cfg.NetworkConfig{}
	_ = cfgNet.InsertDefault()
	p2p := NewP2P(&cfgNet)
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

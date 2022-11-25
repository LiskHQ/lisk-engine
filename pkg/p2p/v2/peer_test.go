package p2p_v2

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/libp2p/go-libp2p-core/network"
)

func TestCreate(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	p, err := Create(logger, context.Background())
	if err != nil {
		t.Fatalf("error while creating a new peer: %v", err)
	}
	if p.host == nil {
		t.Fatalf("host is nil")
	}
	if p.MessageProtocol == nil {
		t.Fatalf("message protocol is nil")
	}
}

func TestClose(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	p, _ := Create(logger, context.Background())
	err := p.host.Close()
	if err != nil {
		t.Fatalf("error while closing a peer: %v", err)
	}
}

func TestConnect(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	p1, _ := Create(logger, context.Background())
	p2, _ := Create(logger, context.Background())
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	err := p1.Connect(*p2AddrInfo)
	if err != nil {
		t.Fatalf("error while connecting to a peer: %v", err)
	}
	if p1.ConnectedPeers()[0] != p2.ID() {
		t.Fatalf("wrong peer connection, got %v, want %v", p2.ID(), p1.host.Network().Peers()[0])
	}
}

type TestMessageReceive struct {
	msg  string
	done chan any
}

func (tmr *TestMessageReceive) onMessageReceive(s network.Stream) {
	buf, _ := io.ReadAll(s)
	s.Close()
	tmr.msg = string(buf)
	tmr.done <- "done"
}

func TestSendProtoMessage(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	tmr := TestMessageReceive{done: make(chan any)}

	p1, _ := Create(logger, context.Background())
	p2, _ := Create(logger, context.Background())
	p2.host.SetStreamHandler(messageProtocolID, tmr.onMessageReceive)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	_ = p1.Connect(*p2AddrInfo)
	err := p1.sendProtoMessage(p2.ID(), messageProtocolID, "Test protocol message")
	if err != nil {
		t.Fatalf("error while sending a protocol message %v", err)
	}

	select {
	case <-tmr.done:
		break
	case <-time.After(time.Second * pingTimeout):
		t.Fatalf("timeout occurs, message was not delivered to a peer")
	}

	if tmr.msg != "Test protocol message" {
		t.Fatalf("message was not delivered to a peer")
	}
}

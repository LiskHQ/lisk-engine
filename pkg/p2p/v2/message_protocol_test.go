package p2p_v2

import (
	"context"
	"testing"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

func TestNewMessageProtocol(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	p, _ := Create(logger, context.Background())

	mp := NewMessageProtocol(p)
	if mp.peer != p {
		t.Fatalf("unexpected peer, got %v, want %v", mp.peer.ID(), p.ID())
	}
}

func TestSendMessage(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	tmr := TestMessageReceive{done: make(chan any)}

	p1, _ := Create(logger, context.Background())
	p2, _ := Create(logger, context.Background())
	p2.host.SetStreamHandler(messageProtocolID, tmr.onMessageReceive)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	_ = p1.Connect(*p2AddrInfo)
	err := p1.SendMessage(p2.ID(), "Test protocol message")
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

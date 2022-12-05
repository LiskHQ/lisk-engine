package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/stretchr/testify/assert"
)

func TestNewMessageProtocol(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	p, _ := NewPeer(context.Background(), logger, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)

	mp := NewMessageProtocol(p)
	assert.Equal(t, p, mp.peer)
}

func TestSendMessage(t *testing.T) {
	logger, _ := log.NewDefaultProductionLogger()
	tmr := TestMessageReceive{done: make(chan any)}

	p1, _ := NewPeer(context.Background(), logger, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	p2, _ := NewPeer(context.Background(), logger, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, PeerSecurityTLS)
	p2.host.SetStreamHandler(messageProtocolID, tmr.onMessageReceive)
	p2Addrs, _ := p2.P2PAddrs()
	p2AddrInfo, _ := PeerInfoFromMultiAddr(p2Addrs[0].String())

	_ = p1.Connect(context.Background(), *p2AddrInfo)
	err := p1.SendMessage(context.Background(), p2.ID(), "Test protocol message")
	assert.Nil(t, err)

	select {
	case <-tmr.done:
		break
	case <-time.After(time.Second * pingTimeout):
		t.Fatalf("timeout occurs, message was not delivered to a peer")
	}

	assert.Contains(t, tmr.msg, "Test protocol message")
}

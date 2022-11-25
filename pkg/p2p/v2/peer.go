package p2p_v2

import (
	"context"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

const numOfPingMessages = 5 // Number of sent ping messages in Ping service.
const pingTimeout = 5       // Ping service timeout in seconds.

// Peer type - a p2p node.
type Peer struct {
	logger log.Logger
	ctx    context.Context
	host   host.Host
	*MessageProtocol
}

// Create a peer with a libp2p host and message protocol.
func Create(logger log.Logger, ctx context.Context) (*Peer, error) {
	host, err := libp2p.New(
		// Multiple listen addresses
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"),
		// Support TLS connections
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		// Support Noise connections
		libp2p.Security(noise.ID, noise.New),
		// Support any other default transports (TCP, QUIC, WS)
		libp2p.DefaultTransports)
	if err != nil {
		return nil, err
	}
	p := &Peer{logger: logger, ctx: ctx, host: host}
	p.MessageProtocol = NewMessageProtocol(p)
	p.logger.Infof("peer successfully created")
	return p, nil
}

// Stop a peer.
func (p *Peer) Stop() error {
	err := p.host.Close()
	if err != nil {
		return err
	}
	p.logger.Infof("peer successfully stopped")
	return nil
}

// Connect to a peer.
func (p *Peer) Connect(peer peer.AddrInfo) error {
	return p.host.Connect(p.ctx, peer)
}

// ID returns a peers's identifier.
func (p Peer) ID() peer.ID {
	return p.host.ID()
}

// Addrs returns a peers's listen addresses.
func (p Peer) Addrs() []ma.Multiaddr {
	return p.host.Addrs()
}

// P2PAddrs returns a peers's listen addresses in multiaddress format.
func (p Peer) P2PAddrs() ([]ma.Multiaddr, error) {
	peerInfo := peer.AddrInfo{
		ID:    p.ID(),
		Addrs: p.Addrs(),
	}
	return peer.AddrInfoToP2pAddrs(&peerInfo)
}

// ConnectedPeers returns a list of all connected peers.
func (p Peer) ConnectedPeers() []peer.ID {
	return p.host.Network().Peers()
}

// Ping a peer.
func (p Peer) Ping(peer peer.ID) (rtt []time.Duration, err error) {
	ps := ping.NewPingService(p.host)
	ch := ps.Ping(p.ctx, peer)

	p.logger.Infof("sending %d ping messages to %v", numOfPingMessages, peer)
	for i := 0; i < numOfPingMessages; i++ {
		select {
		case pingRes := <-ch:
			if pingRes.Error != nil {
				return rtt, pingRes.Error
			}
			p.logger.Infof("pinged %v in %v", peer, pingRes.RTT)
			rtt = append(rtt, pingRes.RTT)
		case <-p.ctx.Done():
			return rtt, errors.New("ping canceled")
		case <-time.After(time.Second * pingTimeout):
			return rtt, errors.New("ping timeout")
		}
	}
	return rtt, nil
}

// sendProtoMessage sends a message to a peer using a stream.
func (p *Peer) sendProtoMessage(id peer.ID, pId protocol.ID, msg string) error {
	s, err := p.host.NewStream(p.ctx, id, pId)
	if err != nil {
		return err
	}
	defer s.Close()

	n, err := s.Write([]byte(msg))
	if err != nil {
		return err
	}
	if n != len([]byte(msg)) {
		return errors.New("error while sending a message, did not sent a whole message")
	}

	return nil
}

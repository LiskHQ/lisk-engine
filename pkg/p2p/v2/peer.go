package p2p

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

// Peer security option type.
type PeerSecurityOption uint8

const (
	PeerSecurityNone  PeerSecurityOption = iota // Do not support any security.
	PeerSecurityTLS                             // Support TLS connections.
	PeerSecurityNoise                           // Support Noise connections.
)

// Peer type - a p2p node.
type Peer struct {
	logger log.Logger
	host   host.Host
	*MessageProtocol
}

// NewPeer creates a peer with a libp2p host and message protocol.
func NewPeer(ctx context.Context, logger log.Logger, addrs []string, security PeerSecurityOption) (*Peer, error) {
	opts := []libp2p.Option{
		// Support default transports (TCP, QUIC, WS)
		libp2p.DefaultTransports,
	}

	if len(addrs) == 0 {
		opts = append(opts, libp2p.NoListenAddrs)
	} else {
		opts = append(opts, libp2p.ListenAddrStrings(addrs...))
	}

	switch security {
	case PeerSecurityNone:
		opts = append(opts, libp2p.NoSecurity)
	case PeerSecurityTLS:
		opts = append(opts, libp2p.Security(libp2ptls.ID, libp2ptls.New))
	case PeerSecurityNoise:
		opts = append(opts, libp2p.Security(noise.ID, noise.New))
	default:
		opts = append(opts, libp2p.NoSecurity)
	}

	host, err := libp2p.New(opts...)
	if err != nil {
		return nil, err
	}
	p := &Peer{logger: logger, host: host}
	p.MessageProtocol = NewMessageProtocol(p)
	p.logger.Infof("peer successfully created")
	return p, nil
}

// Close a peer.
func (p *Peer) Close() error {
	err := p.host.Close()
	if err != nil {
		return err
	}
	p.logger.Infof("peer successfully stopped")
	return nil
}

// Connect to a peer.
func (p *Peer) Connect(ctx context.Context, peer peer.AddrInfo) error {
	return p.host.Connect(ctx, peer)
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

// PingMultiTimes tries to send ping request to a peer for five times.
func (p Peer) PingMultiTimes(ctx context.Context, peer peer.ID) (rtt []time.Duration, err error) {
	pingService := ping.NewPingService(p.host)
	ch := pingService.Ping(ctx, peer)

	p.logger.Infof("sending %d ping messages to %v", numOfPingMessages, peer)
	for i := 0; i < numOfPingMessages; i++ {
		select {
		case pingRes := <-ch:
			if pingRes.Error != nil {
				return rtt, pingRes.Error
			}
			p.logger.Infof("pinged %v in %v", peer, pingRes.RTT)
			rtt = append(rtt, pingRes.RTT)
		case <-ctx.Done():
			return rtt, errors.New("ping canceled")
		case <-time.After(time.Second * pingTimeout):
			return rtt, errors.New("ping timeout")
		}
	}
	return rtt, nil
}

// Ping tries to send a ping request to a peer.
func (p Peer) Ping(ctx context.Context, peer peer.ID) (rtt time.Duration, err error) {
	pingService := ping.NewPingService(p.host)
	ch := pingService.Ping(ctx, peer)

	p.logger.Infof("sending ping messages to %v", peer)
	select {
	case pingRes := <-ch:
		if pingRes.Error != nil {
			return rtt, pingRes.Error
		}
		p.logger.Infof("pinged %v in %v", peer, pingRes.RTT)
		return pingRes.RTT, nil
	case <-ctx.Done():
		return rtt, errors.New("ping canceled")
	case <-time.After(time.Second * pingTimeout):
		return rtt, errors.New("ping timeout")
	}
}

// sendProtoMessage sends a message to a peer using a stream.
func (p *Peer) sendProtoMessage(ctx context.Context, id peer.ID, pId protocol.ID, msg string) error {
	s, err := p.host.NewStream(ctx, id, pId)
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

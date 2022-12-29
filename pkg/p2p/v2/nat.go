package p2p

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/p2p/host/autonat"
)

// newAutoNAT creates a new AutoNAT service.
func newAutoNAT(p *Peer) (autonat.AutoNAT, error) {
	dialback, err := libp2p.New(libp2p.NoListenAddrs)
	if err != nil {
		return nil, err
	}
	nat, err := autonat.New(
		p.host,
		autonat.EnableService(dialback.Network()),
		autonat.WithSchedule(90*time.Second, 15*time.Minute),
		autonat.WithThrottling(30, 1*time.Minute),
		autonat.WithPeerThrottling(3),
	)
	if err != nil {
		return nil, err
	}
	return nat, nil
}

// natTraversalService handles all NAT traversal related events.
func natTraversalService(ctx context.Context, wg *sync.WaitGroup, conf Config, mp *MessageProtocol) {
	defer wg.Done()
	mp.peer.logger.Infof("NAT traversal service started")

	var nat autonat.AutoNAT
	var err error
	if conf.DummyConfigurationFeatureEnable {
		if nat, err = newAutoNAT(mp.peer); err != nil {
			mp.peer.logger.Errorf("Failed to create and enable peer AutoNAT feature: %v", err)
		}
		mp.peer.logger.Debugf("NAT status: %v", nat.Status())
	}

	sub, err := mp.peer.host.EventBus().Subscribe([]interface{}{new(event.EvtLocalReachabilityChanged), new(event.EvtNATDeviceTypeChanged)})
	if err != nil {
		mp.peer.logger.Errorf("Failed to subscribe to bus events: %v", err)
	}

	t := time.NewTicker(10 * time.Second)

	for {
		select {
		// TODO - remove this timer event after testing
		case <-t.C:
			if conf.DummyConfigurationFeatureEnable {
				mp.peer.logger.Debugf("NAT status: %v", nat.Status())
			}
			addrs, _ := mp.peer.P2PAddrs()
			mp.peer.logger.Debugf("My listen addresses: %v", addrs)
			for _, connectedPeer := range mp.peer.ConnectedPeers() {
				ch := make(chan *ResponseMsg, 1)
				err = mp.SendRequestMessage(ctx, connectedPeer, MessageRequestTypeKnownPeers, nil, ch)
				if err != nil {
					mp.peer.logger.Errorf("Failed to send message to peer %v: %v", connectedPeer, err)
				}
				response := <-ch
				mp.peer.logger.Debugf("Received response from peer %v: %+v", connectedPeer, response)
			}
			t.Reset(10 * time.Second)
		case e := <-sub.Out():
			if ev, ok := e.(event.EvtLocalReachabilityChanged); ok {
				mp.peer.logger.Debugf("New NAT event received. Local reachability changed: %v", ev.Reachability.String())
				if ev.Reachability == network.ReachabilityPublic {
					mp.peer.logger.Debugf("We are publicly reachable and are not behind NAT")
				} else if ev.Reachability == network.ReachabilityPrivate {
					mp.peer.logger.Debugf("We are not publicly reachable and are behind NAT")
				}
			}
			if ev, ok := e.(event.EvtNATDeviceTypeChanged); ok {
				mp.peer.logger.Debugf("New NAT event received. NAT device type changed: %v", ev)
			}
		case <-ctx.Done():
			mp.peer.logger.Infof("NAT traversal service stopped")
			return
		}
	}
}

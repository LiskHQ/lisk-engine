package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/log"
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
func natTraversalService(ctx context.Context, wg *sync.WaitGroup, logger log.Logger, conf Config, p *Peer) {
	defer wg.Done()
	logger.Infof("NAT traversal service started")

	var nat autonat.AutoNAT
	var err error
	if conf.DummyConfigurationFeatureEnable {
		if nat, err = newAutoNAT(p); err != nil {
			logger.Errorf("Failed to create and enable peer AutoNAT feature: %v", err)
		}
		p.logger.Debugf("NAT status: %v", nat.Status())
	}

	sub, err := p.host.EventBus().Subscribe([]interface{}{new(event.EvtLocalReachabilityChanged), new(event.EvtNATDeviceTypeChanged)})
	if err != nil {
		logger.Errorf("Failed to subscribe to bus events: %v", err)
	}

	t := time.NewTicker(10 * time.Second)

	for {
		select {
		// TODO - remove this timer event after testing
		case <-t.C:
			if conf.DummyConfigurationFeatureEnable {
				p.logger.Debugf("NAT status: %v", nat.Status())
			}
			addrs, _ := p.P2PAddrs()
			p.logger.Debugf("My listen addresses: %v", addrs)
			for _, connectedPeer := range p.ConnectedPeers() {
				err = p.SendMessage(ctx, connectedPeer, fmt.Sprintf("Sending message to %v", connectedPeer))
				if err != nil {
					p.logger.Errorf("Failed to send message to peer %v: %v", connectedPeer, err)
				}
			}
			t.Reset(10 * time.Second)
		case e := <-sub.Out():
			if ev, ok := e.(event.EvtLocalReachabilityChanged); ok {
				p.logger.Debugf("New NAT event received. Local reachability changed: %v", ev.Reachability.String())
				if ev.Reachability == network.ReachabilityPublic {
					logger.Debugf("We are publicly reachable and are not behind NAT")
				} else if ev.Reachability == network.ReachabilityPrivate {
					logger.Debugf("We are not publicly reachable and are behind NAT")
				}
			}
			if ev, ok := e.(event.EvtNATDeviceTypeChanged); ok {
				p.logger.Debugf("New NAT event received. NAT device type changed: %v", ev)
			}
		case <-ctx.Done():
			logger.Infof("NAT traversal service stopped")
			return
		}
	}
}

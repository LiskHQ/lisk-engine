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

// natService type - a NAT traversal service.
type natService struct {
	autoNAT autonat.AutoNAT
}

// enableAutoNAT enables AutoNAT service.
func (n *natService) enableAutoNAT(p *Peer) error {
	dialback, err := libp2p.New(libp2p.NoListenAddrs)
	if err != nil {
		return err
	}
	an, err := autonat.New(
		p.host,
		autonat.EnableService(dialback.Network()),
		autonat.WithSchedule(90*time.Second, 15*time.Minute),
		autonat.WithThrottling(30, 1*time.Minute),
		autonat.WithPeerThrottling(3),
	)
	if err != nil {
		return err
	}
	n.autoNAT = an
	return nil
}

// status returns a current NAT status.
func (n *natService) status() network.Reachability {
	return n.autoNAT.Status()
}

// natTraversalService handles all NAT traversal related events.
func natTraversalService(ctx context.Context, wg *sync.WaitGroup, logger log.Logger, conf Config, p *Peer) {
	defer wg.Done()
	logger.Infof("NAT traversal service started")

	nat := &natService{}
	if conf.DummyConfigurationFeatureEnable {
		if err := nat.enableAutoNAT(p); err != nil {
			logger.Errorf("Failed to enable peer AutoNAT feature: %v", err)
		}
		p.logger.Infof("NAT status: %v", nat.status())
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
				p.logger.Infof("NAT status: %v", nat.status())
			}
			addrs, _ := p.P2PAddrs()
			p.logger.Infof("My listen addresses: %v", addrs)
			for _, connectedPeer := range p.ConnectedPeers() {
				err = p.SendMessage(ctx, connectedPeer, fmt.Sprintf("Sending message to %v", connectedPeer))
				if err != nil {
					p.logger.Errorf("Failed to send message to peer %v: %v", connectedPeer, err)
				}
			}
			t.Reset(10 * time.Second)
		case e := <-sub.Out():
			if ev, ok := e.(event.EvtLocalReachabilityChanged); ok {
				p.logger.Infof("New NAT event received. Local reachability changed: %v", ev.Reachability.String())
				if ev.Reachability == network.ReachabilityPublic {
					logger.Infof("We are publicly reachable and are not behind NAT")
				} else if ev.Reachability == network.ReachabilityPrivate {
					logger.Infof("We are not publicly reachable and are behind NAT")
				}
			}
			if ev, ok := e.(event.EvtNATDeviceTypeChanged); ok {
				p.logger.Infof("New NAT event received. NAT device type changed: %v", ev)
			}
		case <-ctx.Done():
			logger.Infof("NAT traversal service stopped")
			return
		}
	}
}

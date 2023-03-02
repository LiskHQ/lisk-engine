// nat runs P2P client against specified node for debugging and testing NAT functionality of libp2p library.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/engine/config"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	p2pLib "github.com/LiskHQ/lisk-engine/pkg/p2p"
)

const (
	TopicTransactions = "transactions"
	TopicBlocks       = "blocks"
	TopicEvents       = "events"
)

var Topics = []string{TopicTransactions, TopicBlocks, TopicEvents}

func main() {
	logger, err := log.NewDefaultProductionLogger()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	cfgNet := config.NetworkConfig{
		AllowIncomingConnections: true,
		EnableNATService:         true,
		EnableUsingRelayService:  true,
		EnableRelayService:       true,
		EnableHolePunching:       true,
		Version:                  "1.0",
		ChainID:                  []byte{0x04, 0x00, 0x01, 0x02},
	}
	err = cfgNet.InsertDefault()
	if err != nil {
		panic(err)
	}

	p2p := p2pLib.NewP2P(&cfgNet)

	for _, topic := range Topics {
		err = p2p.RegisterEventHandler(topic, func(event *p2pLib.Event) {
			logger.Infof("Received event: %v", event)
			logger.Infof("PeerID: %v", event.PeerID())
			logger.Infof("Event: %v", event.Event())
			logger.Infof("Data: %s", string(event.Data()))
		}, nil)
		if err != nil {
			panic(err)
		}
	}

	err = p2p.RegisterEventHandler("testEventName", func(event *p2pLib.Event) {
		logger.Infof("Received event: %v", event)
		logger.Infof("PeerID: %v", event.PeerID())
		logger.Infof("Event: %v", event.Event())
		logger.Infof("Data: %s", string(event.Data()))
	}, nil)
	if err != nil {
		panic(err)
	}

	err = p2p.RegisterRPCHandler("ping", func(w p2pLib.ResponseWriter, req *p2pLib.RequestMsg) {
		rtt, err := p2p.PingMultiTimes(ctx, p2p.ConnectedPeers()[0])
		if err != nil {
			panic(err)
		}
		var sum time.Duration
		for _, i := range rtt {
			sum += i
		}
		avg := time.Duration(float64(sum) / float64(len(rtt)))

		w.Write([]byte(fmt.Sprintf("Average RTT with you: %v", avg)))
	})
	if err != nil {
		panic(err)
	}

	err = p2p.RegisterRPCHandler("knownPeers", func(w p2pLib.ResponseWriter, req *p2pLib.RequestMsg) {
		peers := p2p.ConnectedPeers()
		w.Write([]byte(fmt.Sprintf("All known peers: %v", peers)))
	})
	if err != nil {
		panic(err)
	}

	err = p2p.Start(logger, crypto.RandomBytes(32))
	if err != nil {
		panic(err)
	}

	addrs, err := p2p.P2PAddrs()
	if err != nil {
		panic(err)
	}
	logger.Infof("libp2p node addresses: %v", addrs)

	// if a remote peer has been passed on the command line, connect to it
	// and send ping request message, otherwise wait for a signal to stop
	if len(os.Args) > 1 {
		peer, err := p2pLib.PeerInfoFromMultiAddr(os.Args[1])
		if err != nil {
			panic(err)
		}
		if err := p2p.Connect(ctx, *peer); err != nil {
			panic(err)
		}
		response, err := p2p.SendRequestMessage(ctx, peer.ID, "ping", nil)
		if err != nil {
			panic(err)
		}
		logger.Infof("Response message received: %+v", response)
		logger.Infof("%s", string(response.Data()))
	}

	// Start demo routine
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go demoRoutine(ctx, logger, wg, p2p)

	// wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	logger.Infof("Received signal, shutting down a node...")

	// Stop demo routine
	cancel()
	wg.Wait()

	// Stop P2P
	err = p2p.Stop()
	if err != nil {
		panic(err)
	}
}

// demoRoutine starts the demo routine which will publish messages to the network.
func demoRoutine(ctx context.Context, logger log.Logger, wg *sync.WaitGroup, p2p *p2pLib.P2P) {
	defer wg.Done()
	logger.Infof("Demo routine started")

	t := time.NewTicker(10 * time.Second)
	var counter = 0

	for {
		select {
		case <-t.C:
			topicTransactions := "transactions" // Test topic which will be removed after testing
			topicBlocks := "blocks"             // Test topic which will be removed after testing
			topicEvents := "events"             // Test topic which will be removed after testing
			data := []byte(fmt.Sprintf("Timer for %s is running and this is a test transaction message: %v", p2p.Peer.ID().String(), counter))
			err := p2p.Publish(ctx, topicTransactions, data)
			if err != nil {
				logger.Errorf("Error while publishing message: %s", err)
			}
			data = []byte(fmt.Sprintf("Timer for %s is running and this is a test block message: %v", p2p.Peer.ID().String(), counter))
			err = p2p.Publish(ctx, topicBlocks, data)
			if err != nil {
				logger.Errorf("Error while publishing message: %s", err)
			}
			data = []byte(fmt.Sprintf("Timer for %s is running and this is a test event message: %v", p2p.Peer.ID().String(), counter))
			err = p2p.Publish(ctx, topicEvents, data)
			if err != nil {
				logger.Errorf("Error while publishing message: %s", err)
			}
			counter++

			logger.Debugf("List of connected peers: %v", p2p.ConnectedPeers())
			logger.Debugf("List of known peers: %v", p2p.KnownPeers())
			logger.Debugf("List of blacklisted peers: %v", p2p.BlacklistedPeers())

			addrs, _ := p2p.P2PAddrs()
			logger.Debugf("My listen addresses: %v", addrs)
			for _, connectedPeer := range p2p.ConnectedPeers() {
				response, err := p2p.SendRequestMessage(ctx, connectedPeer, "knownPeers", nil)
				if err != nil {
					logger.Errorf("Failed to send message to peer %v: %v", connectedPeer, err)
				} else {
					logger.Debugf("Received response from peer %v: %v", connectedPeer, string(response.Data()))
				}
			}

			t.Reset(10 * time.Second)
		case <-ctx.Done():
			logger.Infof("Demo routine stopped")
			return
		}
	}
}

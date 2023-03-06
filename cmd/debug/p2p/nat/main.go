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
	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"
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

	cfgNet := p2p.Config{
		AllowIncomingConnections: true,
		EnableNATService:         true,
		EnableUsingRelayService:  true,
		EnableRelayService:       true,
		EnableHolePunching:       true,
		Version:                  "1.0",
		ChainID:                  []byte{0x04, 0x00, 0x01, 0x02},
	}

	conn := p2pLib.NewConnection(&cfgNet)

	for _, topic := range Topics {
		err = conn.RegisterEventHandler(topic, func(event *p2pLib.Event) {
			logger.Infof("Received event: %v", event)
			logger.Infof("PeerID: %v", event.PeerID())
			logger.Infof("Event: %v", event.Topic())
			logger.Infof("Data: %s", string(event.Data()))
		}, nil)
		if err != nil {
			panic(err)
		}
	}

	err = conn.RegisterEventHandler("testEventName", func(event *p2pLib.Event) {
		logger.Infof("Received event: %v", event)
		logger.Infof("PeerID: %v", event.PeerID())
		logger.Infof("Event: %v", event.Topic())
		logger.Infof("Data: %s", string(event.Data()))
	}, nil)
	if err != nil {
		panic(err)
	}

	err = conn.RegisterRPCHandler("ping", func(w p2pLib.ResponseWriter, req *p2pLib.Request) {
		rtt, err := conn.PingMultiTimes(ctx, conn.ConnectedPeers()[0])
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

	err = conn.RegisterRPCHandler("knownPeers", func(w p2pLib.ResponseWriter, req *p2pLib.Request) {
		peers := conn.ConnectedPeers()
		w.Write([]byte(fmt.Sprintf("All known peers: %v", peers)))
	})
	if err != nil {
		panic(err)
	}

	err = conn.Start(logger, crypto.RandomBytes(32))
	if err != nil {
		panic(err)
	}

	addrs, err := conn.MultiAddress()
	if err != nil {
		panic(err)
	}
	logger.Infof("libp2p node addresses: %v", addrs)

	// if a remote peer has been passed on the command line, connect to it
	// and send ping request message, otherwise wait for a signal to stop
	if len(os.Args) > 1 {
		peer, err := p2pLib.AddrInfoFromMultiAddr(os.Args[1])
		if err != nil {
			panic(err)
		}
		if err := conn.Connect(ctx, *peer); err != nil {
			panic(err)
		}
		response := conn.RequestFrom(ctx, peer.ID.String(), "ping", nil)
		if response.Error() != nil {
			panic(err)
		}
		logger.Infof("Response message received: %+v", response)
		logger.Infof("%s", string(response.Data()))
	}

	// Start demo routine
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go demoRoutine(ctx, logger, wg, conn)

	// wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	logger.Infof("Received signal, shutting down a node...")

	// Stop demo routine
	cancel()
	wg.Wait()

	// Stop P2P
	err = conn.Stop()
	if err != nil {
		panic(err)
	}
}

// demoRoutine starts the demo routine which will publish messages to the network.
func demoRoutine(ctx context.Context, logger log.Logger, wg *sync.WaitGroup, p2p *p2pLib.Connection) {
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
			logger.Debugf("List of blacklisted peers: %v", p2p.BlacklistedPeers())

			addrs, _ := p2p.MultiAddress()
			logger.Debugf("My listen addresses: %v", addrs)
			for _, connectedPeer := range p2p.ConnectedPeers() {
				response := p2p.RequestFrom(ctx, connectedPeer.String(), "knownPeers", nil)
				if response.Error() != nil {
					logger.Errorf("Failed to send message to peer %v: %v", connectedPeer, response.Error())
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

// nat runs P2P client against specified node for debugging and testing NAT functionality of libp2p library.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	defer cancel()

	config := p2pLib.Config{
		AllowIncomingConnections: true,
		EnableNATService:         true,
		EnableUsingRelayService:  true,
		EnableRelayService:       true,
		EnableHolePunching:       true,
	}
	err = config.InsertDefault()
	if err != nil {
		panic(err)
	}

	p2p := p2pLib.NewP2P(config)

	for _, topic := range Topics {
		err = p2p.RegisterEventHandler(topic, func(event *p2pLib.Event) {
			logger.Infof("Received event: %v", event)
			logger.Infof("PeerID: %v", event.PeerID())
			logger.Infof("Event: %v", event.Event())
			logger.Infof("Data: %s", string(event.Data()))
		})
		if err != nil {
			panic(err)
		}
	}

	err = p2p.RegisterEventHandler("testEventName", func(event *p2pLib.Event) {
		logger.Infof("Received event: %v", event)
		logger.Infof("PeerID: %v", event.PeerID())
		logger.Infof("Event: %v", event.Event())
		logger.Infof("Data: %s", string(event.Data()))
	})
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

	err = p2p.RegisterRPCHandler("connectedPeers", func(w p2pLib.ResponseWriter, req *p2pLib.RequestMsg) {
		peers := p2p.ConnectedPeers()
		w.Write([]byte(fmt.Sprintf("All connected peers: %v", peers)))
	})
	if err != nil {
		panic(err)
	}

	err = p2p.Start(logger)
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

	// wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	logger.Infof("Received signal, shutting down a node...")

	err = p2p.Stop()
	if err != nil {
		panic(err)
	}
}

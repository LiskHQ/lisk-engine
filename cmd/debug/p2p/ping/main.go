// p2p runs P2P client against specified node for debugging and demonstration purpose.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/engine/config"
	"github.com/LiskHQ/lisk-engine/pkg/log"
	p2p "github.com/LiskHQ/lisk-engine/pkg/p2p"
)

func main() {
	logger, err := log.NewDefaultProductionLogger()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := config.NetworkConfig{
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

	wg := &sync.WaitGroup{}
	node, err := p2p.NewPeer(ctx, wg, logger, config)
	if err != nil {
		panic(err)
	}

	mp := p2p.NewMessageProtocol()
	err = mp.RegisterRPCHandler("ping", func(w p2p.ResponseWriter, req *p2p.RequestMsg) {
		rtt, err := node.PingMultiTimes(ctx, node.ConnectedPeers()[0])
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
	mp.Start(ctx, logger, node)

	addrs, err := node.P2PAddrs()
	if err != nil {
		panic(err)
	}
	logger.Infof("libp2p node addresses: %v", addrs)

	// if a remote peer has been passed on the command line, connect to it
	// and send ping request message, otherwise wait for a signal to stop
	if len(os.Args) > 1 {
		peer, err := p2p.PeerInfoFromMultiAddr(os.Args[1])
		if err != nil {
			panic(err)
		}
		if err := node.Connect(ctx, *peer); err != nil {
			panic(err)
		}
		response, err := mp.SendRequestMessage(ctx, peer.ID, "ping", nil)
		if err != nil {
			panic(err)
		}
		logger.Infof("Response message received: %+v", response)
		logger.Infof("%s", string(response.Data()))
	} else {
		// wait for a SIGINT or SIGTERM signal
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		logger.Infof("Received signal, shutting down a node...")
	}

	if err := node.Close(); err != nil {
		panic(err)
	}
}

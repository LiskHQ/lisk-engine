// p2p runs P2P client against specified node for debugging and demonstration purpose.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
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

	cfgNet := config.NetworkConfig{
		AllowIncomingConnections: true,
		EnableNATService:         true,
		EnableUsingRelayService:  true,
		EnableRelayService:       true,
		EnableHolePunching:       true,
	}
	err = cfgNet.InsertDefault()
	if err != nil {
		panic(err)
	}

	node := p2p.NewP2P(&cfgNet)

	if err := node.RegisterRPCHandler("ping", func(w p2p.ResponseWriter, req *p2p.Request) {
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
	}); err != nil {
		panic(err)
	}

	if err := node.Start(log.DefaultLogger, nil); err != nil {
		panic(err)
	}

	addrs, err := node.MultiAddress()
	if err != nil {
		panic(err)
	}
	logger.Infof("libp2p node addresses: %v", addrs)

	// if a remote peer has been passed on the command line, connect to it
	// and send ping request message, otherwise wait for a signal to stop
	if len(os.Args) > 1 {
		peer, err := p2p.AddrInfoFromMultiAddr(os.Args[1])
		if err != nil {
			panic(err)
		}
		if err := node.Connect(ctx, *peer); err != nil {
			panic(err)
		}
		response := node.RequestFrom(ctx, peer.ID.String(), "ping", nil)
		logger.Infof("Response message received: %+v", response)
		logger.Infof("%s", string(response.Data()))
	} else {
		// wait for a SIGINT or SIGTERM signal
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		logger.Infof("Received signal, shutting down a node...")
	}

	if err := node.Stop(); err != nil {
		panic(err)
	}
}

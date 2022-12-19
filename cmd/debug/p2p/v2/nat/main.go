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
	p2pLib "github.com/LiskHQ/lisk-engine/pkg/p2p/v2"
)

func main() {
	logger, err := log.NewDefaultProductionLogger()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p2p := p2pLib.NewP2P()
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
	// and send it 5 ping messages, otherwise wait for a signal to stop
	if len(os.Args) > 1 {
		peer, err := p2pLib.PeerInfoFromMultiAddr(os.Args[1])
		if err != nil {
			panic(err)
		}
		if err := p2p.Connect(ctx, *peer); err != nil {
			panic(err)
		}
		rtt, err := p2p.PingMultiTimes(ctx, peer.ID)
		if err != nil {
			panic(err)
		}

		var sum time.Duration
		for _, i := range rtt {
			sum += i
		}
		avg := time.Duration(float64(sum) / float64(len(rtt)))
		err = p2p.SendMessage(ctx, peer.ID, fmt.Sprintf("Average RTT with you: %v", avg))
		if err != nil {
			panic(err)
		}
		time.Sleep(10 * time.Millisecond) // Wait for a message to be delivered.
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

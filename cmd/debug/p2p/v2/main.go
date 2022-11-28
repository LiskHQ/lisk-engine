// p2p runs P2P client against specified node for debugging and demonstration purpose.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	p2p "github.com/LiskHQ/lisk-engine/pkg/p2p/v2"
)

func main() {
	logger, err := log.NewDefaultProductionLogger()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node, err := p2p.NewPeer(ctx, logger, []string{"/ip4/0.0.0.0/tcp/0", "/ip4/0.0.0.0/udp/0/quic"}, p2p.PeerSecurityTLS)
	if err != nil {
		panic(err)
	}

	addrs, err := node.P2PAddrs()
	if err != nil {
		panic(err)
	}
	logger.Infof("libp2p node addresses: %v", addrs)

	// if a remote peer has been passed on the command line, connect to it
	// and send it 5 ping messages, otherwise wait for a signal to stop
	if len(os.Args) > 1 {
		peer, err := p2p.PeerInfoFromMultiAddr(os.Args[1])
		if err != nil {
			panic(err)
		}
		if err := node.Connect(*peer); err != nil {
			panic(err)
		}
		rtt, err := node.Ping(peer.ID)
		if err != nil {
			panic(err)
		}

		var sum time.Duration
		for _, i := range rtt {
			sum += i
		}
		avg := time.Duration(float64(sum) / float64(len(rtt)))
		err = node.SendMessage(peer.ID, fmt.Sprintf("Average RTT with you: %v", avg))
		if err != nil {
			panic(err)
		}
		time.Sleep(time.Millisecond * 10) // Wait for a message to be delivered.
	} else {
		// wait for a SIGINT or SIGTERM signal
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		logger.Infof("received signal, shutting down a node...")
	}

	if err := node.Close(); err != nil {
		panic(err)
	}
}

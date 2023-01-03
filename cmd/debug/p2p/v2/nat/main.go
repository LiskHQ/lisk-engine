// nat runs P2P client against specified node for debugging and testing NAT functionality of libp2p library.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

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
	// and send ping request message, otherwise wait for a signal to stop
	if len(os.Args) > 1 {
		peer, err := p2pLib.PeerInfoFromMultiAddr(os.Args[1])
		if err != nil {
			panic(err)
		}
		if err := p2p.Connect(ctx, *peer); err != nil {
			panic(err)
		}
		response, err := p2p.SendRequestMessage(ctx, peer.ID, p2pLib.MessageRequestTypePing, nil)
		if err != nil {
			panic(err)
		}
		logger.Infof("Response message received: %+v", response)
		logger.Infof("%s", response.Data)
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

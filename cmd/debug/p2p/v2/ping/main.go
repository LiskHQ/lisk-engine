// p2p runs P2P client against specified node for debugging and demonstration purpose.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

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

	config := p2p.P2PConfig{
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

	node, err := p2p.NewPeer(ctx, logger, config)
	if err != nil {
		panic(err)
	}

	mp := p2p.NewMessageProtocol(ctx, logger, node)

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
		response, err := mp.SendRequestMessage(ctx, peer.ID, p2p.MessageRequestTypePing, nil)
		if err != nil {
			panic(err)
		}
		logger.Infof("Response message received: %+v", response)
		logger.Infof("%s", response.Data)
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

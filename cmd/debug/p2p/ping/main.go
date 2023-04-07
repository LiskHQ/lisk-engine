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
	p2p "github.com/LiskHQ/lisk-engine/pkg/p2p"
)

func main() {
	logger, err := log.NewDefaultProductionLogger()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ip4quic := "/ip4/0.0.0.0/udp/0/quic"
	ip4tcp := "/ip4/0.0.0.0/tcp/0"

	cfg := p2p.Config{
		Addresses:               []string{ip4quic, ip4tcp},
		EnableNATService:        true,
		EnableUsingRelayService: true,
		EnableRelayService:      true,
		EnableHolePunching:      true,
		Version:                 "1.0",
		ChainID:                 []byte{0x04, 0x00, 0x01, 0x02},
	}

	conn := p2p.NewConnection(logger, &cfg)

	if err := conn.RegisterRPCHandler("ping", func(w p2p.ResponseWriter, req *p2p.Request) {
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
	}); err != nil {
		panic(err)
	}

	if err := conn.Start(nil); err != nil {
		panic(err)
	}

	// if a remote peer has been passed on the command line, connect to it
	// and send ping request message, otherwise wait for a signal to stop
	if len(os.Args) > 1 {
		peer, err := p2p.AddrInfoFromMultiAddr(os.Args[1])
		if err != nil {
			panic(err)
		}
		if err := conn.Connect(ctx, *peer); err != nil {
			panic(err)
		}
		response := conn.RequestFrom(ctx, peer.ID, "ping", nil)
		logger.Infof("Response message received: %+v", response)
		logger.Infof("%s", string(response.Data()))
	} else {
		// wait for a SIGINT or SIGTERM signal
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		logger.Infof("Received signal, shutting down a node...")
	}

	if err := conn.Stop(); err != nil {
		panic(err)
	}
}

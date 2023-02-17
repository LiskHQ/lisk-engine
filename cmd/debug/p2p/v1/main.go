// p2p runs P2P client against specified node for debugging.
//
//nolint:all
package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"
	"github.com/LiskHQ/lisk-engine/pkg/p2p/v1/addressbook"
)

func main() {
	config := &p2p.ConnectionConfig{
		SeedAddresses: []*addressbook.Address{
			addressbook.NewAddress("127.0.0.1", 5001),
		},
		MaxInboundConnection:  100,
		MaxOutboundConnection: 20,
	}
	debugLoger := log.DefaultLogger.With("pprof", "cpu")
	go func() {
		debugLoger.Debug("pprof", http.ListenAndServe("localhost:6060", nil))
	}()
	conn := p2p.NewConnection(config)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	fmt.Println("Start")
	go conn.Start(log.DefaultLogger.With("module", "p2p"), &p2p.NodeInfo{
		Port:           4000,
		ChainID:        "01e47ba4e3e57981642150f4b45f64c2160c10bac9434339888210a4fa5df097",
		NetworkVersion: "2.0",
		Nonce:          "O2wTkjqplHIIabab",
	})
	fmt.Println("runnng", runtime.NumGoroutine())
	<-c
	conn.Stop()
	fmt.Println("stopped", runtime.NumGoroutine())
	timer := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-timer.C:
			fmt.Println(runtime.NumGoroutine())
		case <-c:
			return
		}
	}
}

package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	p2p "github.com/LiskHQ/lisk-engine/pkg/p2p"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	tgSync "github.com/testground/sdk-go/sync"
	"golang.org/x/sync/errgroup"
)

var testcases = map[string]interface{}{
	"ping": run.InitializedTestCaseFn(runPing),
}

func main() {
	run.InvokeMap(testcases)
}

func runPing(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	var (
		secureChannel = runenv.StringParam("secure_channel")
		maxLatencyMs  = runenv.IntParam("max_latency_ms")
		iterations    = runenv.IntParam("iterations")
	)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	initCtx.MustWaitAllInstancesInitialized(ctx)
	ip := initCtx.NetClient.MustGetDataNetworkIP()

	config := p2p.Config{
		AllowIncomingConnections: true,
		ConnectionSecurity:       secureChannel,
		Addresses:                []string{fmt.Sprintf("/ip4/%s/tcp/0", ip)},
	}
	err := config.InsertDefault()
	if err != nil {
		panic(err)
	}
	runenv.RecordMessage("started test instance; params: secure_channel=%s, max_latency_ms=%d, iterations=%d", secureChannel, maxLatencyMs, iterations)

	// init a logger to use it for NewPeer
	logger, err := log.NewDefaultProductionLogger()
	if err != nil {
		panic(err)
	}

	wg := &sync.WaitGroup{}
	host, err := p2p.NewPeer(ctx, wg, logger, config)
	if err != nil {
		return fmt.Errorf("failed to instantiate libp2p instance: %w", err)
	}
	defer host.Close()

	addrs := func() []ma.Multiaddr {
		hostAddrs, err := host.P2PAddrs()
		if err != nil {
			panic(err)
		}

		ai, err := p2p.PeerInfoFromMultiAddr(hostAddrs[0].String())
		if err != nil {
			panic(err)
		}
		runenv.RecordMessage("my listen addrs: %s", ai.Addrs[0].String())

		return ai.Addrs
	}

	var (
		hostId     = host.ID()
		ai         = &p2p.PeerAddrInfo{ID: hostId, Addrs: addrs()}
		peersTopic = tgSync.NewTopic("peers", new(p2p.PeerAddrInfo))
		peers      = make([]*p2p.PeerAddrInfo, 0, runenv.TestInstanceCount)
	)

	initCtx.SyncClient.MustPublish(ctx, peersTopic, ai)
	peersCh := make(chan *p2p.PeerAddrInfo)
	sctx, scancel := context.WithCancel(ctx)
	sub := initCtx.SyncClient.MustSubscribe(sctx, peersTopic, peersCh)

	for len(peers) < cap(peers) {
		select {
		case ai := <-peersCh:
			peers = append(peers, ai)
		case err := <-sub.Done():
			{
				scancel()
				return err
			}
		}
	}
	scancel()

	pingPeers := func(tag string) error {
		g, gctx := errgroup.WithContext(ctx)
		for _, ai := range peers {
			if ai.ID == hostId {
				continue
			}
			id := ai.ID
			g.Go(func() error {
				pctx, cancel := context.WithCancel(gctx)
				defer cancel()
				rtt, err := host.Ping(pctx, id)
				if err != nil {
					return err
				}
				runenv.RecordMessage("ping result (%s) from peer %s: %s", tag, id, rtt)
				point := fmt.Sprintf("ping-result,round=%s,peer=%s", tag, id)
				runenv.R().RecordPoint(point, float64(rtt.Nanoseconds()))
				return nil
			})
		}
		return g.Wait()
	}

	for _, ai := range peers {
		if ai.ID == hostId {
			break
		}
		runenv.RecordMessage("Dial peer: %s", ai.ID)
		if err := host.Connect(ctx, *ai); err != nil {
			return err
		}
	}

	runenv.RecordMessage("done dialling my peers")
	initCtx.SyncClient.MustSignalAndWait(ctx, "connected", runenv.TestInstanceCount)
	if err := pingPeers("initial"); err != nil {
		return err
	}
	initCtx.SyncClient.MustSignalAndWait(ctx, "initial", runenv.TestInstanceCount)
	rand.Seed(time.Now().UnixNano() + initCtx.GlobalSeq)

	for i := 1; i <= iterations; i++ {
		runenv.RecordMessage("⚡️  ITERATION ROUND %d", i)

		latency := time.Duration(rand.Int31n(int32(maxLatencyMs))) * time.Millisecond
		runenv.RecordMessage("(round %d) my latency: %s", i, latency)
		initCtx.NetClient.MustConfigureNetwork(ctx, &network.Config{
			Network:        "default",
			Enable:         true,
			Default:        network.LinkShape{Latency: latency},
			CallbackState:  tgSync.State(fmt.Sprintf("network-configured-%d", i)),
			CallbackTarget: runenv.TestInstanceCount,
		})

		if err := pingPeers(fmt.Sprintf("iteration-%d", i)); err != nil {
			return err
		}

		doneState := tgSync.State(fmt.Sprintf("done-%d", i))
		initCtx.SyncClient.MustSignalAndWait(ctx, doneState, runenv.TestInstanceCount)
	}

	_ = host.Close()
	return nil
}

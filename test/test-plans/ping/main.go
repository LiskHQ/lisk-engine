package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"

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

	cfg := p2p.Config{
		AllowIncomingConnections: true,
		ConnectionSecurity:       secureChannel,
		Addresses:                []string{fmt.Sprintf("/ip4/%s/tcp/0", ip)},
		Version:                  "2.0",
		ChainID:                  []byte{0x04, 0x00, 0x01, 0x02},
	}

	runenv.RecordMessage("started test instance; params: secure_channel=%s, max_latency_ms=%d, iterations=%d", secureChannel, maxLatencyMs, iterations)
	host := p2p.NewConnection(&cfg)
	// init a logger to use it for Host
	logger, err := log.NewDefaultProductionLogger()
	if err != nil {
		panic(err)
	}
	if err := host.Start(logger, nil); err != nil {
		panic(err)
	}

	addrs := func() []ma.Multiaddr {
		hostAddrs, err := host.MultiAddress()
		if err != nil {
			panic(err)
		}

		ai, err := p2p.AddrInfoFromMultiAddr(hostAddrs[0])
		if err != nil {
			panic(err)
		}
		runenv.RecordMessage("my listen addrs: %s", ai.Addrs[0].String())

		return ai.Addrs
	}

	var (
		hostId     = host.ID()
		ai         = &p2p.AddrInfo{ID: hostId, Addrs: addrs()}
		peersTopic = tgSync.NewTopic("peers", new(p2p.AddrInfo))
		peers      = make([]*p2p.AddrInfo, 0, runenv.TestInstanceCount)
	)

	initCtx.SyncClient.MustPublish(ctx, peersTopic, ai)
	peersCh := make(chan *p2p.AddrInfo)
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

	return nil
}

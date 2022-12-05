package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	p2p "github.com/LiskHQ/lisk-engine/pkg/p2p/v2"

	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
	"golang.org/x/sync/errgroup"
)

var testcases = map[string]interface{}{
	"ping": run.InitializedTestCaseFn(runPing),
}

func main() {
	run.InvokeMap(testcases)
}

func getSecurityByName(secureChannel string) p2p.PeerSecurityOption {
	switch secureChannel {
	case "noise":
		return p2p.PeerSecurityNoise
	case "tls":
		return p2p.PeerSecurityTLS
	case "none":
		return p2p.PeerSecurityNone
	}

	panic(fmt.Sprintf("unknown secure channel: %s", secureChannel))
}

func runPing(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	var (
		secureChannel = runenv.StringParam("secure_channel")
		maxLatencyMs  = runenv.IntParam("max_latency_ms")
		iterations    = runenv.IntParam("iterations")
	)

	runenv.RecordMessage("started test instance; params: secure_channel=%s, max_latency_ms=%d, iterations=%d", secureChannel, maxLatencyMs, iterations)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	initCtx.MustWaitAllInstancesInitialized(ctx)
	ip := initCtx.NetClient.MustGetDataNetworkIP()

	// init a logger to use it for NewPeer
	logger, err := log.NewDefaultProductionLogger()
	if err != nil {
		panic(err)
	}

	listenAddr := fmt.Sprintf("/ip4/%s/tcp/0", ip)
	host, err := p2p.NewPeer(ctx, logger, []string{listenAddr}, getSecurityByName(secureChannel))

	if err != nil {
		return fmt.Errorf("failed to instantiate libp2p instance: %w", err)
	}
	defer host.Close()

	runenv.RecordMessage("my listen addrs: %v", host.Addrs())

	var (
		hostId     = host.ID()
		ai         = &p2p.PeerAddrInfo{ID: hostId, Addrs: host.Addrs()}
		peersTopic = sync.NewTopic("peers", new(p2p.PeerAddrInfo))
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
				runenv.R().RecordPoint(point, float64(rtt.Milliseconds()))
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
			CallbackState:  sync.State(fmt.Sprintf("network-configured-%d", i)),
			CallbackTarget: runenv.TestInstanceCount,
		})

		if err := pingPeers(fmt.Sprintf("iteration-%d", i)); err != nil {
			return err
		}

		doneState := sync.State(fmt.Sprintf("done-%d", i))
		initCtx.SyncClient.MustSignalAndWait(ctx, doneState, runenv.TestInstanceCount)
	}

	_ = host.Close()
	return nil
}

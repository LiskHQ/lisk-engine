package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	rt "runtime"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/log"
	"github.com/LiskHQ/lisk-engine/pkg/p2p"

	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"golang.org/x/sync/errgroup"

	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

// Add an offset to attack nodes so it's easy to identify them
const attackSeqOffset = 1000000

// Listen on the address in the testground data network
func listenAddrs(netclient *network.Client) []multiaddr.Multiaddr {
	ip, err := netclient.GetDataNetworkIP()
	if err == network.ErrNoTrafficShaping {
		ip = net.ParseIP("0.0.0.0")
	} else if err != nil {
		panic(fmt.Errorf("error getting data network addr: %s", err))
	}

	dataAddr, err := manet.FromIP(ip)
	if err != nil {
		panic(fmt.Errorf("could not convert IP to multiaddr; ip=%s, err=%s", ip, err))
	}

	// add /tcp/0 to auto select TCP listen port
	listenAddr := dataAddr.Encapsulate(multiaddr.StringCast("/tcp/0"))
	return []multiaddr.Multiaddr{listenAddr}
}

type testInstance struct {
	*runtime.RunEnv
	params testParams

	conn           p2p.Connection
	seq            int64
	nodeTypeSeq    int64
	nodeIdx        int
	latency        time.Duration
	connsDef       *ConnectionsDef
	client         *sync.DefaultClient
	discovery      *SyncDiscovery
	peerSubscriber *PeerSubscriber
}

type Message struct {
	Name string
	Body string
	Time int64
}

func RunSimulation(ctx context.Context, initCtx *run.InitContext, runenv *runtime.RunEnv, params testParams) error {
	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()

	// Create the connections, but don't listen yet (we need to set up the data
	// network before listening)
	conns := make([]p2p.Connection, params.nodesPerContainer)
	for i := 0; i < params.nodesPerContainer; i++ {
		cfg := p2p.Config{
			Version: "2.0",
			ChainID: []byte{0x04, 0x00, 0x01, 0x02},
		}

		// Don't listen yet, we need to set up networking first
		conn := *p2p.NewConnection(&cfg)
		logger, err := log.NewDefaultProductionLogger()
		if err != nil {
			panic(err)
		}
		if err := conn.Start(logger, nil); err != nil {
			panic(err)
		}
		conns[i] = conn
	}

	// Get sequence number within a node type (eg honest-1, honest-2, etc)
	nodeTypeSeq, err := getNodeTypeSeqNum(ctx, client, conns[0], params.nodeType)
	if err != nil {
		return fmt.Errorf("failed to get node type sequence number: %w", err)
	}

	// Make sure each container has a distinct random seed
	rand.Seed(nodeTypeSeq * time.Now().UnixNano())

	runenv.RecordMessage("%s container %d num cpus: %d", params.nodeType, nodeTypeSeq, rt.NumCPU())

	// Get the sequence number of each node in the container
	peers := sync.NewTopic("nodes", &p2p.AddrInfo{})
	seqs := make([]int64, params.nodesPerContainer)
	for nodeIdx := 0; nodeIdx < params.nodesPerContainer; nodeIdx++ {
		seq, err := client.Publish(ctx, peers, conns[nodeIdx].Info())
		if err != nil {
			return fmt.Errorf("failed to write peer subtree in sync service: %w", err)
		}

		// Add an offset to attack nodes so it's easy to identify them
		if params.nodeType != NodeTypeHonest {
			seq += attackSeqOffset
		}
		seqs[nodeIdx] = seq
	}

	// If a topology definition was provided, read the latency from it
	if len(params.connsDef) > 0 {
		// Note: The latency is the same for all nodes in the same container
		nodeIdx := 0
		connsDef, err := loadConnections(params.connsDef, params.nodeType, nodeTypeSeq, nodeIdx)
		if err != nil {
			return err
		}

		params.netParams.latency = connsDef.Latency
		params.netParams.latencyMax = time.Duration(0)
	}

	netclient := network.NewClient(client, runenv)

	// Set up traffic shaping. Note: this is the same for all nodes in the same container.
	if err := setupNetwork(ctx, runenv, params.netParams, netclient); err != nil {
		return fmt.Errorf("Failed to set up network: %w", err)
	}

	// Set up a subscription for node information from all peers in all containers.
	// Note that there is only on PeerSubscriber per container (but there may be
	// several nodes per container).
	peerSubscriber := NewPeerSubscriber(ctx, runenv, client, runenv.TestInstanceCount, params.containerNodesTotal)

	// Create each node in the container
	errgrp, ctx := errgroup.WithContext(ctx)
	for nodeIdx := 0; nodeIdx < params.nodesPerContainer; nodeIdx++ {
		nodeIdx := nodeIdx

		errgrp.Go(func() (err error) {
			t := testInstance{
				RunEnv:         runenv,
				conn:           conns[nodeIdx],
				seq:            seqs[nodeIdx],
				nodeTypeSeq:    nodeTypeSeq,
				nodeIdx:        nodeIdx,
				params:         params,
				client:         client,
				peerSubscriber: peerSubscriber,
			}

			// Load the connection definition for the node
			var connsDef *ConnectionsDef
			if len(params.connsDef) > 0 {
				connsDef, err = loadConnections(params.connsDef, params.nodeType, nodeTypeSeq, nodeIdx)
				if err != nil {
					return
				}
				t.connsDef = connsDef
			}

			// Listen for incoming connections
			laddr := listenAddrs(netclient)
			runenv.RecordMessage("listening on %s", laddr)
			if err = t.conn.GetHost().Network().Listen(laddr...); err != nil {
				return nil
			}

			id := t.conn.ID().Pretty()
			runenv.RecordMessage("Connection peer ID: %s, seq %d, node type: %s, node type seq: %d, node index: %d / %d, addrs: %v",
				id, t.seq, params.nodeType, nodeTypeSeq, nodeIdx, params.nodesPerContainer, t.conn.GetHost().Addrs())

			switch params.nodeType {
			case NodeTypeSybil:
				err = t.startSybil(ctx)
			default:
				err = t.startHonest(ctx)
			}

			return
		})
	}

	return errgrp.Wait()
}

func getNodeTypeSeqNum(ctx context.Context, client *sync.DefaultClient, conn p2p.Connection, nodeType NodeType) (int64, error) {
	topic := sync.NewTopic("node-type-"+string(nodeType), &p2p.AddrInfo{})
	return client.Publish(ctx, topic, conn.Info())
}

func (t *testInstance) startHonest(ctx context.Context) error {
	tracerOut := fmt.Sprintf("%s%ctracer-output-honest-%d", t.TestOutputsPath, os.PathSeparator, t.seq)
	t.RecordMessage("writing honest node tracer output to %s", tracerOut)

	// if we're a publisher, our message publish rate should be a fraction of
	// the total message rate for each topic. For now, we distribute the
	// publish rates uniformly across the number of instances in our
	// testground composition
	topics := make([]TopicConfig, len(t.params.topics))
	if t.params.publisher {
		// FIXME: this assumes all publishers are in the same group, might not always hold up.
		nPublishers := t.TestGroupInstanceCount
		for i, topic := range t.params.topics {
			topics[i] = topic
			topics[i].MessageRate.Quantity /= float64(nPublishers)
		}
	} else {
		topics = t.params.topics
	}

	tracer, err := NewTestTracer(tracerOut, t.conn.ID(), t.params.fullTraces)
	if err != nil {
		return fmt.Errorf("error making test tracer: %s", err)
	}

	scoreInspectParams := InspectParams{}
	if t.params.scoreInspectPeriod != 0 {
		scoreInspectParams.Period = t.params.scoreInspectPeriod

		outpath := fmt.Sprintf("%s%cpeer-scores-honest-%d.json", t.TestOutputsPath, os.PathSeparator, t.seq)
		file, err := os.OpenFile(outpath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			return fmt.Errorf("error opening peer score output file at %s: %s", outpath, err)
		}
		defer file.Close()
		t.RecordMessage("recording peer scores to %s", outpath)
		enc := json.NewEncoder(file)
		type entry struct {
			Timestamp int64
			PeerID    string
			Scores    map[string]float64
		}
		scoreInspectParams.Inspect = func(scores map[p2p.PeerID]float64) {
			ts := time.Now().UnixNano()
			pretty := make(map[string]float64, len(scores))
			for p, s := range scores {
				pretty[p.Pretty()] = s
			}
			e := entry{
				Timestamp: ts,
				PeerID:    t.conn.ID().Pretty(),
				Scores:    pretty,
			}
			err := enc.Encode(e)
			if err != nil {
				t.RecordMessage("error encoding peer scores: %s", err)
			}
		}
	}

	cfg := HonestNodeConfig{
		Publisher:               t.params.publisher,
		FloodPublishing:         t.params.floodPublishing,
		PeerScoreParams:         t.params.scoreParams,
		OverlayParams:           t.params.overlayParams,
		PeerScoreInspect:        scoreInspectParams,
		Topics:                  topics,
		Tracer:                  tracer,
		Seq:                     t.seq,
		Warmup:                  t.params.warmup,
		Cooldown:                t.params.cooldown,
		Heartbeat:               t.params.heartbeat,
		ValidateQueueSize:       t.params.validateQueueSize,
		OutboundQueueSize:       t.params.outboundQueueSize,
		OpportunisticGraftTicks: t.params.opportunisticGraftTicks,
	}

	n, err := NewPubsubNode(t.RunEnv, ctx, t.conn, cfg)
	if err != nil {
		return err
	}

	discovery, err := t.setupDiscovery(ctx)
	if err != nil {
		return err
	}

	err = n.Run(t.params.runtime, func(ctx context.Context) error {
		// wait for all other nodes to be ready
		if err := t.waitForReadyState(ctx); err != nil {
			return err
		}

		// connect topology async
		go t.connectTopology(ctx)

		return nil
	})
	if err2 := tracer.Stop(); err2 != nil {
		t.RecordMessage("error stopping test tracer: %s", err2)
	}

	return t.outputConns(discovery)
}

func (t *testInstance) startSybil(ctx context.Context) error {
	t.RecordMessage("starting sybil attack node")

	bb, err := NewBadBoy(ctx, t.RunEnv, t.conn, t.nodeTypeSeq, t.params.sybilParams)
	if err != nil {
		return err
	}

	discovery, err := t.setupDiscovery(ctx)
	if err != nil {
		return err
	}

	// If the censorSingleNode parameter is true, censor the first publisher node
	if t.params.censorSingleNode {
		var ps []PeerRegistration
		for _, p := range discovery.allPeers {
			ps = append(ps, p)
		}
		publisher := selectSinglePublisher(ps)
		if publisher != nil {
			bb.Censor(publisher.Info.ID)
		}
	}

	// Wait for all nodes to be in the ready state (including honest nodes)
	// then start connecting (asynchonously)
	if err := t.waitForReadyStateThenConnect(ctx); err != nil {
		return err
	}

	bb.OnReady()

	return t.runAttackNode(ctx, discovery)
}

func (t *testInstance) runAttackNode(ctx context.Context, discovery *SyncDiscovery) error {
	expectedTime := t.params.runtime + t.params.warmup + t.params.cooldown
	ctx, cancel := context.WithTimeout(ctx, expectedTime)
	defer cancel()

	// just wait for the test to complete
	<-ctx.Done()

	// save topology to a file
	return t.outputConns(discovery)
}

func (t *testInstance) setupDiscovery(ctx context.Context) (*SyncDiscovery, error) {
	t.RecordMessage("Setup discovery")

	// By default connect to a randomly-chosen subset of all honest nodes
	var topology Topology
	topology = RandomHonestTopology{
		Count:          t.params.degree,
		PublishersOnly: t.params.connectToPublishersOnly,
	}

	// If a topology file was supplied, use the topology defined there
	if t.connsDef != nil {
		topology = FixedTopology{t.connsDef}
	} else if t.params.nodeType != NodeTypeHonest && t.params.attackSingleNode {
		// If it's an attack against a single node, choose a topology that
		// connects to one publisher node
		topology = SinglePublisherTopology{}
	}

	// Register this node and get node information for all peers
	discovery, err := NewSyncDiscovery(t.conn, t.RunEnv, t.peerSubscriber, topology,
		t.params.nodeType, t.nodeTypeSeq, t.nodeIdx, t.params.publisher)
	if err != nil {
		return nil, fmt.Errorf("error creating discovery service: %s", err)
	}
	t.discovery = discovery

	err = discovery.registerAndWait(ctx)
	if err != nil {
		return nil, fmt.Errorf("error waiting for discovery service: %s", err)
	}

	return discovery, nil
}

// Called when nodes are ready to start the run, and are waiting for all other nodes to be ready
func (t *testInstance) waitForReadyStateThenConnect(ctx context.Context) error {
	// wait for all other nodes to be ready
	if err := t.waitForReadyState(ctx); err != nil {
		return err
	}

	// connect topology
	return t.connectTopology(ctx)
}

// Called when nodes are ready to start the run, and are waiting for all other nodes to be ready
func (t *testInstance) waitForReadyState(ctx context.Context) error {
	// Set a state barrier.
	state := sync.State("ready")
	doneCh := t.client.MustBarrier(ctx, state, t.params.containerNodesTotal).C

	// Signal we've entered the state.
	t.RecordMessage("Signalling ready state")
	_, err := t.client.SignalEntry(ctx, state)
	if err != nil {
		return err
	}

	// Wait until all others have signalled.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-doneCh:
		if err != nil {
			return err
		}
		t.RecordMessage("All instances in ready state, continuing")
	}

	return nil
}

func (t *testInstance) connectTopology(ctx context.Context) error {
	// Default to a connect delay in the range of 0s - 1s
	delay := time.Duration(float64(time.Second) * rand.Float64())

	// If an explicit delay was specified, calculate the delay
	nodeTypeIdx := int(t.nodeTypeSeq - 1)
	if nodeTypeIdx < len(t.params.connectDelays) {
		expdelay := t.params.connectDelays[nodeTypeIdx]

		// Add +/- jitter percent
		delay = expdelay + time.Duration(t.params.connectDelayJitterPct)*expdelay/100
		delay += time.Duration(rand.Float64() * float64(time.Duration(t.params.connectDelayJitterPct*2)*expdelay/100))
	}

	// Connect to other peers in the topology
	err := t.discovery.ConnectTopology(ctx, delay)
	if err != nil {
		t.RecordMessage("Error connecting to topology peer: %s", err)
	}

	return nil
}

// Wait for all nodes to signal that they have completed the run
// (or there's a timeout)
func (t *testInstance) waitForCompleteState(ctx context.Context) error {
	// Set a state barrier.
	state := sync.State("complete")

	// Signal we've entered the state, and wait until all others have signalled.
	t.RecordMessage("Signalling complete state")
	_, err := t.client.SignalAndWait(ctx, state, t.params.containerNodesTotal)
	if err != nil {
		return err
	}
	t.RecordMessage("All instances in complete state, done")
	return nil
}

type ConnectionsDef struct {
	Latency     time.Duration
	Connections []string
}

func loadConnections(connsDef map[string]*ConnectionsDef, nodeType NodeType, nodeTypeSeq int64, nodeIdx int) (*ConnectionsDef, error) {
	nodeKey := fmt.Sprintf("%s-%d-%d", nodeType, nodeTypeSeq, nodeIdx)
	def, ok := connsDef[nodeKey]
	if !ok {
		return nil, fmt.Errorf("Topology file '%s' has no entry for '%s'", nodeType, nodeKey)
	}
	return def, nil
}

func (t *testInstance) outputConns(discovery *SyncDiscovery) error {
	connsOut := fmt.Sprintf("%s%cconnections-%s-%d-%d.json", t.TestOutputsPath, os.PathSeparator, t.params.nodeType, t.nodeTypeSeq, t.nodeIdx)

	var conns []string
	for _, p := range discovery.Connected() {
		conns = append(conns, fmt.Sprintf("%s-%d-%d", p.NType, p.NodeTypeSeq, p.NodeIdx))
	}

	jsonstr, err := json.MarshalIndent(ConnectionsDef{
		Latency:     t.params.netParams.latency,
		Connections: conns,
	}, "", "  ")

	if err != nil {
		return err
	}
	return ioutil.WriteFile(connsOut, jsonstr, os.ModePerm)
}

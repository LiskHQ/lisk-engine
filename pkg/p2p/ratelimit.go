package p2p

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	ma "github.com/multiformats/go-multiaddr"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

const rateLimiterHandleInterval = 10 * time.Second // Interval to handle messages rate limiting
const defaultRateLimit = 100                       // Default maximum allowed number of received messages
const defaultRateLimitPenalty = 10                 // Default rate limit penalty for received messages

// WithRPCMessageCounter sets a rate limit for a specific RPC message handler.
func WithRPCMessageCounter(limit, penalty int) RPCHandlerOption {
	return func(mp *MessageProtocol, name string) error {
		if mp.rateLimit == nil {
			return errors.New("cannot set RPC message counter because rate limiter is not available")
		}
		mp.rateLimit.rpcMessageCounters[name].limit = limit
		mp.rateLimit.rpcMessageCounters[name].penalty = penalty
		return nil
	}
}

// rateLimit type for rate limiting messages.
type rateLimit struct {
	logger             log.Logger
	peer               *Peer
	rpcMessageCounters map[string]*rpcMessageCounter
	interval           time.Duration
}

// rpcMessageCounter type for RPC message counters.
type rpcMessageCounter struct {
	mu       sync.Mutex
	limit    int
	penalty  int
	counters map[PeerID]int
}

// newRateLimit creates a new rate limit.
func newRateLimit() *rateLimit {
	return &rateLimit{
		rpcMessageCounters: make(map[string]*rpcMessageCounter),
		interval:           rateLimiterHandleInterval,
	}
}

// start starts the rate limit.
func (rl *rateLimit) start(logger log.Logger, peer *Peer) {
	rl.logger = logger
	rl.peer = peer
}

// addRPCMessageCounter adds a new RPC message counter.
func (rl *rateLimit) addRPCMessageCounter(rpcName string) error {
	if _, ok := rl.rpcMessageCounters[rpcName]; ok {
		return fmt.Errorf("rpcMessageCounter %s already exists", rpcName)
	}
	rl.rpcMessageCounters[rpcName] =
		&rpcMessageCounter{
			limit:    defaultRateLimit,
			penalty:  defaultRateLimitPenalty,
			counters: make(map[PeerID]int),
		}
	return nil
}

// increaseCounter increases the counter for a specific RPC message.
func (rl *rateLimit) increaseCounter(rpcName string, peerID PeerID) {
	rl.rpcMessageCounters[rpcName].mu.Lock()
	defer rl.rpcMessageCounters[rpcName].mu.Unlock()
	rl.rpcMessageCounters[rpcName].counters[peerID]++
}

// checkLimit checks if the rate limit for a specific RPC message has been reached and applies a penalty if needed.
func (rl *rateLimit) checkLimit(rpcName string, peerID PeerID, peerAddr ma.Multiaddr) error {
	if rl.peer == nil {
		return errors.New("cannot check rate limits because rate limiter is not started")
	}

	msgCounter := rl.rpcMessageCounters[rpcName]
	msgCounter.mu.Lock()
	defer msgCounter.mu.Unlock()

	if msgCounter.counters[peerID] > msgCounter.limit {
		rl.logger.Debugf("Peer %s sent too many messages of type %s, applying penalty", peerID, rpcName)
		addr := peerAddr.String() + "/p2p/" + peerID.String()
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			rl.logger.Warningf("Failed to create a new multiaddr: %s", err)
			return err
		}
		if err := rl.peer.addPenalty(maddr, msgCounter.penalty); err != nil {
			rl.logger.Errorf("Failed to apply penalty to peer %s: %v", peerID, err)
			return err
		}
		msgCounter.counters[peerID] = 0
	}

	return nil
}

// rateLimiterHandler handles the messages rate limiting and banning peers that send too many messages.
func rateLimiterHandler(ctx context.Context, wg *sync.WaitGroup, rl *rateLimit) {
	defer wg.Done()
	rl.logger.Infof("Rate limiter handler started")

	t := time.NewTicker(rl.interval)

	for {
		select {
		case <-t.C:
			// Iterate over the rate limits map and reset the rate limiters counters.
			for _, rpcMessageCounter := range rl.rpcMessageCounters {
				rpcMessageCounter.mu.Lock()
				rpcMessageCounter.counters = make(map[PeerID]int)
				rpcMessageCounter.mu.Unlock()
			}
			t.Reset(rl.interval)
		case <-ctx.Done():
			rl.logger.Infof("Rate limiter handler stopped")
			return
		}
	}
}

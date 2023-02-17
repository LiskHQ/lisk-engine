package p2p

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"

	"github.com/LiskHQ/lisk-engine/pkg/log"
)

var (
	errInvalidDuration       = errors.New("the value of duration is invalid")
	errConnGaterIsNotrunning = errors.New("to be able add new peer, please call start function")
)

// connectionGater extendis the BasicConnectionGater of the libp2p to use expire
// time to remove peer ID from blockPeers.
type connectionGater struct {
	logger         log.Logger
	connGater      *conngater.BasicConnectionGater
	blockedPeers   map[peer.ID]int64
	expireDuration time.Duration
	intervalCheck  time.Duration
	mutex          sync.RWMutex
	isStarted      bool
}

// newConnGater returns a new connectionGater.
func newConnGater(exDuration, iCheck time.Duration) (*connectionGater, error) {
	if exDuration <= 0 || iCheck <= 0 {
		return nil, errInvalidDuration
	}

	cg, err := conngater.NewBasicConnectionGater(nil)
	if err != nil {
		return nil, err
	}

	return &connectionGater{
		connGater:      cg,
		blockedPeers:   make(map[peer.ID]int64),
		expireDuration: exDuration,
		intervalCheck:  iCheck,
		mutex:          sync.RWMutex{},
	}, nil
}

// blockPeer blocks the given peer ID.
func (cg *connectionGater) blockPeer(pid peer.ID) error {
	if !cg.isStarted {
		return errConnGaterIsNotrunning
	}

	err := cg.connGater.BlockPeer(pid)
	if err != nil {
		return err
	}

	cg.mutex.Lock()
	defer cg.mutex.Unlock()
	cg.blockedPeers[pid] = time.Now().Unix() + int64(cg.expireDuration.Seconds())

	return nil
}

// start runs a new goroutine to check the expiration time based on
// intervaliCheck, it will be run automatically.
func (cg *connectionGater) start(ctx context.Context) {
	if !cg.isStarted {
		t := time.NewTicker(cg.intervalCheck)
		go func() {
			for {
				select {
				case <-t.C:
					for p, t := range cg.blockedPeers {
						if time.Now().Unix() > t {
							err := cg.connGater.UnblockPeer(p)
							if err != nil {
								cg.logger.Error("UnblockPeer with error:", err)
								// Just make a log
							} else {
								cg.mutex.Lock()
								delete(cg.blockedPeers, p)
								cg.mutex.Unlock()
							}
						}
					}
				case <-ctx.Done():
					return
				}
			}
		}()
		cg.isStarted = true
	}
}

// optionWithBlacklist returns the ConnectionGater option of the libp2p which
// is usable to reject incoming and outgoing connections based on blacklists.
// Locally verifying the blacklist in unit test is not feasible because it needs multiple IPs.
// It is tested in the network.
func (cg *connectionGater) optionWithBlacklist(bl []string) (libp2p.Option, error) {
	if len(bl) > 0 {
		blIPs := []net.IP{}
		for _, ipStr := range bl {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				return nil, fmt.Errorf("IP %s is invalid", ipStr)
			}
			blIPs = append(blIPs, ip)
		}

		for _, adr := range blIPs {
			err := cg.connGater.BlockAddr(adr)
			if err != nil {
				return nil, err
			}
		}
	}

	return libp2p.ConnectionGater(cg.connGater), nil
}

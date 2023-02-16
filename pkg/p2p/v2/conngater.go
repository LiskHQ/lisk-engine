package p2p

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"
)

type Gater struct {
	gc          *conngater.BasicConnectionGater
	blockedPeer map[peer.ID]int64
	expireTime  time.Duration
	interval    time.Duration
	mutex       sync.RWMutex
}

func newGater(expireTime, interval time.Duration) (*Gater, error) {
	gc, err := conngater.NewBasicConnectionGater(nil)
	if err != nil {
		return nil, err
	}

	return &Gater{
		gc:          gc,
		blockedPeer: make(map[peer.ID]int64),
		expireTime:  expireTime,
		interval:    interval,
		mutex:       sync.RWMutex{},
	}, nil
}

func (g *Gater) addPeer(pid peer.ID) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	err := g.gc.BlockPeer(pid)
	if err != nil {
		return err
	}
	g.blockedPeer[pid] = time.Now().Unix() + int64(g.expireTime.Seconds())

	return nil
}

func (g *Gater) start() {
	go func() {
		for range time.Tick(g.interval) {
			for p, t := range g.blockedPeer {
				if time.Now().Unix() > t {
					err := g.gc.UnblockPeer(p)
					if err != nil {
						panic(err)
					}
					g.mutex.Lock()
					delete(g.blockedPeer, p)
					g.mutex.Unlock()
				}
			}
		}
	}()
}

func (g *Gater) connectionGater() *conngater.BasicConnectionGater {
	return g.gc
}

// ConnectionGaterOption returns the ConnectionGater option of the libp2p which
// is usable to reject incoming and outgoing connections based on blacklists.
// Locally verifying the blacklist in unit test is not feasible because it needs multiple IPs.
// It is tested in the network.
func ConnectionGaterOption(cg *conngater.BasicConnectionGater, bl []string) (libp2p.Option, error) {
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
			err := cg.BlockAddr(adr)
			if err != nil {
				return nil, err
			}
		}
	}

	return libp2p.ConnectionGater(cg), nil
}

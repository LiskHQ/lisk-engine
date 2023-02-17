package p2p

import (
	"math/rand"
	"time"
)

type bufferedTicker struct {
	C     chan time.Time
	stopc chan bool
	min   int64
	max   int64
}

func newBufferedTicker(duration time.Duration) *bufferedTicker {
	ticker := &bufferedTicker{
		C:     make(chan time.Time, 1),
		stopc: make(chan bool),
		max:   duration.Nanoseconds(),
		min:   duration.Nanoseconds(),
	}
	go ticker.loop()
	return ticker
}

func newRandomBufferedTicker(min, max time.Duration) *bufferedTicker {
	ticker := &bufferedTicker{
		C:     make(chan time.Time, 1),
		stopc: make(chan bool),
		max:   max.Nanoseconds(),
		min:   min.Nanoseconds(),
	}
	go ticker.loop()
	return ticker
}

func (t *bufferedTicker) next() time.Duration {
	if t.min == t.max {
		return time.Duration(t.min)
	}
	interval := rand.Int63n(t.max-t.min) + t.min
	return time.Duration(interval)
}

func (t *bufferedTicker) stop() {
	close(t.stopc)
	close(t.C)
}

func (t *bufferedTicker) loop() {
	timer := time.NewTimer(t.next())
	for {
		select {
		case <-t.stopc:
			timer.Stop()
			return
		case now := <-timer.C:
			timer.Stop()
			timer = time.NewTimer(t.next())
			select {
			case t.C <- now:
			default:
			}
		}
	}
}

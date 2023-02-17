package socket

import (
	"sync/atomic"
)

type counter struct {
	count uint64
}

func (c *counter) IncrementAndGet() int {
	return int(atomic.AddUint64(&c.count, 1))
}

func (c *counter) GetAndIncrement() int {
	defer atomic.AddUint64(&c.count, 1)
	return int(atomic.LoadUint64(&c.count))
}

func (c *counter) Reset() {
	atomic.StoreUint64(&c.count, 0)
}

func (c *counter) Get() int {
	return int(atomic.LoadUint64(&c.count))
}

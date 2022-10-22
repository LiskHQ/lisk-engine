package db

import (
	"sync"

	"github.com/cockroachdb/pebble"
)

type Batch struct {
	inner *pebble.Batch
	mutex *sync.Mutex
}

func (b *Batch) Set(key, value []byte) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.inner.Set(key, value, nil)
}

func (b *Batch) Del(key []byte) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.inner.Delete(key, nil)
}

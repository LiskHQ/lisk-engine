package db

import (
	"sync"

	"github.com/cockroachdb/pebble"
)

type Batch struct {
	inner *pebble.Batch
	mutex *sync.Mutex
}

func (b *Batch) Set(key, value []byte) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if err := b.inner.Set(key, value, nil); err != nil {
		panic(err)
	}
}

func (b *Batch) Del(key []byte) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if err := b.inner.Delete(key, nil); err != nil {
		panic(err)
	}
}

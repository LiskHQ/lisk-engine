// Package db implements key-value database functionality with prefix feature.
package db

import (
	"errors"
	"sync"

	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/cockroachdb/pebble"
)

var (
	ErrDataNotFound = errors.New("data was not found")
)

func upperBound(b []byte) []byte {
	end := make([]byte, len(b))
	copy(end, b)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end[:i+1]
		}
	}
	return nil // no upper-bound
}

type KeyValue interface {
	Key() []byte
	Value() []byte
}

func NewKeyValue(key, value []byte) KeyValue {
	return &keyValue{
		key:   key,
		value: value,
	}
}

type keyValue struct {
	key   []byte
	value []byte
}

func (k *keyValue) Key() []byte   { return k.key }
func (k *keyValue) Value() []byte { return k.value }

type DB struct {
	pebbleDB *pebble.DB
}

func NewDB(path string) (*DB, error) {
	pebbleDB, err := pebble.Open(path, &pebble.Options{
		ErrorIfExists: false,
	})
	if err != nil {
		return nil, err
	}
	db := &DB{
		pebbleDB: pebbleDB,
	}
	return db, nil
}

func (db *DB) Close() error {
	return db.pebbleDB.Close()
}

func (db *DB) Get(key []byte) ([]byte, bool) {
	data, closer, err := db.pebbleDB.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, false
		}
		// unknown error. if this fails, there is a problem in underlying DB which cannot be recovered.
		// Also, the pebble.Get only returns ErrNotFound, so this should never happen.
		panic(err)
	}
	copied := bytes.Copy(data)
	if err := closer.Close(); err != nil {
		// if this fails, application should crash otherwise, memory will likely to leak.
		// Another option is to ignore the erorr close, and have manually run GC.
		panic(err)
	}
	return copied, true
}

func (db *DB) Exist(key []byte) bool {
	_, exist := db.Get(key)
	return exist
}

func (db *DB) Set(key, value []byte) {
	if err := db.pebbleDB.Set(key, value, pebble.Sync); err != nil {
		// if it fails here, there is a problem in underlying DB which cannot be recovered.
		panic(err)
	}
}

func (db *DB) Del(key []byte) {
	if err := db.pebbleDB.Delete(key, pebble.Sync); err != nil {
		// if it fails here, there is a problem in underlying DB which cannot be recovered.
		panic(err)
	}
}

func (db *DB) IterateKey(prefix []byte, limit int, reverse bool) [][]byte {
	iter := db.pebbleDB.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound(prefix),
	})
	return iterateKeyPrefix(iter, prefix, limit, reverse)
}

func (db *DB) Iterate(prefix []byte, limit int, reverse bool) []KeyValue {
	iter := db.pebbleDB.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound(prefix),
	})
	return iteratePrefix(iter, prefix, limit, reverse)
}

func (db *DB) IterateRange(start, end []byte, limit int, reverse bool) []KeyValue {
	iter := db.pebbleDB.NewIter(nil)
	return iterateRange(iter, start, end, limit, reverse)
}

func (db *DB) NewBatch() *Batch {
	return &Batch{
		inner: db.pebbleDB.NewBatch(),
		mutex: new(sync.Mutex),
	}
}

func (db *DB) NewReader() *Reader {
	snapshot := db.pebbleDB.NewSnapshot()
	return &Reader{
		snapshot: snapshot,
	}
}

func (db *DB) Write(batch *Batch) {
	if err := db.pebbleDB.Apply(batch.inner, pebble.Sync); err != nil {
		// Apply returns error on Readonly mode, WAL disabled or errNoSplit which are configuration issues.
		// At this point, it will be better to panic.
		panic(err)
	}
}

func (db *DB) DropAll() {
	if err := db.pebbleDB.DeleteRange([]byte{0}, []byte{255}, pebble.NoSync); err != nil {
		// DeleteRange internally calls Apply, and with the same reason as above Write it will be better to panic.
		panic(err)
	}
}

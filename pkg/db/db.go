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

func (db *DB) Get(key []byte) ([]byte, error) {
	data, closer, err := db.pebbleDB.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrDataNotFound
		}
		return nil, err
	}
	copied := bytes.Copy(data)
	if err := closer.Close(); err != nil {
		return nil, err
	}
	return copied, nil
}

func (db *DB) Exist(key []byte) (bool, error) {
	_, err := db.Get(key)
	if err != nil && !errors.Is(err, ErrDataNotFound) {
		return false, err
	}
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (db *DB) Set(key, value []byte) error {
	return db.pebbleDB.Set(key, value, pebble.Sync)
}

func (db *DB) Del(key []byte) error {
	return db.pebbleDB.Delete(key, pebble.Sync)
}

func (db *DB) IterateKey(prefix []byte, limit int, reverse bool) ([][]byte, error) {
	iter := db.pebbleDB.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound(prefix),
	})
	return iterateKeyPrefix(iter, prefix, limit, reverse)
}

func (db *DB) Iterate(prefix []byte, limit int, reverse bool) ([]KeyValue, error) {
	iter := db.pebbleDB.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound(prefix),
	})
	return iteratePrefix(iter, prefix, limit, reverse)
}

func (db *DB) IterateRange(start, end []byte, limit int, reverse bool) ([]KeyValue, error) {
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

func (db *DB) Write(batch *Batch) error {
	return db.pebbleDB.Apply(batch.inner, pebble.Sync)
}

func (db *DB) DropAll() error {
	return db.pebbleDB.DeleteRange([]byte{0}, []byte{255}, pebble.NoSync)
}

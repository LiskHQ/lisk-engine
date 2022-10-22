// Package diffdb provides diff functionality, which is used for rolling back the stored data.
package diffdb

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/db"
)

const (
	prefixSize = 6
	KeySize    = 32 + prefixSize
)

var (
	// ErrNotFound is returned when data do not exist in the underline database.
	ErrNotFound = db.ErrDataNotFound
)

// DatabaseWriter interface only has put and set.
type DatabaseWriter interface {
	Set(key, value []byte) error
	Del(key []byte) error
}

// DatabaseReader interface.
type DatabaseReader interface {
	Get(key []byte) ([]byte, error)
	Iterate(prefix []byte, limit int, reverse bool) ([]db.KeyValue, error)
	IterateRange(start, end []byte, limit int, reverse bool) ([]db.KeyValue, error)
}

// DatabaseIterator interface.
type DatabaseIterator interface {
	IterateKey(prefix []byte, limit int, reverse bool) ([][]byte, error)
}

// DatabaseReadWriter interface

type keyValue struct {
	key   []byte
	value []byte
}

func (k *keyValue) Key() []byte   { return k.key }
func (k *keyValue) Value() []byte { return k.value }

// Database to store all state data.
type Database struct {
	store         DatabaseReader
	mutex         *sync.Mutex
	prefix        []byte
	prefixLength  int
	cache         *cacheDB
	snapshots     map[int]*cacheDB
	snapshotCount int
}

func New(store DatabaseReader, prefix []byte) *Database {
	return &Database{
		mutex:        new(sync.Mutex),
		store:        store,
		cache:        newCacheDB(),
		prefix:       prefix,
		prefixLength: len(prefix),
		snapshots:    make(map[int]*cacheDB),
	}
}

func (s *Database) WithPrefix(prefix []byte) *Database {
	nextPrefix := bytes.Join(s.prefix, prefix)
	return &Database{
		mutex:        s.mutex,
		store:        s.store,
		cache:        s.cache,
		prefix:       nextPrefix,
		prefixLength: len(nextPrefix),
		snapshots:    make(map[int]*cacheDB),
	}
}

func (s *Database) Has(key []byte) (bool, error) {
	_, err := s.Get(key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Get returns value with specified key from cache or from storage.
func (s *Database) Get(key []byte) ([]byte, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	val, exist, deleted := s.cache.get(s.getKey(key))
	if exist {
		return bytes.Copy(val), nil
	}
	if deleted {
		return nil, ErrNotFound
	}
	val, err := s.store.Get(s.getKey(key))
	if err != nil {
		if errors.Is(err, db.ErrDataNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	// set to cache
	s.cache.cache(s.getKey(key), val)
	return bytes.Copy(val), nil
}

func (s *Database) Range(start, end []byte, limit int, reverse bool) ([]db.KeyValue, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	prefixedStart := s.getKey(start)
	prefixedEnd := s.getKey(end)
	kv, err := s.store.IterateRange(prefixedStart, prefixedEnd, limit, reverse)
	if err != nil {
		return nil, err
	}
	storedValue := []db.KeyValue{}
	for _, data := range kv {
		val, exist, deleted := s.cache.get(data.Key())
		if deleted {
			continue
		}
		if !exist {
			s.cache.cache(data.Key(), data.Value())
		}
		storedValue = append(storedValue, &keyValue{
			key:   data.Key()[s.prefixLength:],
			value: bytes.Copy(val),
		})
	}
	cachedValue := s.cache.dataBetween(prefixedStart, prefixedEnd, s.prefixLength)

	return s.mergeSortLimit(cachedValue, storedValue, reverse, limit), nil
}

func (s *Database) Iterate(prefix []byte, limit int, reverse bool) ([]db.KeyValue, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	prefixedKey := s.getKey(prefix)
	kv, err := s.store.Iterate(prefixedKey, limit, reverse)
	if err != nil {
		return nil, err
	}
	storedValue := []db.KeyValue{}
	for _, data := range kv {
		val, exist, deleted := s.cache.get(data.Key())
		if deleted {
			continue
		}
		if !exist {
			s.cache.cache(data.Key(), data.Value())
		}
		storedValue = append(storedValue, &keyValue{
			key:   data.Key()[s.prefixLength:],
			value: bytes.Copy(val),
		})
	}
	cachedValue := s.cache.withPrefix(prefix, s.prefixLength)

	return s.mergeSortLimit(cachedValue, storedValue, reverse, limit), nil
}

// Set the value with specified key to cache.
func (s *Database) Set(key, value []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	prefixedKey := s.getKey(key)
	// 1. it does exist in cache just needs update => update
	// 2. it did exist in cache, but it was deleted => update
	if s.cache.existAny(prefixedKey) {
		s.cache.set(prefixedKey, value)
		return nil
	}
	// 3. it does not exist in cache, but it does exist in DB => cache first and update
	dataExist := s.ensureCache(prefixedKey)
	if dataExist {
		s.cache.set(prefixedKey, value)
		return nil
	}
	// 4. it does not exist in cache, and it does not exist in DB => add as new
	s.cache.add(prefixedKey, value)
	return nil
}

// Del the key.
func (s *Database) Del(key []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	prefixedKey := s.getKey(key)
	// if it does not exist it cache, ensure db state is reflected in DB
	if !s.cache.existAny(prefixedKey) {
		s.ensureCache(prefixedKey)
	}
	// remove from cache. if the cache exist, register for deletion. if not cached, just delete from memory
	s.cache.del(prefixedKey)
	// if it exist in cache and DB, remove cache and register for deletion
	return nil
}

func (s *Database) Commit(batch DatabaseWriter) (*Diff, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	// Create new child state db for state root and update here
	diff, err := s.cache.commit(batch)
	return diff, err
}

func (s *Database) RevertDiff(batch DatabaseWriter, diff *Diff) error {
	// Revert diff
	for _, added := range diff.Added {
		if err := batch.Del(added); err != nil {
			return err
		}
	}
	for _, deleted := range diff.Deleted {
		if err := batch.Set(deleted.Key, deleted.Value); err != nil {
			return err
		}
	}
	for _, updated := range diff.Updated {
		if err := batch.Set(updated.Key, updated.Value); err != nil {
			return err
		}
	}
	return nil
}

// Snapshot data and returns snapshot id.
func (s *Database) Snapshot() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	copied := s.cache.copy()
	id := s.snapshotCount
	s.snapshots[id] = copied
	s.snapshotCount++
	return id
}

// DeleteSnapshot by id.
func (s *Database) DeleteSnapshot(id int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.snapshots, id)
}

// RestoreSnapshot to data.
func (s *Database) RestoreSnapshot(id int) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	snapshot, exist := s.snapshots[id]
	if !exist {
		return fmt.Errorf("snapshot %d does not exist", id)
	}
	s.cache = snapshot
	delete(s.snapshots, id)
	return nil
}

// ensure if key is in DB, it is in cache also. return true if exist in cache or DB.
func (s *Database) ensureCache(key []byte) bool {
	val, err := s.store.Get(key)
	if err != nil {
		return false
	}
	s.cache.cache(key, val)
	return true
}

func (s *Database) getKey(key []byte) []byte {
	return bytes.JoinSize(s.prefixLength+len(key), s.prefix, key)
}

func (s *Database) mergeSortLimit(cached, stored []db.KeyValue, reverse bool, limit int) []db.KeyValue {
	existingMap := map[string]bool{}
	result := []db.KeyValue{}
	for _, data := range cached {
		existingMap[string(data.Key())] = true
		result = append(result, data)
	}
	for _, data := range stored {
		_, exist := existingMap[string(data.Key())]
		if !exist {
			result = append(result, data)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if reverse {
			return bytes.Compare(result[i].Key(), result[j].Key()) > 0
		}
		return bytes.Compare(result[i].Key(), result[j].Key()) < 0
	})
	if limit > -1 && len(result) > limit {
		result = result[:limit]
	}
	return result
}

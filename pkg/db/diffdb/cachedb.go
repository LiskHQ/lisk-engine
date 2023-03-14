package diffdb

import (
	"bytes"

	"github.com/LiskHQ/lisk-engine/pkg/db"
)

type cacheValue struct {
	// init value before the commit
	init []byte
	// current updated value
	value []byte
	// it does exist in the persisted data, and updated
	dirty bool
	// it does exist in the persisted data, and deleted
	deleted bool
}

func (c *cacheValue) copy() *cacheValue {
	val := make([]byte, len(c.value))
	copy(val, c.value)
	res := &cacheValue{
		value:   val,
		dirty:   c.dirty,
		deleted: c.deleted,
	}
	if c.init != nil {
		dest := make([]byte, len(c.init))
		copy(dest, c.init)
		res.init = dest
	}
	return res
}

// cacheDB is not threadsafe. It should be protected from outside.
type cacheDB struct {
	data map[string]*cacheValue
}

func newCacheDB() *cacheDB {
	return &cacheDB{
		data: make(map[string]*cacheValue),
	}
}

// patterns to cover
// when it exist in db, and not updated
// when it exist in db, and updated
// when it exist in db, and deleted
// when it doesn't exist in db, and added
// when it doesn't exist in db, and deleted
// when it doesn't exist in db, and updated

// add value to the cache which does not exist in persisted data.
func (c *cacheDB) add(key, value []byte) {
	cacheValue := &cacheValue{
		// no original
		value:   value,
		dirty:   false,
		deleted: false,
	}
	c.data[string(key)] = cacheValue
}

// cache value which does exist in persisted data
// it should be checked that cacheDB does not contain the key.
func (c *cacheDB) cache(key, value []byte) {
	strKey := string(key)
	init := make([]byte, len(value))
	copy(init, value)
	cacheValue := &cacheValue{
		init:  init,
		value: value,
	}
	c.data[strKey] = cacheValue
}

// set the value for the data which already exist in cache.
func (c *cacheDB) set(key, value []byte) {
	original, exist := c.data[string(key)]
	if !exist {
		panic("it should exist")
	}
	original.deleted = false
	original.dirty = true
	original.value = value
}

// get returns value, exist, deleted.
func (c *cacheDB) get(key []byte) ([]byte, bool, bool) {
	original, exist := c.data[string(key)]
	// if exist but deleted, return nil false
	if !exist {
		return nil, false, false
	}
	if original.deleted {
		return nil, false, true
	}
	return original.value, true, false
}

func (c *cacheDB) del(key []byte) {
	strKey := string(key)
	original, exist := c.data[strKey]
	if !exist {
		return
	}
	// if initial value is nil (ie: not in the DB), completely delete
	if original.init == nil {
		delete(c.data, strKey)
		return
	}
	original.deleted = true
	// else, set to deleted
}

// existAny returns exist true regardless of internal state.
func (c *cacheDB) existAny(key []byte) bool {
	_, exist := c.data[string(key)]
	return exist
}

func (c *cacheDB) withPrefix(prefix []byte, prefixLength int) []db.KeyValue {
	result := []db.KeyValue{}
	for key, value := range c.data {
		if value.deleted {
			continue
		}
		if bytes.HasPrefix([]byte(key), prefix) {
			result = append(result, &keyValue{
				key:   []byte(key)[prefixLength:],
				value: value.value,
			})
		}
	}
	return result
}

func (c *cacheDB) dataBetween(start, end []byte, prefixLength int) []db.KeyValue {
	result := []db.KeyValue{}
	for key, value := range c.data {
		if value.deleted {
			continue
		}
		if bytes.Compare([]byte(key), start) >= 0 && bytes.Compare([]byte(key), end) <= 0 {
			result = append(result, &keyValue{
				key:   []byte(key)[prefixLength:],
				value: value.value,
			})
		}
	}
	return result
}

func (c *cacheDB) copy() *cacheDB {
	copied := map[string]*cacheValue{}
	for k, v := range c.data {
		copied[k] = v.copy()
	}
	return &cacheDB{
		data: copied,
	}
}
func (c *cacheDB) commit(writer DatabaseWriter) *Diff {
	result := &Diff{}
	added := [][]byte{}
	updated := []*KV{}
	deleted := []*KV{}
	for key, value := range c.data {
		keyBytes := []byte(key)
		if value.init == nil {
			added = append(added, keyBytes)
			writer.Set(keyBytes, value.value)
			continue
		}
		if value.deleted {
			deleted = append(deleted, &KV{
				Key:   keyBytes,
				Value: value.init,
			})
			writer.Del(keyBytes)
			continue
		}
		if value.dirty {
			updated = append(updated, &KV{
				Key:   keyBytes,
				Value: value.init,
			})
			writer.Set(keyBytes, value.value)
		}
	}
	result.Added = added
	result.Updated = updated
	result.Deleted = deleted
	return result
}

package diffdb

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/db"
)

var (
	testModulePrefix uint32 = 1
	testStorePrefix  uint16 = 0
	testPrefix              = bytes.Join(bytes.FromUint32(1), bytes.FromUint16(0))
	statePrefix             = blockchain.DBPrefixToBytes(blockchain.DBPrefixState)
)

func TestStateStoreGet(t *testing.T) {
	data, _ := db.NewInMemoryDB()
	store := New(data, statePrefix)
	_, exist := store.Get([]byte("random key"))
	assert.False(t, exist)

	// Data exists in DB
	expectedKey := []byte("address")
	expectedValue := []byte("somedata")
	data.Set(bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), expectedKey), expectedValue)
	childStore := store.WithPrefix(testPrefix)
	actualValue, exist := childStore.Get(expectedKey)
	assert.True(t, exist)
	assert.Equal(t, expectedValue, actualValue)

	// Reading again
	actualValue, exist = childStore.Get(expectedKey)
	assert.True(t, exist)
	assert.Equal(t, expectedValue, actualValue)

	// Data deleted
	childStore.Del(expectedKey)
	_, exist = childStore.Get(expectedKey)
	assert.False(t, exist)
}

func TestStateStoreSet(t *testing.T) {
	data, _ := db.NewInMemoryDB()
	expectedKey := []byte("address")
	expectedValue := []byte("somedata")
	data.Set(bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), expectedKey), expectedValue)

	store := New(data, statePrefix)
	// Adding non existing key in DB should cache
	randomKey := []byte("random key")
	randomValue := []byte("random key")
	store.Set(randomKey, randomValue)
	val, exist := store.Get(randomKey)
	assert.True(t, exist)
	assert.Equal(t, randomValue, val)

	// Set existing data, but not in cache
	childStore := store.WithPrefix(testPrefix)
	childStore.Set(expectedKey, []byte("new data"))
	actual, exist, deleted := childStore.cache.get(bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), expectedKey))
	assert.Equal(t, []byte("new data"), actual)
	assert.True(t, exist)
	assert.False(t, deleted)

	// delete the new data
	childStore.Del(expectedKey)
	_, exist = childStore.Get(expectedKey)
	assert.False(t, exist)

	// Set to deleted key again
	childStore.Set(expectedKey, []byte("more new data"))
	actual, exist, deleted = childStore.cache.get(bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), expectedKey))
	assert.Equal(t, []byte("more new data"), actual)
	assert.True(t, exist)
	assert.False(t, deleted)

	// Update existing cache data
	childStore.Set(expectedKey, []byte("even more new data"))
	actual, exist, deleted = childStore.cache.get(bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), expectedKey))
	assert.Equal(t, []byte("even more new data"), actual)
	assert.True(t, exist)
	assert.False(t, deleted)
}

func TestStateStoreDel(t *testing.T) {
	data, _ := db.NewInMemoryDB()
	expectedKey := []byte("address")
	expectedValue := []byte("somedata")
	data.Set(bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), expectedKey), expectedValue)

	store := New(data, statePrefix)
	childStore := store.WithPrefix(testPrefix)

	// delete key which does not exist in cache
	childStore.Del(expectedKey)
	actual, exist, deleted := childStore.cache.get(bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), expectedKey))
	assert.Nil(t, actual)
	assert.False(t, exist)
	assert.True(t, deleted)
}

func TestStateStoreHas(t *testing.T) {
	data, _ := db.NewInMemoryDB()
	expectedKey := crypto.RandomBytes(20)
	expectedValue := []byte("somedata")
	data.Set(bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), expectedKey), expectedValue)

	store := New(data, statePrefix)
	childStore := store.WithPrefix(testPrefix)

	exist := childStore.Has(expectedKey)
	assert.True(t, exist)

	exist = childStore.Has(crypto.RandomBytes(20))
	assert.False(t, exist)
}

func TestStateStoreRange(t *testing.T) {
	data, _ := db.NewInMemoryDB()

	kvs := []struct {
		key []byte
		val []byte
	}{
		{
			key: []byte{0, 0},
			val: []byte("val1"),
		},
		{
			key: []byte{0, 1},
			val: []byte("val2"),
		},
		{
			key: []byte{1, 0},
			val: []byte("val3"),
		},
		{
			key: []byte{1, 1},
			val: []byte("val4"),
		},
	}
	for _, kv := range kvs {
		data.Set(bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), kv.key), kv.val)
	}

	store := New(data, statePrefix)
	childStore := store.WithPrefix(testPrefix)

	res := childStore.Range([]byte{0, 0}, []byte{0, 99}, 2, false)
	assert.Len(t, res, 2)
	assert.Equal(t, []byte{0, 0}, res[0].Key())
	res = childStore.Range([]byte{0, 0}, []byte{0, 99}, 1, true)
	assert.Len(t, res, 1)
	assert.Equal(t, []byte{0, 1}, res[0].Key())

	res = childStore.Iterate([]byte{0}, -1, false)
	assert.Len(t, res, 2)
	assert.Equal(t, []byte{0, 0}, res[0].Key())
	res = childStore.Iterate([]byte{0}, 1, true)
	assert.Len(t, res, 1)
	assert.Equal(t, []byte{0, 1}, res[0].Key())

	res = childStore.Iterate([]byte{3}, 2, false)
	assert.Len(t, res, 0)
}

func TestStateStoreSnapshot(t *testing.T) {
	data, _ := db.NewInMemoryDB()
	expectedKey := []byte("address")
	expectedValue := []byte("somedata")
	data.Set(bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), expectedKey), expectedValue)

	store := New(data, statePrefix)

	childStore := store.WithPrefix(testPrefix)
	childStore.Set(expectedKey, []byte("random value"))
	snapID := childStore.Snapshot()
	childStore.Set(expectedKey, []byte("more random value"))
	val, exist := childStore.Get(expectedKey)
	assert.True(t, exist)
	assert.Equal(t, []byte("more random value"), val)
	childStore.RestoreSnapshot(snapID)
	val, exist = childStore.Get(expectedKey)
	assert.True(t, exist)
	assert.Equal(t, []byte("random value"), val)
	assert.Equal(t, 0, len(childStore.snapshots))
}

func TestStateStoreCommit(t *testing.T) {
	data, _ := db.NewInMemoryDB()
	updatingKey := []byte("update address")
	updatingValue := []byte("update somedata")
	key1 := bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), updatingKey)
	data.Set(key1, updatingValue)
	deletingKey := []byte("delete address")
	deletingValue := []byte("delete somedata")
	key2 := bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), deletingKey)
	data.Set(key2, deletingValue)
	addKey := []byte("new address")
	addValue := []byte("new somedata")
	key3 := bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), addKey)

	store := New(data, statePrefix)

	childStore := store.WithPrefix(testPrefix)
	expectedUpdatedValue := []byte("new value")
	childStore.Set(updatingKey, expectedUpdatedValue)
	childStore.Del(deletingKey)
	childStore.Set(addKey, addValue)

	diff := store.Commit(data)
	// Check Diff
	assert.Equal(t, 1, len(diff.Deleted))
	assert.Equal(t, 1, len(diff.Updated))
	assert.Equal(t, 1, len(diff.Added))
	assert.Equal(t, key1, diff.Updated[0].Key)
	assert.Equal(t, updatingValue, diff.Updated[0].Value, "Value should be original value")
	assert.Equal(t, key2, diff.Deleted[0].Key)
	assert.Equal(t, deletingValue, diff.Deleted[0].Value, "Value should be original value")

	// Check DB
	updated, exist := data.Get(key1)
	assert.True(t, exist)
	assert.Equal(t, expectedUpdatedValue, updated, "Value should be updated value")
	// deleted data should not exist
	keyExist := data.Exist(key2)
	assert.False(t, keyExist, "Deleted data should not exist")
	keyExist = data.Exist(key3)
	assert.True(t, keyExist, "Added data should exist")
}

func TestStateStoreRevert(t *testing.T) {
	// Arrange
	data, _ := db.NewInMemoryDB()
	statePrefix := statePrefix
	diffKey := bytes.Join(blockchain.DBPrefixToBytes(blockchain.DBPrefixStateDiff), bytes.FromUint32(100))
	diff := &Diff{
		Added: [][]byte{
			bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), crypto.RandomBytes(20)),
			bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), crypto.RandomBytes(20)),
			bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), crypto.RandomBytes(20)),
		},
		Updated: []*KV{
			{
				Key:   bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), crypto.RandomBytes(20)),
				Value: crypto.RandomBytes(100),
			},
		},
		Deleted: []*KV{
			{
				Key:   bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), crypto.RandomBytes(20)),
				Value: crypto.RandomBytes(100),
			},
		},
	}
	data.Set(diffKey, diff.Encode())
	for _, key := range diff.Added {
		data.Set(key, crypto.RandomBytes(100))
	}
	for _, kv := range diff.Updated {
		data.Set(kv.Key, crypto.RandomBytes(100))
	}

	store := New(data, []byte{0})

	batch := data.NewBatch()
	store.RevertDiff(batch, diff)
	data.Write(batch)
}

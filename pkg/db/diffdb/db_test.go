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
	_, err := store.Get([]byte("random key"))
	if assert.Error(t, err) {
		assert.Equal(t, ErrNotFound, err)
	}
	// Data exists in DB
	expectedKey := []byte("address")
	expectedValue := []byte("somedata")
	data.Set(bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), expectedKey), expectedValue)
	childStore := store.WithPrefix(testPrefix)
	actualValue, err := childStore.Get(expectedKey)
	assert.Nil(t, err)
	assert.Equal(t, expectedValue, actualValue)

	// Reading again
	actualValue, err = childStore.Get(expectedKey)
	assert.Nil(t, err)
	assert.Equal(t, expectedValue, actualValue)

	// Data deleted
	childStore.Del(expectedKey)
	_, err = childStore.Get(expectedKey)
	if assert.Error(t, err) {
		assert.Equal(t, ErrNotFound, err)
	}
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
	val, err := store.Get(randomKey)
	assert.Nil(t, err)
	assert.Equal(t, randomValue, val)

	// Set existing data, but not in cache
	childStore := store.WithPrefix(testPrefix)
	err = childStore.Set(expectedKey, []byte("new data"))
	assert.Nil(t, err)
	actual, exist, deleted := childStore.cache.get(bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), expectedKey))
	assert.Equal(t, []byte("new data"), actual)
	assert.True(t, exist)
	assert.False(t, deleted)

	// delete the new data
	childStore.Del(expectedKey)
	_, err = childStore.Get(expectedKey)
	if assert.Error(t, err) {
		assert.Equal(t, ErrNotFound, err)
	}
	// Set to deleted key again
	err = childStore.Set(expectedKey, []byte("more new data"))
	assert.Nil(t, err)
	actual, exist, deleted = childStore.cache.get(bytes.Join(statePrefix, bytes.FromUint32(testModulePrefix), bytes.FromUint16(testStorePrefix), expectedKey))
	assert.Equal(t, []byte("more new data"), actual)
	assert.True(t, exist)
	assert.False(t, deleted)

	// Update existing cache data
	err = childStore.Set(expectedKey, []byte("even more new data"))
	assert.Nil(t, err)
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

	exist, err := childStore.Has(expectedKey)
	assert.NoError(t, err)
	assert.True(t, exist)

	exist, err = childStore.Has(crypto.RandomBytes(20))
	assert.NoError(t, err)
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

	res, err := childStore.Range([]byte{0, 0}, []byte{0, 99}, 2, false)
	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assert.Equal(t, []byte{0, 0}, res[0].Key())
	res, err = childStore.Range([]byte{0, 0}, []byte{0, 99}, 1, true)
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, []byte{0, 1}, res[0].Key())

	res, err = childStore.Iterate([]byte{0}, -1, false)
	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assert.Equal(t, []byte{0, 0}, res[0].Key())
	res, err = childStore.Iterate([]byte{0}, 1, true)
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, []byte{0, 1}, res[0].Key())

	res, err = childStore.Iterate([]byte{3}, 2, false)
	assert.NoError(t, err)
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
	val, err := childStore.Get(expectedKey)
	assert.Nil(t, err)
	assert.Equal(t, []byte("more random value"), val)
	childStore.RestoreSnapshot(snapID)
	val, err = childStore.Get(expectedKey)
	assert.Nil(t, err)
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

	diff, err := store.Commit(data)
	assert.Nil(t, err)
	// Check Diff
	assert.Nil(t, err)
	assert.Equal(t, 1, len(diff.Deleted))
	assert.Equal(t, 1, len(diff.Updated))
	assert.Equal(t, 1, len(diff.Added))
	assert.Equal(t, key1, diff.Updated[0].Key)
	assert.Equal(t, updatingValue, diff.Updated[0].Value, "Value should be original value")
	assert.Equal(t, key2, diff.Deleted[0].Key)
	assert.Equal(t, deletingValue, diff.Deleted[0].Value, "Value should be original value")

	// Check DB
	updated, err := data.Get(key1)
	assert.Nil(t, err)
	assert.Equal(t, expectedUpdatedValue, updated, "Value should be updated value")
	// deleted data should not exist
	keyExist, _ := data.Exist(key2)
	assert.False(t, keyExist, "Deleted data should not exist")
	keyExist, _ = data.Exist(key3)
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
	err := data.Set(diffKey, diff.MustEncode())
	assert.NoError(t, err)
	for _, key := range diff.Added {
		err := data.Set(key, crypto.RandomBytes(100))
		assert.NoError(t, err)
	}
	for _, kv := range diff.Updated {
		err := data.Set(kv.Key, crypto.RandomBytes(100))
		assert.NoError(t, err)
	}

	store := New(data, []byte{0})

	batch := data.NewBatch()
	err = store.RevertDiff(batch, diff)
	assert.NoError(t, err)
	err = data.Write(batch)
	assert.NoError(t, err)
}

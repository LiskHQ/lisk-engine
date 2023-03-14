package db

import (
	"encoding/hex"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

func randomTempDir() string {
	return path.Join(os.TempDir(), hex.EncodeToString(crypto.RandomBytes(10)))
}

type dbInterface interface {
	Get(key []byte) ([]byte, bool)
	Set(key, value []byte)
	Exist(key []byte) bool
	Iterate(prefix []byte, limit int, reverse bool) []KeyValue
	IterateKey(prefix []byte, limit int, reverse bool) [][]byte
	IterateRange(start, end []byte, limit int, reverse bool) []KeyValue
	NewReader() *Reader
}

var testData = []struct {
	Key   []byte
	Value []byte
}{
	{
		Key:   []byte{0, 0},
		Value: crypto.RandomBytes(100),
	},
	{
		Key:   []byte{0, 1},
		Value: crypto.RandomBytes(100),
	},
	{
		Key:   []byte{1, 0},
		Value: crypto.RandomBytes(100),
	},
	{
		Key:   []byte{1, 1},
		Value: crypto.RandomBytes(100),
	},
}

func TestDB(t *testing.T) {
	db, err := NewDB(randomTempDir())
	assert.NoError(t, err)
	defer db.Close()

	inmemoryDB, err := NewInMemoryDB()
	assert.NoError(t, err)

	interfaces := []dbInterface{
		db,
		inmemoryDB,
	}

	for _, dbi := range interfaces {
		for _, kv := range testData {
			dbi.Set(kv.Key, kv.Value)
		}

		fetched, exist := dbi.Get(testData[0].Key)
		assert.True(t, exist)
		assert.Equal(t, testData[0].Value, fetched)

		exist = dbi.Exist(testData[0].Key)
		assert.Equal(t, true, exist)

		exist = dbi.Exist(crypto.RandomBytes(5))
		assert.Equal(t, false, exist)

		result := dbi.Iterate([]byte{0}, 1, false)
		assert.Len(t, result, 1)
		assert.Equal(t, testData[0].Key, result[0].Key())
		assert.Equal(t, testData[0].Value, result[0].Value())

		result = dbi.Iterate([]byte{0}, 1, true)
		assert.Len(t, result, 1)
		assert.Equal(t, testData[1].Key, result[0].Key())
		assert.Equal(t, testData[1].Value, result[0].Value())

		result = dbi.Iterate([]byte{0}, -1, true)
		assert.Len(t, result, 2)
		assert.Equal(t, testData[1].Key, result[0].Key())
		assert.Equal(t, testData[1].Value, result[0].Value())

		result = dbi.IterateRange([]byte{0, 1}, []byte{1, 1}, -1, false)
		assert.Len(t, result, 3)
		assert.Equal(t, testData[1].Key, result[0].Key())
		assert.Equal(t, testData[1].Value, result[0].Value())

		result = dbi.IterateRange([]byte{0, 1}, []byte{1, 1}, 2, true)
		assert.Len(t, result, 2)
		assert.Equal(t, testData[3].Key, result[0].Key())
		assert.Equal(t, testData[3].Value, result[0].Value())

		keys := dbi.IterateKey([]byte{0}, -1, true)
		assert.Len(t, keys, 2)
		assert.Equal(t, testData[1].Key, keys[0])

		keys = dbi.IterateKey([]byte{0}, 1, false)
		assert.Len(t, keys, 1)
		assert.Equal(t, testData[0].Key, keys[0])
	}
}

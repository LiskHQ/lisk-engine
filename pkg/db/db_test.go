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
	Get(key []byte) ([]byte, error)
	Set(key, value []byte) error
	Exist(key []byte) (bool, error)
	Iterate(prefix []byte, limit int, reverse bool) ([]KeyValue, error)
	IterateKey(prefix []byte, limit int, reverse bool) ([][]byte, error)
	IterateRange(start, end []byte, limit int, reverse bool) ([]KeyValue, error)
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
			err = dbi.Set(kv.Key, kv.Value)
			assert.NoError(t, err)
		}

		fetched, err := dbi.Get(testData[0].Key)
		assert.NoError(t, err)
		assert.Equal(t, testData[0].Value, fetched)

		exist, err := dbi.Exist(testData[0].Key)
		assert.NoError(t, err)
		assert.Equal(t, true, exist)

		exist, err = dbi.Exist(crypto.RandomBytes(5))
		assert.NoError(t, err)
		assert.Equal(t, false, exist)

		result, err := dbi.Iterate([]byte{0}, 1, false)
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, testData[0].Key, result[0].Key())
		assert.Equal(t, testData[0].Value, result[0].Value())

		result, err = dbi.Iterate([]byte{0}, 1, true)
		assert.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, testData[1].Key, result[0].Key())
		assert.Equal(t, testData[1].Value, result[0].Value())

		result, err = dbi.Iterate([]byte{0}, -1, true)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, testData[1].Key, result[0].Key())
		assert.Equal(t, testData[1].Value, result[0].Value())

		result, err = dbi.IterateRange([]byte{0, 1}, []byte{1, 1}, -1, false)
		assert.NoError(t, err)
		assert.Len(t, result, 3)
		assert.Equal(t, testData[1].Key, result[0].Key())
		assert.Equal(t, testData[1].Value, result[0].Value())

		result, err = dbi.IterateRange([]byte{0, 1}, []byte{1, 1}, 2, true)
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, testData[3].Key, result[0].Key())
		assert.Equal(t, testData[3].Value, result[0].Value())

		keys, err := dbi.IterateKey([]byte{0}, -1, true)
		assert.NoError(t, err)
		assert.Len(t, keys, 2)
		assert.Equal(t, testData[1].Key, keys[0])

		keys, err = dbi.IterateKey([]byte{0}, 1, false)
		assert.NoError(t, err)
		assert.Len(t, keys, 1)
		assert.Equal(t, testData[0].Key, keys[0])
	}
}

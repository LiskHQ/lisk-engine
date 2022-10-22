package db

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

func TestReader(t *testing.T) {
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

		reader1 := dbi.NewReader()
		defer reader1.Close()
		reader2 := dbi.NewReader()
		defer reader2.Close()

		updated := crypto.RandomBytes(32)
		dbi.Set(testData[0].Key, updated)

		val1, err := reader1.Get(testData[0].Key)
		assert.NoError(t, err)
		val2, err := reader2.Get(testData[0].Key)
		assert.NoError(t, err)

		valMain, err := dbi.Get(testData[0].Key)
		assert.NoError(t, err)

		assert.Equal(t, val1, val2)
		assert.NotEqual(t, val1, valMain)
		assert.NotEqual(t, val2, updated)
	}
}

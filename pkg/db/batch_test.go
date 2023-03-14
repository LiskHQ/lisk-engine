package db

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

func TestBatch(t *testing.T) {
	db, err := NewDB(randomTempDir())
	assert.NoError(t, err)
	defer db.Close()
	batch := db.NewBatch()

	key1 := crypto.RandomBytes(38)
	val1 := crypto.RandomBytes(100)
	key2 := crypto.RandomBytes(38)
	val2 := crypto.RandomBytes(100)

	batch.Set(key1, val1)

	batch.Set(key2, val2)

	batch.Del(key1)

	db.Write(batch)

	_, exist := db.Get(key1)
	assert.False(t, exist)

	val, exist := db.Get(key2)
	assert.True(t, exist)
	assert.Equal(t, val2, val)
}

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

	err = batch.Set(key1, val1)
	assert.NoError(t, err)

	err = batch.Set(key2, val2)
	assert.NoError(t, err)

	err = batch.Del(key1)
	assert.NoError(t, err)

	err = db.Write(batch)
	assert.NoError(t, err)

	_, err = db.Get(key1)
	assert.EqualError(t, err, "data was not found")

	val, err := db.Get(key2)
	assert.NoError(t, err)
	assert.Equal(t, val2, val)
}

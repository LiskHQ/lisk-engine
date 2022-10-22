package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInmemoryDB(t *testing.T) {
	db, err := NewInMemoryDB()
	assert.NoError(t, err)
	for _, kv := range testData {
		err := db.Set(kv.Key, kv.Value)
		assert.NoError(t, err)
	}

	err = db.Del(testData[0].Key)
	assert.NoError(t, err)

	exist, err := db.Exist(testData[0].Key)
	assert.NoError(t, err)
	assert.False(t, exist)
}

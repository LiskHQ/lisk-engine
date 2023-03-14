package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInmemoryDB(t *testing.T) {
	db, err := NewInMemoryDB()
	assert.NoError(t, err)
	for _, kv := range testData {
		db.Set(kv.Key, kv.Value)
	}

	db.Del(testData[0].Key)

	exist := db.Exist(testData[0].Key)
	assert.False(t, exist)
}

// Package batchdb provides the prefixed db feature without diff functionality.
package batchdb

import (
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/db"
)

type Database struct {
	database *db.DB
	batch    *db.Batch
	prefix   []byte
}

// BatchDB fetch data from underlying DB and set/del to batch.
func New(database *db.DB, batch *db.Batch) *Database {
	return &Database{
		database: database,
		batch:    batch,
		prefix:   []byte{},
	}
}

func NewWithPrefix(database *db.DB, batch *db.Batch, prefix []byte) *Database {
	return &Database{
		database: database,
		batch:    batch,
		prefix:   prefix,
	}
}

func (b *Database) Get(key []byte) ([]byte, bool) {
	return b.database.Get(b.prefixedKey(key))
}

func (b *Database) Set(key, value []byte) {
	b.batch.Set(b.prefixedKey(key), value)
}

func (b *Database) Del(key []byte) {
	b.batch.Del(b.prefixedKey(key))
}

func (b *Database) prefixedKey(key []byte) []byte {
	return bytes.JoinSize(len(b.prefix)+len(key), b.prefix, key)
}

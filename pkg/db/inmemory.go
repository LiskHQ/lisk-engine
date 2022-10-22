package db

import (
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
)

// NewInMemoryDB returns new instance of in-memory db.
func NewInMemoryDB() (*DB, error) {
	pebbleDB, err := pebble.Open("", &pebble.Options{FS: vfs.NewMem()})
	if err != nil {
		return nil, err
	}
	return &DB{
		pebbleDB: pebbleDB,
	}, nil
}

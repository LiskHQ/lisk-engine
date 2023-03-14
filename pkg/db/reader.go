package db

import (
	"errors"

	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/cockroachdb/pebble"
)

type Reader struct {
	snapshot *pebble.Snapshot
}

func (r *Reader) Get(key []byte) ([]byte, bool) {
	data, closer, err := r.snapshot.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, false
		}
		// unknown error. if this fails, there is a problem in underlying DB which cannot be recovered.
		panic(err)
	}
	copied := bytes.Copy(data)
	if err := closer.Close(); err != nil {
		// if this fails, application should crash otherwise, memory will likely to leak.
		panic(err)
	}
	return copied, true
}

func (r *Reader) Exist(key []byte) bool {
	_, exist := r.Get(key)
	return exist
}

func (r *Reader) IterateKey(prefix []byte, limit int, reverse bool) [][]byte {
	iter := r.snapshot.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound(prefix),
	})
	return iterateKeyPrefix(iter, prefix, limit, reverse)
}

func (r *Reader) Iterate(prefix []byte, limit int, reverse bool) []KeyValue {
	iter := r.snapshot.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound(prefix),
	})
	return iteratePrefix(iter, prefix, limit, reverse)
}

func (r *Reader) IterateRange(start, end []byte, limit int, reverse bool) []KeyValue {
	iter := r.snapshot.NewIter(nil)
	return iterateRange(iter, start, end, limit, reverse)
}

func (r *Reader) Close() error {
	return r.snapshot.Close()
}

package db

import (
	"errors"

	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/cockroachdb/pebble"
)

type Reader struct {
	snapshot *pebble.Snapshot
}

func (r *Reader) Get(key []byte) ([]byte, error) {
	data, closer, err := r.snapshot.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrDataNotFound
		}
		return nil, err
	}
	copied := bytes.Copy(data)
	if err := closer.Close(); err != nil {
		return nil, err
	}
	return copied, nil
}

func (r *Reader) Exist(key []byte) (bool, error) {
	_, err := r.Get(key)
	if err != nil && !errors.Is(err, ErrDataNotFound) {
		return false, err
	}
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (r *Reader) IterateKey(prefix []byte, limit int, reverse bool) ([][]byte, error) {
	iter := r.snapshot.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound(prefix),
	})
	return iterateKeyPrefix(iter, prefix, limit, reverse)
}

func (r *Reader) Iterate(prefix []byte, limit int, reverse bool) ([]KeyValue, error) {
	iter := r.snapshot.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: upperBound(prefix),
	})
	return iteratePrefix(iter, prefix, limit, reverse)
}

func (r *Reader) IterateRange(start, end []byte, limit int, reverse bool) ([]KeyValue, error) {
	iter := r.snapshot.NewIter(nil)
	return iterateRange(iter, start, end, limit, reverse)
}

func (r *Reader) Close() error {
	return r.snapshot.Close()
}

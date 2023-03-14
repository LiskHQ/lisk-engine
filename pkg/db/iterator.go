package db

import (
	"github.com/cockroachdb/pebble"

	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
)

func iterateRange(iter *pebble.Iterator, start, end []byte, limit int, reverse bool) []KeyValue {
	var data []KeyValue
	count := 0
	if !reverse {
		for iter.SeekGE(start); iter.Valid(); iter.Next() {
			key := iter.Key()
			if bytes.Compare(key, end) > 0 {
				break
			}
			kv := &keyValue{
				key:   bytes.Copy(key),
				value: bytes.Copy(iter.Value()),
			}
			data = append(data, kv)
			count++
			if limit != -1 && count >= limit {
				break
			}
		}
	} else {
		for iter.SeekLT(upperBound(end)); iter.Valid(); iter.Prev() {
			key := iter.Key()
			if bytes.Compare(key, start) < 0 {
				break
			}
			kv := &keyValue{
				key:   bytes.Copy(key),
				value: bytes.Copy(iter.Value()),
			}
			data = append(data, kv)
			count++
			if limit != -1 && count >= limit {
				break
			}
		}
	}
	return data
}

func iteratePrefix(iter *pebble.Iterator, prefix []byte, limit int, reverse bool) []KeyValue {
	var data []KeyValue
	count := 0
	if !reverse {
		for iter.First(); iter.Valid(); iter.Next() {
			kv := &keyValue{
				key:   bytes.Copy(iter.Key()),
				value: bytes.Copy(iter.Value()),
			}
			data = append(data, kv)
			count++
			if limit != -1 && count >= limit {
				break
			}
		}
	} else {
		for iter.Last(); iter.Valid(); iter.Prev() {
			kv := &keyValue{
				key:   bytes.Copy(iter.Key()),
				value: bytes.Copy(iter.Value()),
			}
			data = append(data, kv)
			count++
			if limit != -1 && count >= limit {
				break
			}
		}
	}

	if err := iter.Close(); err != nil {
		// iter.Close should never fail. if it fails here, there is a problem in underlying DB which cannot be recovered.
		panic(err)
	}
	return data
}

func iterateKeyPrefix(iter *pebble.Iterator, prefix []byte, limit int, reverse bool) [][]byte {
	var data [][]byte
	count := 0
	if !reverse {
		for iter.First(); iter.Valid(); iter.Next() {
			key := bytes.Copy(iter.Key())
			data = append(data, key)
			count++
			if limit != -1 && count >= limit {
				break
			}
		}
	} else {
		for iter.Last(); iter.Valid(); iter.Prev() {
			key := bytes.Copy(iter.Key())
			data = append(data, key)
			count++
			if limit != -1 && count >= limit {
				break
			}
		}
	}

	if err := iter.Close(); err != nil {
		// iter.Close should never fail. if it fails here, there is a problem in underlying DB which cannot be recovered.
		panic(err)
	}
	return data
}

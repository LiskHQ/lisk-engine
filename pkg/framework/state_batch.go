package framework

import (
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/db"
)

const (
	dbPrefixSize = 1
	prefixSize   = 6 // 4bytes module ID and 2 bytes store prefix
)

var (
	StateDBPrefixState     []byte = []byte{0}
	StateDBPrefixTree      []byte = []byte{1}
	StateDBPrefixDiff      []byte = []byte{2}
	StateDBPrefixTreeState []byte = []byte{3}
	emptyHash                     = crypto.Hash([]byte{})
	hashSize                      = len(emptyHash)
	stateTreeKeySize              = hashSize + prefixSize
)

// stateSMTBatch update batch and prepare key/value pairs for SMT update.
type stateSMTBatch struct {
	batch  *db.Batch
	keys   [][]byte
	values [][]byte
}

func getTreeKey(keyBytes []byte) []byte {
	hashedKey := crypto.Hash(keyBytes[dbPrefixSize+prefixSize:])
	return bytes.JoinSize(hashSize+prefixSize, keyBytes[dbPrefixSize:dbPrefixSize+prefixSize], hashedKey)
}

func newStateBatch(batch *db.Batch) *stateSMTBatch {
	return &stateSMTBatch{
		batch:  batch,
		keys:   [][]byte{},
		values: [][]byte{},
	}
}

func (b *stateSMTBatch) Set(key, value []byte) error {
	if err := b.batch.Set(key, value); err != nil {
		return err
	}
	b.keys = append(b.keys, getTreeKey(key))
	b.values = append(b.values, crypto.Hash(value))
	return nil
}

func (b *stateSMTBatch) Del(key []byte) error {
	if err := b.batch.Del(key); err != nil {
		return err
	}
	b.keys = append(b.keys, getTreeKey(key))
	b.values = append(b.values, emptyHash)
	return nil
}

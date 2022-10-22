package smt_test

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/db"
	"github.com/LiskHQ/lisk-engine/pkg/db/batchdb"
	"github.com/LiskHQ/lisk-engine/pkg/trie/smt"
)

// go test -bench=. -cpuprofile profile.out -blockprofile block.out -memprofile mem.out
// go tool pprof -http :9999 profile.out.
func BenchmarkSMTBatchUpdate(b *testing.B) {
	database, err := db.NewDB(filepath.Join(".", "blockchain.db"))
	if err != nil {
		panic(err)
	}
	defer database.Close()
	// var root []byte
	dataSize := 10000

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		count := 0
		tree := smt.NewTrie(nil, 38)

		keysRaw := make([][]byte, dataSize)
		valuesRaw := make([][]byte, dataSize)
		for i := 0; i < dataSize; i++ {
			keysRaw[i] = bytes.Join(
				bytes.FromUint32(2),
				bytes.FromUint16(0),
				crypto.RandomBytes(20),
			)
			valuesRaw[i] = crypto.RandomBytes(1500)
		}

		keys := make([][]byte, len(keysRaw))
		values := make([][]byte, len(valuesRaw))
		var wg sync.WaitGroup
		for i := 0; i < dataSize; i++ {
			i := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				keys[i] = bytes.Join(keysRaw[i][:6], crypto.Hash(keysRaw[i][6:]))
				values[i] = crypto.Hash(valuesRaw[i])
			}()
		}
		wg.Wait()
		b.StartTimer()
		batch := database.NewBatch()
		batchDB := batchdb.New(database, batch)
		if _, err := tree.Update(batchDB, keys, values); err != nil {
			panic(err)
		}
		if err := database.Write(batch); err != nil {
			panic(err)
		}

		count += dataSize
	}
}

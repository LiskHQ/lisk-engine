// smt runs updates on a database multiple times for benchmark and for debugging.
package main

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/db"
	"github.com/LiskHQ/lisk-engine/pkg/db/batchdb"
	"github.com/LiskHQ/lisk-engine/pkg/trie/smt"
)

func main() {
	database, err := db.NewDB(filepath.Join(".", "state.db"))
	if err != nil {
		panic(err)
	}
	defer database.Close()
	dataSize := 500
	root := []byte{}
	count := 0

	for i := 0; i < 10000; i++ {
		tree := smt.NewTrie(root, 38)

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

		now := time.Now()
		batch := database.NewBatch()
		batchDB := batchdb.New(database, batch)
		var err error
		if root, err = tree.Update(batchDB, keys, values); err != nil {
			panic(err)
		}
		if err := database.Write(batch); err != nil {
			panic(err)
		}
		count += dataSize

		fmt.Printf("Current count %d: %s \n", count, time.Since(now))
	}
}

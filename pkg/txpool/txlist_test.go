package txpool

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

func TestTransactionListAdd(t *testing.T) {
	address := crypto.RandomBytes(20)
	minDiff := uint64(100)
	maxSize := 64
	list := newAddressTransactions(address, maxSize, minDiff)

	tx := &TransactionWithFeePriority{
		receivedAt:  time.Now(),
		FeePriority: 100,
		Transaction: &blockchain.Transaction{
			ID:    crypto.RandomBytes(32),
			Nonce: 0,
			Fee:   10000000000000 + minDiff - 1,
		},
	}
	added, removedID, errMsg := list.Add(tx, true)
	assert.True(t, added)
	assert.Nil(t, removedID)
	assert.Empty(t, errMsg)
	txs := make([]*TransactionWithFeePriority, maxSize-1)
	for i := range txs {
		txs[i] = &TransactionWithFeePriority{
			receivedAt:  time.Now(),
			FeePriority: 100,
			Transaction: &blockchain.Transaction{
				ID:      crypto.RandomBytes(32),
				Module:  fmt.Sprintf("%d", i),
				Command: "transfer",
				Params:  crypto.RandomBytes(20),
				Nonce:   uint64(i) + 1,
				Fee:     10000000000000,
			},
		}
		added, removedID, errMsg := list.Add(txs[i], false)
		assert.True(t, added)
		assert.Nil(t, removedID)
		assert.Empty(t, errMsg)
	}

	// Same tx with the same nonce
	tx = &TransactionWithFeePriority{
		receivedAt:  time.Now(),
		FeePriority: 100,
		Transaction: &blockchain.Transaction{
			ID:    crypto.RandomBytes(32),
			Nonce: 1,
			Fee:   10000000000000 + minDiff - 1,
		},
	}
	added, removedID, errMsg = list.Add(tx, false)
	assert.False(t, added)
	assert.Nil(t, removedID)
	assert.Contains(t, errMsg, "Incoming transaction fee is not sufficient to replace existing transaction")

	// Replace tx with higher fee
	tx = &TransactionWithFeePriority{
		receivedAt:  time.Now(),
		FeePriority: 100,
		Transaction: &blockchain.Transaction{
			ID:    crypto.RandomBytes(32),
			Nonce: 1,
			Fee:   10000000000000 + minDiff,
		},
	}
	added, removedID, errMsg = list.Add(tx, false)
	assert.True(t, added)
	assert.Equal(t, txs[0].ID, removedID)
	assert.Empty(t, errMsg)
	assert.Equal(t, 64, list.Size())

	tx = &TransactionWithFeePriority{
		receivedAt:  time.Now(),
		FeePriority: 100,
		Transaction: &blockchain.Transaction{
			ID:    crypto.RandomBytes(32),
			Nonce: 65,
			Fee:   10000000000000 + minDiff,
		},
	}
	added, removedID, errMsg = list.Add(tx, false)
	assert.False(t, added)
	assert.Nil(t, removedID)
	assert.Equal(t, errMsg, "Incoming transaction exceeds maximum transaction limit per account")
}

func TestTransactionListPromote(t *testing.T) {
	address := crypto.RandomBytes(20)
	minDiff := uint64(100)
	maxSize := 64
	list := newAddressTransactions(address, maxSize, minDiff)

	tx := &TransactionWithFeePriority{
		receivedAt:  time.Now(),
		FeePriority: 100,
		Transaction: &blockchain.Transaction{
			ID:    crypto.RandomBytes(32),
			Nonce: 0,
			Fee:   10000000000000 + minDiff - 1,
		},
	}
	added, removedID, errMsg := list.Add(tx, true)
	assert.True(t, added)
	assert.Nil(t, removedID)
	assert.Empty(t, errMsg)
	txs := make([]*TransactionWithFeePriority, maxSize-1)
	for i := range txs {
		txs[i] = &TransactionWithFeePriority{
			receivedAt:  time.Now(),
			FeePriority: 100,
			Transaction: &blockchain.Transaction{
				ID:      crypto.RandomBytes(32),
				Module:  fmt.Sprintf("%d", i),
				Command: "transfer",
				Params:  crypto.RandomBytes(20),
				Nonce:   uint64(i) + 1,
				Fee:     10000000000000,
			},
		}
		added, removedID, errMsg := list.Add(txs[i], false)
		assert.True(t, added)
		assert.Nil(t, removedID)
		assert.Empty(t, errMsg)
	}

	assert.Len(t, list.GetProcessables(), 1)
	assert.Len(t, list.GetUnprocessables(), 63)

	promotables := list.GetPromotable()
	ok := list.Promote(promotables)
	assert.True(t, ok)
	assert.Len(t, list.GetProcessables(), 64)
	assert.Len(t, list.GetUnprocessables(), 0)
}

func TestTransactionListRemove(t *testing.T) {
	// Arrange full list with all processable
	address := crypto.RandomBytes(20)
	minDiff := uint64(100)
	maxSize := 64
	list := newAddressTransactions(address, maxSize, minDiff)

	tx := &TransactionWithFeePriority{
		receivedAt:  time.Now(),
		FeePriority: 100,
		Transaction: &blockchain.Transaction{
			ID:    crypto.RandomBytes(32),
			Nonce: 0,
			Fee:   10000000000000 + minDiff - 1,
		},
	}
	added, removedID, errMsg := list.Add(tx, true)
	assert.True(t, added)
	assert.Nil(t, removedID)
	assert.Empty(t, errMsg)
	txs := make([]*TransactionWithFeePriority, maxSize-1)
	for i := range txs {
		txs[i] = &TransactionWithFeePriority{
			receivedAt:  time.Now(),
			FeePriority: 100,
			Transaction: &blockchain.Transaction{
				ID:      crypto.RandomBytes(32),
				Module:  fmt.Sprintf("%d", i),
				Command: "transfer",
				Params:  crypto.RandomBytes(20),
				Nonce:   uint64(i) + 1,
				Fee:     10000000000000,
			},
		}
		added, removedID, errMsg := list.Add(txs[i], false)
		assert.True(t, added)
		assert.Nil(t, removedID)
		assert.Empty(t, errMsg)
	}
	promotables := list.GetPromotable()
	ok := list.Promote(promotables)
	assert.True(t, ok)

	removedID = list.Remove(99)
	assert.Nil(t, removedID)

	removedID = list.Remove(10)
	assert.Equal(t, txs[9].ID, removedID)
	assert.Len(t, list.GetProcessables(), 10)
	assert.Len(t, list.GetUnprocessables(), 53)

	_, exist := list.Get(10)
	assert.False(t, exist)
	_, exist = list.Get(11)
	assert.True(t, exist)
}

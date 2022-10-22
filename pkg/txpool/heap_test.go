package txpool

import (
	"container/heap"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

func TestNonceMinHeap(t *testing.T) {
	initVal := NonceMinHeap{3, 2, 1, 5}
	heap.Init(&initVal)

	heap.Push(&initVal, uint64(4))

	val := heap.Pop(&initVal).(uint64)
	assert.Equal(t, uint64(1), val)
	val = heap.Pop(&initVal).(uint64)
	assert.Equal(t, uint64(2), val)
	val = heap.Pop(&initVal).(uint64)
	assert.Equal(t, uint64(3), val)
	val = heap.Pop(&initVal).(uint64)
	assert.Equal(t, uint64(4), val)
}

func TestFeePriorityMaxHeap(t *testing.T) {
	initVal := FeeMaxHeap{
		{
			Transaction: &blockchain.Transaction{
				ID: codec.Hex{1},
			},
			FeePriority: 100,
		},
		{
			Transaction: &blockchain.Transaction{
				ID: codec.Hex{2},
			},
			FeePriority: 200,
		},
		{
			Transaction: &blockchain.Transaction{
				ID: codec.Hex{3},
			},
			FeePriority: 150,
		},
	}
	heap.Init(&initVal)

	heap.Push(&initVal, &TransactionWithFeePriority{
		FeePriority: 170,
		Transaction: &blockchain.Transaction{
			ID: codec.Hex{4},
		},
	})

	val := heap.Pop(&initVal).(*TransactionWithFeePriority)
	assert.Equal(t, codec.Hex{2}, val.ID)
	val = heap.Pop(&initVal).(*TransactionWithFeePriority)
	assert.Equal(t, codec.Hex{4}, val.ID)
}

func TestFeeMinHeap(t *testing.T) {
	initVal := FeeMinHeap{
		{
			Transaction: &blockchain.Transaction{
				ID: codec.Hex{1},
			},
			FeePriority: 100,
		},
		{
			Transaction: &blockchain.Transaction{
				ID: codec.Hex{2},
			},
			FeePriority: 200,
		},
		{
			Transaction: &blockchain.Transaction{
				ID: codec.Hex{3},
			},
			FeePriority: 150,
		},
	}
	heap.Init(&initVal)

	heap.Push(&initVal, &TransactionWithFeePriority{
		FeePriority: 170,
		Transaction: &blockchain.Transaction{
			ID: codec.Hex{4},
		},
	})

	val := heap.Pop(&initVal).(*TransactionWithFeePriority)
	assert.Equal(t, codec.Hex{1}, val.ID)
	val = heap.Pop(&initVal).(*TransactionWithFeePriority)
	assert.Equal(t, codec.Hex{3}, val.ID)
}

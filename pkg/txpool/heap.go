package txpool

import (
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
)

type NonceMinHeap []uint64

func (h NonceMinHeap) Len() int { return len(h) }
func (h NonceMinHeap) Less(i, j int) bool {
	return h[i] < h[j]
}
func (h NonceMinHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *NonceMinHeap) Push(x interface{}) {
	*h = append(*h, x.(uint64))
}
func (h *NonceMinHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type TransactionWithFeePriority struct {
	*blockchain.Transaction
	FeePriority uint64
	receivedAt  time.Time
}

type FeeMaxHeap []*TransactionWithFeePriority

func (h FeeMaxHeap) Len() int { return len(h) }
func (h FeeMaxHeap) Less(i, j int) bool {
	return h[i].FeePriority > h[j].FeePriority
}
func (h FeeMaxHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *FeeMaxHeap) Push(x interface{}) {
	*h = append(*h, x.(*TransactionWithFeePriority))
}
func (h *FeeMaxHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type FeeMinHeap []*TransactionWithFeePriority

func (h FeeMinHeap) Len() int { return len(h) }
func (h FeeMinHeap) Less(i, j int) bool {
	return h[i].FeePriority < h[j].FeePriority
}
func (h FeeMinHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *FeeMinHeap) Push(x interface{}) {
	*h = append(*h, x.(*TransactionWithFeePriority))
}
func (h *FeeMinHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

package generator

import (
	"sort"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
)

type TransactionWithFeePriority struct {
	*blockchain.Transaction
	FeePriority int
}

type FeePriorityTransactions []*TransactionWithFeePriority

func (h FeePriorityTransactions) Len() int { return len(h) }
func (h FeePriorityTransactions) Less(i, j int) bool {
	return uint64(h[i].FeePriority) > uint64(h[j].FeePriority)
}
func (h FeePriorityTransactions) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *FeePriorityTransactions) Push(x interface{}) {
	*h = append(*h, x.(*TransactionWithFeePriority))
}
func (h *FeePriorityTransactions) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func getSortedTransactionMapByNonce(transactions []*blockchain.Transaction) map[string][]*TransactionWithFeePriority {
	result := map[string][]*TransactionWithFeePriority{}
	for _, tx := range transactions {
		senderMap, exist := result[string(tx.SenderAddress())]
		if !exist {
			senderMap = []*TransactionWithFeePriority{}
		}
		priority := tx.Fee / uint64(tx.Size())
		senderMap = append(senderMap, &TransactionWithFeePriority{
			Transaction: tx,
			FeePriority: int(priority),
		})
		result[string(tx.SenderAddress())] = senderMap
	}
	for key, val := range result {
		sort.Slice(val, func(i, j int) bool { return val[i].Nonce < val[j].Nonce })
		result[key] = val
	}
	return result
}

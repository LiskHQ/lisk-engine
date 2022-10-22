package txpool

import (
	"bytes"
	"container/heap"
	"sort"
	"sync"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

type addressTransactions struct {
	address                     codec.Lisk32
	maxSize                     int
	minReplacementFeeDifference uint64
	transactions                map[uint64]*TransactionWithFeePriority
	processables                []uint64
	nonces                      NonceMinHeap
	mutex                       *sync.Mutex
}

func newAddressTransactions(address codec.Lisk32, maxSize int, minReplacementFeeDiff uint64) *addressTransactions {
	addrTxs := &addressTransactions{
		address:                     address,
		maxSize:                     maxSize,
		minReplacementFeeDifference: minReplacementFeeDiff,
		transactions:                map[uint64]*TransactionWithFeePriority{},
		processables:                []uint64{},
		nonces:                      NonceMinHeap{},
		mutex:                       new(sync.Mutex),
	}
	heap.Init(&addrTxs.nonces)
	return addrTxs
}

func (a *addressTransactions) Get(nonce uint64) (*TransactionWithFeePriority, bool) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	tx, ok := a.transactions[nonce]
	return tx, ok
}

func (a *addressTransactions) Size() int {
	return len(a.nonces)
}

func (a *addressTransactions) GetProcessables() []*TransactionWithFeePriority {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	result := []*TransactionWithFeePriority{}
	for _, nonce := range a.processables {
		tx := a.transactions[nonce]
		result = append(result, tx)
	}
	return result
}

func (a *addressTransactions) GetUnprocessables() []*TransactionWithFeePriority {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	result := []*TransactionWithFeePriority{}
	if len(a.nonces) == 0 {
		return result
	}
	if len(a.nonces) == len(a.processables) {
		return result
	}
	copied := make(NonceMinHeap, len(a.nonces))
	copy(copied, a.nonces)
	for range a.processables {
		heap.Pop(&copied)
	}
	remaining := len(copied)
	for i := 0; i < remaining; i++ {
		nonce := heap.Pop(&copied).(uint64)
		tx := a.transactions[nonce]
		result = append(result, tx)
	}

	return result
}

func (a *addressTransactions) Add(incomingTx *TransactionWithFeePriority, processable bool) (ok bool, idRemoved codec.Hex, errMsg string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	existingTx, exist := a.transactions[incomingTx.Nonce]
	if exist {
		if incomingTx.Fee < existingTx.Fee+a.minReplacementFeeDifference {
			return false, nil, "Incoming transaction fee is not sufficient to replace existing transaction"
		}
		a.demoteAfter(incomingTx.Nonce)
		a.transactions[incomingTx.Nonce] = incomingTx
		return true, existingTx.ID, ""
	}
	var removedID codec.Hex
	if len(a.nonces)+1 > a.maxSize {
		maxNonce := a.maxNonce()
		if incomingTx.Nonce > maxNonce {
			return false, nil, "Incoming transaction exceeds maximum transaction limit per account"
		}
		removedID = a.remove(maxNonce)
	}
	a.transactions[incomingTx.Nonce] = incomingTx
	heap.Push(&a.nonces, incomingTx.Nonce)
	if processable && len(a.processables) == 0 {
		a.processables = append(a.processables, incomingTx.Nonce)
	}
	return true, removedID, ""
}

func (a *addressTransactions) Remove(target uint64) codec.Hex {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.remove(target)
}

func (a *addressTransactions) Promote(txs []*TransactionWithFeePriority) bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	promiting := []uint64{}
	for _, tx := range txs {
		existing, exist := a.transactions[tx.Nonce]
		if !exist {
			return false
		}
		if !bytes.Equal(existing.ID, tx.ID) {
			return false
		}
		promiting = append(promiting, tx.Nonce)
	}
	a.processables = append(a.processables, promiting...)
	nonceMap := map[uint64]bool{}
	for _, nonce := range a.processables {
		nonceMap[nonce] = true
	}
	uniqueNonce := make([]uint64, len(nonceMap))
	index := 0
	for nonce := range nonceMap {
		uniqueNonce[index] = nonce
		index++
	}
	sort.Slice(uniqueNonce, func(i, j int) bool { return uniqueNonce[i] < uniqueNonce[j] })
	a.processables = uniqueNonce
	return true
}

func (a *addressTransactions) GetPromotable() []*TransactionWithFeePriority {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	result := []*TransactionWithFeePriority{}
	if len(a.nonces) == 0 {
		return result
	}
	if len(a.nonces) == len(a.processables) {
		return result
	}
	copied := make(NonceMinHeap, len(a.nonces))
	copy(copied, a.nonces)
	for range a.processables {
		heap.Pop(&copied)
	}
	remaining := len(copied)
	if remaining == 0 {
		return result
	}
	firstUnprocessable := heap.Pop(&copied).(uint64)
	if len(a.processables) != 0 {
		highestProcessable := a.processables[len(a.processables)-1]
		if firstUnprocessable != highestProcessable+1 {
			return result
		}
	}
	firstPromitable := a.transactions[firstUnprocessable]
	result = append(result, firstPromitable)
	// update remaining
	remaining = len(copied)
	lastPromited := firstUnprocessable
	for i := 0; i < remaining; i++ {
		next := heap.Pop(&copied).(uint64)
		if next != lastPromited+1 {
			return result
		}
		promotable := a.transactions[next]
		lastPromited = next
		result = append(result, promotable)
	}

	return result
}

func (a *addressTransactions) remove(target uint64) codec.Hex {
	existingTx, exist := a.transactions[target]
	if !exist {
		return nil
	}
	newHeap := NonceMinHeap{}
	for _, nonce := range a.nonces {
		if nonce != target {
			newHeap = append(newHeap, nonce)
		}
	}
	heap.Init(&newHeap)
	a.nonces = newHeap
	a.demoteAfter(target)
	delete(a.transactions, target)
	return existingTx.ID
}

func (a *addressTransactions) demoteAfter(target uint64) {
	newProcessables := []uint64{}
	for _, nonce := range a.processables {
		if nonce < target {
			newProcessables = append(newProcessables, nonce)
		}
	}
	sort.Slice(newProcessables, func(i, j int) bool { return newProcessables[i] < newProcessables[j] })
	a.processables = newProcessables
}

func (a *addressTransactions) maxNonce() uint64 {
	max := uint64(0)
	for _, nonce := range a.nonces {
		if nonce > max {
			max = nonce
		}
	}
	return max
}

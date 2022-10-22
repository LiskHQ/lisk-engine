package txpool

import "github.com/LiskHQ/lisk-engine/pkg/blockchain"

func calculateFeePriority(tx *blockchain.Transaction) uint64 {
	return tx.Fee / uint64(tx.Size())
}

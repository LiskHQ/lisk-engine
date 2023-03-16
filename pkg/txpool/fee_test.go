package txpool

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

func TestCalculateFeePriority(t *testing.T) {
	tx := &blockchain.Transaction{
		Module:          "token",
		Command:         "transfer",
		Nonce:           100,
		Fee:             100000000,
		SenderPublicKey: crypto.RandomBytes(32),
		Params:          crypto.RandomBytes(100),
		Signatures:      []codec.Hex{crypto.RandomBytes(64)},
	}
	tx.Init()
	size := tx.Size()

	priority := calculateFeePriority(tx)

	assert.Equal(t, tx.Fee/uint64(size), priority)
}

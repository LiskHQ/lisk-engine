package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

func TestGenesisBlock(t *testing.T) {
	prevID := crypto.RandomBytes(32)
	genesis := NewGenesisBlock(123, 1234567890, prevID, BlockAssets{
		{
			Module: "token",
			Data:   crypto.RandomBytes(100),
		},
		{
			Module: "dpos",
			Data:   crypto.RandomBytes(20),
		},
	})
	assert.Equal(t, uint32(0), genesis.Header.Version)
	assert.Equal(t, uint32(123), genesis.Header.Height)
	assert.Equal(t, uint32(1234567890), genesis.Header.Timestamp)
	assert.Equal(t, codec.Hex(prevID), genesis.Header.PreviousBlockID)
	assert.Equal(t, uint32(123), genesis.Header.MaxHeightPrevoted)
	assert.Equal(t, uint32(0), genesis.Header.MaxHeightGenerated)
	assert.Equal(t, codec.Hex{}, genesis.Header.Signature)
	assert.Equal(t, codec.Lisk32(bytes.Repeat([]byte{0}, AddressLength)), genesis.Header.GeneratorAddress)
	assert.Equal(t, codec.Hex(emptyHash), genesis.Header.TransactionRoot)
	assert.Equal(t, uint32(0), genesis.Header.AggregateCommit.Height)
	assert.Equal(t, codec.Hex{}, genesis.Header.AggregateCommit.AggregationBits)
	assert.Equal(t, codec.Hex{}, genesis.Header.AggregateCommit.CertificateSignature)
	assert.Equal(t, "dpos", genesis.Assets[0].Module)
	assert.Len(t, genesis.Transactions, 0)
}

package blockchain

import (
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/trie/rmt"
)

func createRandomBlock(height uint32) *Block {
	block := &Block{
		Header: &BlockHeader{
			Version:            2,
			Height:             height,
			Timestamp:          0,
			PreviousBlockID:    crypto.RandomBytes(32),
			GeneratorAddress:   crypto.RandomBytes(20),
			TransactionRoot:    crypto.RandomBytes(32),
			EventRoot:          crypto.RandomBytes(32),
			AssetRoot:          crypto.RandomBytes(32),
			StateRoot:          crypto.RandomBytes(32),
			MaxHeightPrevoted:  0,
			MaxHeightGenerated: 0,
			ImpliesMaxPrevotes: false,
			ValidatorsHash:     crypto.RandomBytes(32),
			AggregateCommit: &AggregateCommit{
				Height:               height,
				AggregationBits:      crypto.RandomBytes(32),
				CertificateSignature: crypto.RandomBytes(32),
			},
			Signature: crypto.RandomBytes(64),
		},
		Assets: BlockAssets{
			{
				Module: "dpos",
				Data:   crypto.RandomBytes(100),
			},
			{
				Module: "token",
				Data:   crypto.RandomBytes(100),
			},
		},
		Transactions: []*Transaction{
			{
				Module:          "token",
				Command:         "crossChainTransfer",
				Nonce:           34,
				Fee:             10000000000000000000,
				SenderPublicKey: crypto.RandomBytes(32),
				Params:          crypto.RandomBytes(100),
				Signatures:      []codec.Hex{crypto.RandomBytes(64), crypto.RandomBytes(64)},
			},
			{
				Module:          "sample",
				Command:         "command",
				Nonce:           2,
				Fee:             10000000000000000000,
				SenderPublicKey: crypto.RandomBytes(32),
				Params:          crypto.RandomBytes(100),
				Signatures:      []codec.Hex{crypto.RandomBytes(64)},
			},
		},
	}
	block.Init()

	txIDs := make([][]byte, len(block.Transactions))
	for i, tx := range block.Transactions {
		txIDs[i] = tx.ID
	}
	block.Header.TransactionRoot = rmt.CalculateRoot(txIDs)
	assertRoot := BlockAssets(block.Assets).GetRoot()

	block.Header.AssetRoot = assertRoot
	block.Init()
	return block
}

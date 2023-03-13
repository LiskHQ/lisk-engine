package blockchain

import (
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

const (
	GenesisBlockVersion = 0
)

// NewGenesisBlock returns a valid genesis block but without validatorsHash and stateRoot.
func NewGenesisBlock(height, timestamp uint32, previoudBlockID codec.Hex, assets BlockAssets) (*Block, error) {
	assets.Sort()
	assetRoot, err := assets.GetRoot()
	if err != nil {
		return nil, err
	}
	return &Block{
		Header: &BlockHeader{
			Version:            GenesisBlockVersion,
			PreviousBlockID:    previoudBlockID,
			Height:             height,
			Timestamp:          timestamp,
			GeneratorAddress:   bytes.Repeat([]byte{0}, AddressLength),
			MaxHeightGenerated: 0,
			MaxHeightPrevoted:  height,
			Signature:          []byte{},
			TransactionRoot:    crypto.Hash([]byte{}),
			AssetRoot:          assetRoot,
			AggregateCommit: &AggregateCommit{
				Height:               0,
				AggregationBits:      []byte{},
				CertificateSignature: []byte{},
			},
		},
		Assets:       assets,
		Transactions: []*Transaction{},
	}, nil
}

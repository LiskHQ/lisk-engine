package blockchain

import (
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

var (
	TagBlockHeader = []byte("LSK_BH_")
	TagTransaction = []byte("LSK_TX_")
)

func ValidateBlockSignature(publicKey, signature, chainID, signingBytes []byte) bool {
	message := bytes.Join(TagBlockHeader, chainID, signingBytes)
	err := crypto.VerifySignature(publicKey, signature, crypto.Hash(message))
	return err == nil
}

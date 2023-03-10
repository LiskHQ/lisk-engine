package mock

import (
	"fmt"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

func verifySignatures(userAccount *UserAccount, chainID []byte, transaction blockchain.FrozenTransaction) error {
	signingBytes, err := transaction.SigningBytes()
	if err != nil {
		return err
	}
	networkSigningBytes := crypto.Hash(bytes.Join(blockchain.TagTransaction, chainID, signingBytes))
	if len(transaction.Signatures()) != 1 {
		return fmt.Errorf("non multi-signature account must have signature of length 1 but received %d", len(transaction.Signatures()))
	}
	if err := crypto.VerifySignature(transaction.SenderPublicKey(), transaction.Signatures()[0], networkSigningBytes); err != nil {
		return err
	}
	return nil
}

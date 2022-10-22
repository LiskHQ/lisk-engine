package auth

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
	if !userAccount.IsMultisignatureAccount() {
		if len(transaction.Signatures()) != 1 {
			return fmt.Errorf("non multi-signature account must have signature of length 1 but received %d", len(transaction.Signatures()))
		}
		if err := crypto.VerifySignature(transaction.SenderPublicKey(), transaction.Signatures()[0], networkSigningBytes); err != nil {
			return err
		}
		return nil
	}
	nonEmptySignaturesCount := 0
	for _, signature := range transaction.Signatures() {
		if len(signature) != 0 {
			nonEmptySignaturesCount++
		}
	}
	if nonEmptySignaturesCount != int(userAccount.NumberOfSignatures) {
		return fmt.Errorf("number of signature required is %d but received %d", userAccount.NumberOfSignatures, nonEmptySignaturesCount)
	}
	if len(transaction.Signatures()) != len(userAccount.MandatoryKeys)+len(userAccount.OptionalKeys) {
		return fmt.Errorf("lengh of signatures must be %d", len(userAccount.MandatoryKeys)+len(userAccount.OptionalKeys))
	}
	for i, key := range userAccount.MandatoryKeys {
		if err := crypto.VerifySignature(key, transaction.Signatures()[i], networkSigningBytes); err != nil {
			return err
		}
	}
	for i, key := range userAccount.OptionalKeys {
		signature := transaction.Signatures()[len(userAccount.MandatoryKeys)+i]
		if len(signature) == 0 {
			continue
		}
		if err := crypto.VerifySignature(key, signature, networkSigningBytes); err != nil {
			return err
		}
	}
	return nil
}

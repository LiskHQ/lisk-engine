package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tyler-smith/go-bip39"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

func TestTransactionValidate(t *testing.T) {
	entropy, _ := bip39.NewEntropy(256)
	passphrase, _ := bip39.NewMnemonic(entropy)
	pk, sk, _ := crypto.GetKeys(passphrase)
	address := crypto.GetAddress(pk)
	networkID := crypto.RandomBytes(32)
	validTx := &Transaction{
		Module:          "token",
		Command:         "transfer",
		Nonce:           1,
		Fee:             100000000,
		SenderPublicKey: pk,
		Params:          crypto.RandomBytes(100),
	}
	signature := validTx.GetSignature(networkID, sk)
	validTx.Signatures = []codec.Hex{signature}
	validTx.Init()

	assert.Equal(t, codec.Lisk32(address), validTx.SenderAddress())

	invalidTx := validTx.Copy()
	invalidTx.Module = "dpos!!!"
	assert.Error(t, invalidTx.Validate(), "Module must be alpha numeric")

	invalidTx = validTx.Copy()
	invalidTx.Command = "_vote_"
	assert.Error(t, invalidTx.Validate(), "Command must be alpha numeric")

	invalidTx = validTx.Copy()
	invalidTx.Params = crypto.RandomBytes(MaxTransactionParamsSize + 1)
	assert.Error(t, invalidTx.Validate(), "Transaction params must be less than max transaction params size")

	invalidTx = validTx.Copy()
	invalidTx.SenderPublicKey = crypto.RandomBytes(10)
	assert.Error(t, invalidTx.Validate(), "SenderPublicKey must have length of 32 but received")

	invalidTx = validTx.Copy()
	invalidTx.Signatures = []codec.Hex{}
	assert.Error(t, invalidTx.Validate(), "Signatures must have length at least 1 but received")

	invalidTx = validTx.Copy()
	invalidTx.Signatures = []codec.Hex{crypto.RandomBytes(64), crypto.RandomBytes(32)}
	assert.Error(t, invalidTx.Validate(), "Signatures must have length of 64 but received")
}

func TestFrozenTransaction(t *testing.T) {
	entropy, _ := bip39.NewEntropy(256)
	passphrase, _ := bip39.NewMnemonic(entropy)
	pk, sk, _ := crypto.GetKeys(passphrase)
	networkID := crypto.RandomBytes(32)
	validTx := &Transaction{
		Module:          "token",
		Command:         "tranfer",
		Nonce:           1,
		Fee:             100000000,
		SenderPublicKey: pk,
		Params:          crypto.RandomBytes(100),
	}
	signature := validTx.GetSignature(networkID, sk)
	validTx.Signatures = []codec.Hex{signature}
	validTx.Init()

	frozen := validTx.Freeze()
	assert.Equal(t, validTx.Module, frozen.Module())
	assert.Equal(t, validTx.Command, frozen.Command())
	assert.Equal(t, validTx.SenderAddress(), frozen.SenderAddress())
	assert.Equal(t, validTx.Nonce, frozen.Nonce())
	assert.Equal(t, validTx.SenderPublicKey, frozen.SenderPublicKey())
	assert.Equal(t, validTx.Fee, frozen.Fee())
	assert.Equal(t, validTx.size, frozen.Size())

	signing := frozen.SigningBytes()
	validSigning := validTx.SigningBytes()

	assert.Equal(t, validSigning, signing)
}

func TestTransactionCodec(t *testing.T) {
	validTx := &Transaction{
		Module:          "token",
		Command:         "transfer",
		Nonce:           1,
		Fee:             100000000,
		SenderPublicKey: crypto.RandomBytes(32),
		Params:          crypto.RandomBytes(100),
		Signatures: []codec.Hex{
			crypto.RandomBytes(64),
			crypto.RandomBytes(64),
			crypto.RandomBytes(64),
			crypto.RandomBytes(64),
			crypto.RandomBytes(64),
			crypto.RandomBytes(64),
		},
	}

	encoded := validTx.Encode()

	signingTx := &SigningTransaction{}
	err := signingTx.Decode(encoded)
	assert.NoError(t, err)

	// This should not panic
	signingTx.MustDecode(encoded)
	signingTx.Encode()
}

func FuzzTransactionCodec(f *testing.F) {
	f.Add(crypto.RandomBytes(500))
	f.Fuzz(func(t *testing.T, randomBytes []byte) {
		tx := &Transaction{}
		err := tx.DecodeStrict(randomBytes)
		if err == nil {
			tx.Encode()
		}
	})
}

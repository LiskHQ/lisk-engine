package certificate

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tyler-smith/go-bip39"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

func newPassphrase() string {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		panic(err)
	}
	pass, err := bip39.NewMnemonic(entropy)
	if err != nil {
		panic(err)
	}
	return pass
}

func TestCertificateSign(t *testing.T) {
	certificate := NewCertificateFromBlock(&blockchain.BlockHeader{
		ID:             crypto.RandomBytes(32),
		Height:         100,
		StateRoot:      crypto.RandomBytes(32),
		ValidatorsHash: crypto.RandomBytes(32),
	})

	newCert := &Certificate{}
	err := newCert.Decode(certificate.MustEncode())
	assert.NoError(t, err)

	keys := crypto.BLSKeyGen([]byte(newPassphrase()))
	networkID := crypto.RandomBytes(32)

	err = certificate.Sign(networkID, keys.PrivateKey)
	assert.NoError(t, err)

	valid, err := certificate.Verify(networkID, certificate.Signature, keys.PublicKey)
	assert.NoError(t, err)
	assert.Equal(t, valid, true)

	assert.True(t, Certificate{
		BlockID:         []byte{},
		Timestamp:       0,
		StateRoot:       []byte{},
		ValidatorsHash:  []byte{},
		AggregationBits: []byte{},
		Signature:       []byte{},
	}.Empty())
}

func TestSingleCommit(t *testing.T) {
	networkID := crypto.RandomBytes(32)

	passphrases := []string{}
	for i := 0; i < 3; i++ {
		passphrases = append(passphrases, newPassphrase())
	}

	keypairs := make(AddressKeyPairs, len(passphrases))
	for i, pass := range passphrases {
		publicKey, _, err := crypto.GetKeys(pass)
		assert.NoError(t, err)
		addr := crypto.GetAddress(publicKey)
		keys := crypto.BLSKeyGen([]byte(pass))
		keypairs[i] = &AddressKeyPair{
			Address: addr,
			BLSKey:  keys.PublicKey,
		}
	}

	commits := make(SingleCommits, len(passphrases))
	header := &blockchain.BlockHeader{
		ID:             crypto.RandomBytes(32),
		Height:         100,
		Timestamp:      123,
		StateRoot:      crypto.RandomBytes(32),
		ValidatorsHash: crypto.RandomBytes(32),
	}
	for i, pass := range passphrases {
		publicKey, _, err := crypto.GetKeys(pass)
		assert.NoError(t, err)
		addr := crypto.GetAddress(publicKey)
		keys := crypto.BLSKeyGen([]byte(pass))

		commits[i], _ = NewSingleCommit(header, codec.Lisk32(addr), networkID, keys.PrivateKey)
		newcommit := &SingleCommit{}
		err = newcommit.Decode(commits[i].MustEncode())
		assert.NoError(t, err)
	}

	aggCommit, err := commits.Aggregate(keypairs)
	assert.NoError(t, err)

	cert := &Certificate{
		BlockID:         header.ID,
		Height:          header.Height,
		Timestamp:       header.Timestamp,
		StateRoot:       header.StateRoot,
		ValidatorsHash:  header.ValidatorsHash,
		AggregationBits: aggCommit.AggregationBits,
		Signature:       aggCommit.CertificateSignature,
	}

	valid, err := cert.VerifyAggregateCertificateSignature(keypairs.BLSKeys(), []uint64{1, 1, 1}, 2, networkID)
	assert.NoError(t, err)
	assert.Equal(t, true, valid)
}

func TestSingleCommit_Validate(t *testing.T) {
	invalidSingleCommits := []*SingleCommit{
		{
			blockID:              crypto.RandomBytes(blockchain.IDLength - 1),
			height:               99,
			validatorAddress:     crypto.RandomBytes(blockchain.AddressLength),
			certificateSignature: crypto.RandomBytes(crypto.BLSSignatureLength),
		},
		{
			blockID:              crypto.RandomBytes(blockchain.IDLength),
			height:               99,
			validatorAddress:     crypto.RandomBytes(blockchain.AddressLength + 1),
			certificateSignature: crypto.RandomBytes(crypto.BLSSignatureLength),
		},
		{
			blockID:              crypto.RandomBytes(blockchain.IDLength),
			height:               99,
			validatorAddress:     crypto.RandomBytes(blockchain.AddressLength),
			certificateSignature: crypto.RandomBytes(crypto.BLSSignatureLength + 1),
		},
	}

	for _, commit := range invalidSingleCommits {
		assert.Error(t, commit.Validate(), "invalid single commit should have error")
	}
	validSingleCommits := []*SingleCommit{
		{
			blockID:              crypto.RandomBytes(blockchain.IDLength),
			height:               99,
			validatorAddress:     crypto.RandomBytes(blockchain.AddressLength),
			certificateSignature: crypto.RandomBytes(crypto.BLSSignatureLength),
		},
	}
	for _, commit := range validSingleCommits {
		assert.NoError(t, commit.Validate(), "valid single commit should not have error")
	}
}

func TestAddressKeyPair(t *testing.T) {
	passphrases := []string{}
	for i := 0; i < 3; i++ {
		passphrases = append(passphrases, newPassphrase())
	}

	keypairs := make(AddressKeyPairs, len(passphrases))
	for i, pass := range passphrases {
		publicKey, _, err := crypto.GetKeys(pass)
		assert.NoError(t, err)
		addr := crypto.GetAddress(publicKey)
		keys := crypto.BLSKeyGen([]byte(pass))
		keypairs[i] = &AddressKeyPair{
			Address: addr,
			BLSKey:  keys.PublicKey,
		}
	}

	publicKey, _, err := crypto.GetKeys(passphrases[0])
	assert.NoError(t, err)
	addr := crypto.GetAddress(publicKey)

	blsKey, exist := keypairs.BLSKey(addr)
	assert.True(t, exist)
	assert.Equal(t, blsKey, crypto.BLSKeyGen([]byte(passphrases[0])).PublicKey)
	_, exist = keypairs.BLSKey(crypto.RandomBytes(20))
	assert.False(t, exist)

	keypairs.Sort()
	assert.Equal(t, bytes.Compare(keypairs[0].BLSKey, keypairs[1].BLSKey), 1)

	assert.Len(t, keypairs.BLSKeys(), 3)
}

package blockchain

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/sync/errgroup"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

func TestBlockValidation(t *testing.T) {
	fixtureFile, err := os.ReadFile("./fixtures_test/block.json")
	assert.NoError(t, err)
	fixture := struct {
		ValidBlock    *Block `json:"validBlock"`
		InvalidBlocks []struct {
			Block *Block `json:"block"`
			Err   string `json:"err"`
		} `json:"invalidBlocks"`
	}{}

	err = json.Unmarshal(fixtureFile, &fixture)
	assert.NoError(t, err)
	for _, testCase := range fixture.InvalidBlocks {
		err = testCase.Block.Validate()
		assert.EqualError(t, err, testCase.Err)
	}

	err = fixture.ValidBlock.Validate()
	assert.NoError(t, err)
}

func TestValidateGenesis(t *testing.T) {
	fixtureFile, err := os.ReadFile("./fixtures_test/genesis_block.json")
	assert.NoError(t, err)
	fixture := struct {
		ValidBlock    *Block `json:"validBlock"`
		InvalidBlocks []struct {
			Block *Block `json:"block"`
			Err   string `json:"err"`
		} `json:"invalidBlocks"`
	}{}

	err = json.Unmarshal(fixtureFile, &fixture)
	assert.NoError(t, err)
	for _, testCase := range fixture.InvalidBlocks {
		err = testCase.Block.ValidateGenesis()
		assert.EqualError(t, err, testCase.Err)
	}

	err = fixture.ValidBlock.ValidateGenesis()
	assert.NoError(t, err)
}

func TestHeaderCreate(t *testing.T) {
	header, err := NewBlockHeaderWithValues(
		2,
		100,
		1,
		crypto.RandomBytes(32),
		crypto.RandomBytes(32),
		crypto.RandomBytes(32),
		11,
		10,
		crypto.RandomBytes(32),
		crypto.RandomBytes(20),
		crypto.RandomBytes(20),
		&AggregateCommit{
			Height:               0,
			AggregationBits:      []byte{},
			CertificateSignature: []byte{},
		},
		crypto.RandomBytes(64),
	)
	assert.NoError(t, err)
	assert.NotEqual(t, len(header.ID), 0)
}

func TestBlockSignature(t *testing.T) {
	entropy, _ := bip39.NewEntropy(256)
	passphrase, _ := bip39.NewMnemonic(entropy)
	pk, sk, _ := crypto.GetKeys(passphrase)
	address := crypto.GetAddress(pk)
	header := &BlockHeader{
		Version:            2,
		Timestamp:          100,
		Height:             1,
		PreviousBlockID:    crypto.RandomBytes(32),
		GeneratorAddress:   address,
		TransactionRoot:    emptyHash,
		StateRoot:          emptyHash,
		AssetRoot:          emptyHash,
		MaxHeightPrevoted:  0,
		MaxHeightGenerated: 0,
		ValidatorsHash:     emptyHash,
		AggregateCommit: &AggregateCommit{
			Height:               0,
			AggregationBits:      []byte{},
			CertificateSignature: []byte{},
		},
	}
	signingBytes := header.SigningBytes()

	networkID := crypto.RandomBytes(32)
	expectedSignature := crypto.Sign(sk, crypto.Hash(bytes.Join(
		TagBlockHeader,
		networkID,
		signingBytes,
	)))

	header.Sign(networkID, sk)

	assert.Equal(t, expectedSignature, []byte(header.Signature))

	valid := header.VerifySignature(networkID, pk)
	assert.True(t, valid)

	header.Version = 0
	valid = header.VerifySignature(networkID, pk)
	assert.False(t, valid)
}

func TestBlockAssets(t *testing.T) {
	assets := BlockAssets{
		{
			Module: "transfer",
			Data:   crypto.RandomBytes(100),
		},
		{
			Module: "dpos",
			Data:   crypto.RandomBytes(100),
		},
	}
	assets.Sort()
	assert.Equal(t, "dpos", assets[0].Module)

	data, exist := assets.GetAsset("dpos")
	assert.True(t, exist)
	assert.Equal(t, assets[0].Data, codec.Hex(data))

	_, exist = assets.GetAsset("random")
	assert.False(t, exist)

	updatingData := crypto.RandomBytes(20)
	assets.SetAsset("dpos", updatingData)
	data, exist = assets.GetAsset("dpos")
	assert.True(t, exist)
	assert.Equal(t, updatingData, data)

	assets.SetAsset("token", updatingData)
	data, exist = assets.GetAsset("token")
	assert.True(t, exist)
	assert.Equal(t, updatingData, data)
}

func TestBlockCodec(t *testing.T) {
	// Test should not panic
	block := createRandomBlock(100)

	// Raw Block
	encoded := block.Encode()
	rawBlock := &RawBlock{}
	err := rawBlock.Decode(encoded)

	assert.NoError(t, err)
	rawBlock.MustDecode(encoded)

	reEncoded := rawBlock.Encode()
	assert.Equal(t, encoded, reEncoded)

	// Block
	newBlock := &Block{}
	newBlock.MustDecode(reEncoded)

	// Signing block
	signingBlock := &signingBlockHeader{}
	signingBlock.MustDecode(rawBlock.Header)
	encodedSigning := signingBlock.Encode()
	// Signature should not be part of the encoding
	assert.NotEqual(t, rawBlock.Header, encodedSigning)

	// Header
	header := &BlockHeader{}
	header.MustDecode(rawBlock.Header)
	encodedHeader := header.Encode()
	assert.Equal(t, rawBlock.Header, encodedHeader)

	encodedCommit := header.AggregateCommit.Encode()
	aggCommit := &AggregateCommit{}
	aggCommit.MustDecode(encodedCommit)
}

func TestBlockCodecFuzz(t *testing.T) {
	fuzzCount := 10000
	eg := new(errgroup.Group)
	for i := 0; i < fuzzCount; i++ {
		eg.Go(func() error {
			randomBytes := crypto.RandomBytes(500)
			block := &Block{}
			err := block.Decode(randomBytes)
			if err == nil {
				block.Encode()
			}
			return nil
		})
	}
	err := eg.Wait()
	assert.NoError(t, err)
}

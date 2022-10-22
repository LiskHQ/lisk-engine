// Package certificate implements block certificate generation protocol following [LIP-0061].
//
// [LIP-0061]: https://github.com/LiskHQ/lips/blob/main/proposals/lip-0061.md
package certificate

import (
	"fmt"
	"sort"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

const (
	CommitRangeStored = uint32(100)
)

var certificateTag = []byte("LSK_CE_")

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

type Certificate struct {
	BlockID         codec.Hex `fieldNumber:"1"`
	Height          uint32    `fieldNumber:"2"`
	Timestamp       uint32    `fieldNumber:"3"`
	StateRoot       codec.Hex `fieldNumber:"4"`
	ValidatorsHash  codec.Hex `fieldNumber:"5"`
	AggregationBits codec.Hex `fieldNumber:"6"`
	Signature       codec.Hex `fieldNumber:"7"`
}

type SigningCertificate struct {
	BlockID        codec.Hex `fieldNumber:"1"`
	Height         uint32    `fieldNumber:"2"`
	Timestamp      uint32    `fieldNumber:"3"`
	StateRoot      codec.Hex `fieldNumber:"4"`
	ValidatorsHash codec.Hex `fieldNumber:"5"`
}

func NewCertificateFromBlock(h *blockchain.BlockHeader) *Certificate {
	return &Certificate{
		BlockID:        bytes.Copy(h.ID),
		Height:         h.Height,
		Timestamp:      h.Timestamp,
		StateRoot:      bytes.Copy(h.StateRoot),
		ValidatorsHash: bytes.Copy(h.ValidatorsHash),
	}
}

func (c Certificate) Empty() bool {
	return len(c.BlockID) == 0 &&
		c.Height == 0 &&
		c.Timestamp == 0 &&
		len(c.StateRoot) == 0 &&
		len(c.ValidatorsHash) == 0 &&
		len(c.AggregationBits) == 0 &&
		len(c.Signature) == 0
}

func (c *Certificate) SigningBytes() ([]byte, error) {
	certificate := &SigningCertificate{
		BlockID:        c.BlockID,
		Height:         c.Height,
		Timestamp:      c.Timestamp,
		StateRoot:      c.StateRoot,
		ValidatorsHash: c.ValidatorsHash,
	}
	return certificate.Encode()
}

func (c *Certificate) Sign(chainID, blsSecretKey []byte) error {
	signingBytes, err := c.SigningBytes()
	if err != nil {
		return err
	}
	signature := crypto.BLSSign(crypto.Hash(bytes.Join(certificateTag, chainID, signingBytes)), blsSecretKey)
	c.Signature = signature
	return nil
}

func (c Certificate) Verify(chainID, signature, blsPublicKey []byte) (bool, error) {
	signingBytes, err := c.SigningBytes()
	if err != nil {
		return false, err
	}
	valid := crypto.BLSVerify(crypto.Hash(bytes.Join(certificateTag, chainID, signingBytes)), signature, blsPublicKey)
	return valid, nil
}

func (c Certificate) VerifyAggregateCertificateSignature(keyList [][]byte, weights []uint64, threshold uint64, chainID []byte) (bool, error) {
	aggregateSignature := bytes.Copy(c.Signature)
	aggregationBits := bytes.Copy(c.AggregationBits)
	signingBytes, err := c.SigningBytes()
	if err != nil {
		return false, err
	}
	valid := crypto.BLSVerifyWeightedAggSig(keyList, aggregationBits, aggregateSignature, weights, threshold, crypto.Hash(bytes.Join(certificateTag, chainID, signingBytes)))
	return valid, nil
}

func NewSingleCommit(header *blockchain.BlockHeader, address codec.Lisk32, chainID, sk []byte) (*SingleCommit, error) {
	singleCommit := &SingleCommit{
		blockID:          header.ID,
		height:           header.Height,
		validatorAddress: address,
		internal:         true,
	}
	certificate := NewCertificateFromBlock(header)
	if err := certificate.Sign(chainID, sk); err != nil {
		return nil, err
	}
	singleCommit.certificateSignature = certificate.Signature
	return singleCommit, nil
}

type SingleCommit struct {
	blockID              codec.Hex    `fieldNumber:"1"`
	height               uint32       `fieldNumber:"2"`
	validatorAddress     codec.Lisk32 `fieldNumber:"3"`
	certificateSignature codec.Hex    `fieldNumber:"4"`
	internal             bool
}

func (c *SingleCommit) Height() uint32                  { return c.height }
func (c *SingleCommit) BlockID() codec.Hex              { return c.blockID }
func (c *SingleCommit) CertificateSignature() codec.Hex { return c.certificateSignature }
func (c *SingleCommit) ValidatorAddress() codec.Lisk32  { return c.validatorAddress }

type SingleCommits []*SingleCommit

func (s SingleCommits) has(val *SingleCommit) bool {
	for _, commit := range s {
		if bytes.Equal(commit.blockID, val.blockID) && bytes.Equal(commit.validatorAddress, val.validatorAddress) {
			return true
		}
	}
	return false
}

// GetUntil returns SingleCommits below the height. It expects to be sorted by asc.
func (s SingleCommits) GetUntil(height uint32) SingleCommits {
	result := SingleCommits{}
	for _, commit := range s {
		if commit.height >= height {
			return result
		}
		result = append(result, commit)
	}
	return result
}

// GetLargestWithLimit returns SingleCommits with limit from largest number.
func (s SingleCommits) GetLargestWithLimit(limit int, internal bool) SingleCommits {
	result := SingleCommits{}
	for i := len(s) - 1; i >= 0; i-- {
		if s[i].internal == internal {
			result = append(result, s[i])
		}
		if len(result) >= limit {
			return result
		}
	}
	return result
}

// Sort asc.
func (s *SingleCommits) Sort() {
	original := *s
	sort.Slice(original, func(i, j int) bool {
		return original[i].height < original[j].height
	})
	*s = original
}

// Aggregate single commits where all single commits is for a block
// bls keys are BLS keys of active validators at the height ordered lexiographically.
func (s SingleCommits) Aggregate(keypairs AddressKeyPairs) (*blockchain.AggregateCommit, error) {
	if len(s) == 0 {
		return nil, fmt.Errorf("single commit is empty")
	}
	height := s[0].height
	keypairs.Sort()
	psPair := make([]*crypto.BLSPublicKeySignaturePair, len(s))
	for i, commit := range s {
		blsKey, exist := keypairs.BLSKey(commit.validatorAddress)
		if !exist {
			return nil, fmt.Errorf("bls key for address %s does not exist in the given keypairs", commit.validatorAddress)
		}
		psPair[i] = &crypto.BLSPublicKeySignaturePair{
			PublicKey: blsKey,
			Signature: commit.certificateSignature,
		}
	}
	aggregateBits, signature := crypto.BLSCreateAggSig(keypairs.BLSKeys(), psPair)
	ac := &blockchain.AggregateCommit{
		Height:               height,
		AggregationBits:      aggregateBits,
		CertificateSignature: signature,
	}
	return ac, nil
}

type AddressKeyPair struct {
	Address codec.Lisk32
	BLSKey  codec.Hex
}

type AddressKeyPairs []*AddressKeyPair

func (kps AddressKeyPairs) BLSKey(address []byte) ([]byte, bool) {
	for _, kp := range kps {
		if bytes.Equal(address, kp.Address) {
			return kp.BLSKey, true
		}
	}
	return nil, false
}

func (kps *AddressKeyPairs) Sort() {
	original := *kps
	sort.Slice(original, func(i, j int) bool {
		return bytes.Compare(original[i].BLSKey, original[j].BLSKey) > 0
	})
	*kps = original
}

func (kps AddressKeyPairs) BLSKeys() [][]byte {
	keys := make([][]byte, len(kps))
	for i, kp := range kps {
		keys[i] = kp.BLSKey
	}
	return keys
}

func GetMaxRemovalHeight(h *blockchain.BlockHeader) uint32 {
	return h.AggregateCommit.Height
}

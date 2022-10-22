package validator

import (
	"bytes"
	"sort"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

type HashValidator interface {
	BLSKey() codec.Hex
	BFTWeight() uint64
}

type hashValidator struct {
	blsKey    []byte `fieldNumber:"1"`
	bftWeight uint64 `fieldNumber:"2"`
}
type HashValidators []HashValidator

func (v *hashValidator) BLSKey() codec.Hex { return v.blsKey }
func (v *hashValidator) BFTWeight() uint64 { return v.bftWeight }

func NewHashValidator(blsKey []byte, bftWeight uint64) *hashValidator {
	return &hashValidator{
		blsKey:    blsKey,
		bftWeight: bftWeight,
	}
}

type validatrorsHashData struct {
	activeValidators     []*hashValidator `fieldNumber:"1"`
	certificateThreshold uint64           `fieldNumber:"2"`
}

func ComputeValidatorsHash(validators []HashValidator, certificateThreshold uint64) ([]byte, error) {
	hashingValidators := make([]*hashValidator, len(validators))
	for i, val := range validators {
		hashingValidators[i] = &hashValidator{
			blsKey:    val.BLSKey(),
			bftWeight: val.BFTWeight(),
		}
	}

	sort.Slice(hashingValidators, func(i, j int) bool {
		return bytes.Compare(hashingValidators[i].blsKey, hashingValidators[j].blsKey) < 0
	})

	validatorsHash := &validatrorsHashData{
		activeValidators:     hashingValidators,
		certificateThreshold: certificateThreshold,
	}

	encodedInfo, err := validatorsHash.Encode()
	if err != nil {
		return nil, err
	}
	return crypto.Hash(encodedInfo), nil
}

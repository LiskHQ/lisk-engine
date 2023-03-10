package mock

import (
	"fmt"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

type UserAccount struct {
	Nonce uint64 `fieldNumber:"1" json:"nonce,string"`
}

type ValidatorKey struct {
	Address       codec.Lisk32 `fieldNumber:"1" json:"address"`
	GenerationKey codec.Hex    `fieldNumber:"2" json:"generatorKey"`
	BLSKey        codec.Hex    `fieldNumber:"3" json:"blsKey"`
	BFTWeight     uint64       `fieldNumber:"4" json:"bftWeight,string"`
}

func (v *ValidatorKey) validate() error {
	if len(v.Address) != blockchain.AddressLength {
		return fmt.Errorf("invalid address length")
	}
	if len(v.GenerationKey) != crypto.EdPublicKeyLength {
		return fmt.Errorf("invalid generatorKey length")
	}
	if len(v.BLSKey) != crypto.BLSPublicKeyLength {
		return fmt.Errorf("invalid BLSKey length")
	}
	return nil
}

type ValidatorsData struct {
	Keys []*ValidatorKey `fieldNumber:"1" json:"keys"`
}

func (v *ValidatorsData) validate() error {
	if len(v.Keys) == 0 {
		return fmt.Errorf("params must contain at least one key")
	}
	for _, key := range v.Keys {
		if err := key.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (v *ValidatorsData) toLABIValidators() []*labi.Validator {
	validators := make([]*labi.Validator, len(v.Keys))
	for i, acct := range v.Keys {
		validators[i] = &labi.Validator{
			Address:      acct.Address,
			GeneratorKey: acct.GenerationKey,
			BLSKey:       acct.BLSKey,
			BFTWeight:    acct.BFTWeight,
		}
	}
	return validators
}

func (v *ValidatorsData) totalWeight() uint64 {
	sum := uint64(0)
	for _, acct := range v.Keys {
		sum += acct.BFTWeight
	}
	return sum
}

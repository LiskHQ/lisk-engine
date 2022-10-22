package auth

import (
	"fmt"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

const storePrefixUserAccount uint16 = 0

type UserAccount struct {
	Nonce              uint64   `fieldNumber:"1" json:"nonce,string"`
	NumberOfSignatures uint32   `fieldNumber:"2" json:"numberOfSignatures"`
	MandatoryKeys      [][]byte `fieldNumber:"3" json:"mandatoryKeys"`
	OptionalKeys       [][]byte `fieldNumber:"4" json:"optionalKeys"`
}

type GetAccountRequest struct {
	Address codec.Lisk32 `json:"address" fieldNumber:"1"`
}

type GetAccountResponse struct {
	Nonce              uint64      `fieldNumber:"1" json:"nonce,string"`
	NumberOfSignatures uint32      `fieldNumber:"2" json:"numberOfSignatures"`
	MandatoryKeys      []codec.Hex `fieldNumber:"3" json:"mandatoryKeys"`
	OptionalKeys       []codec.Hex `fieldNumber:"4" json:"optionalKeys"`
}

func (k *UserAccount) IsMultisignatureAccount() bool {
	return k.NumberOfSignatures != 0
}

type RegisterMultisignatureParams struct {
	NumberOfSignatures uint32   `fieldNumber:"1"`
	MandatoryKeys      [][]byte `fieldNumber:"2"`
	OptionalKeys       [][]byte `fieldNumber:"3"`
}

func (k *RegisterMultisignatureParams) Validate() error {
	if k.NumberOfSignatures == 0 || k.NumberOfSignatures > 64 {
		return fmt.Errorf("number of signatures is out of range %d", k.NumberOfSignatures)
	}
	if len(k.MandatoryKeys) > 64 || len(k.MandatoryKeys) > int(k.NumberOfSignatures) {
		return fmt.Errorf("length of mandatory keys is out of range %d", len(k.MandatoryKeys))
	}
	if len(k.OptionalKeys) > 64 {
		return fmt.Errorf("length of optional keys is out of range %d", len(k.OptionalKeys))
	}

	if len(bytes.Unique(k.MandatoryKeys)) != len(k.MandatoryKeys) {
		return fmt.Errorf("mandatory keys are not unique")
	}
	for _, key := range k.MandatoryKeys {
		if len(key) != 32 {
			return fmt.Errorf("invalid key length %d in mandatory keys", len(key))
		}
	}
	if !bytes.IsSorted(k.MandatoryKeys) {
		return fmt.Errorf("mandatory keys are not sorted")
	}

	if len(bytes.Unique(k.OptionalKeys)) != len(k.OptionalKeys) {
		return fmt.Errorf("mandatory keys are not unique")
	}
	for _, key := range k.OptionalKeys {
		if len(key) != 32 {
			return fmt.Errorf("invalid key length %d in optional keys", len(key))
		}
	}
	if !bytes.IsSorted(k.OptionalKeys) {
		return fmt.Errorf("optional keys are not sorted")
	}

	combined := bytes.JoinSlice(k.MandatoryKeys, k.OptionalKeys)
	totalSize := len(combined)
	if totalSize > 64 || totalSize == 0 || totalSize < int(k.NumberOfSignatures) {
		return fmt.Errorf("length of mandatory and optional keys is out of range %d", totalSize)
	}
	if len(bytes.Unique(combined)) != len(combined) {
		return fmt.Errorf("combined keys are not unique")
	}
	return nil
}

type GenesisAssetAccount struct {
	Nonce              uint64      `fieldNumber:"1"`
	NumberOfSignatures uint32      `fieldNumber:"2"`
	MandatoryKeys      []codec.Hex `fieldNumber:"3"`
	OptionalKeys       []codec.Hex `fieldNumber:"4"`
}

type AuthData struct {
	Address     codec.Lisk32         `fieldNumber:"1"`
	AuthAccount *GenesisAssetAccount `fieldNumber:"2"`
}

type GenesisAsset struct {
	AuthDataSubstore []*AuthData `fieldNumber:"1"`
}

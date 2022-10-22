package validators

import (
	"errors"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

// GeneratorList holds set of validators.
type GeneratorList struct {
	Addresses []codec.Lisk32 `fieldNumber:"1"`
}

type GenesisState struct {
	GenesisTimestamp uint32 `fieldNumber:"1"`
}

type ValidatorAccount struct {
	GenerationKey codec.Hex `fieldNumber:"1"`
	BLSKey        codec.Hex `fieldNumber:"2"`
}

type ValidatorAddress struct {
	Address codec.Lisk32 `fieldNumber:"1"`
}

// Endpoints

type GetValidatorRequest struct {
	Timestamp uint32 `json:"timestamp" fieldNumber:"1"`
}

type GetValidatorResponse struct {
	Address codec.Lisk32 `json:"address" fieldNumber:"1"`
}

// Commands

type UpdateGenerationKeyParams struct {
	GenerationKey codec.Hex `fieldNumber:"1"`
}

func (p *UpdateGenerationKeyParams) Validate() error {
	if len(p.GenerationKey) != 32 {
		return errors.New("generation key must have length 32")
	}
	return nil
}

// GenesisAssets

type GenesisAssetAccount struct {
	Address       codec.Lisk32 `fieldNumber:"1"`
	BLSKey        codec.Hex    `fieldNumber:"2"`
	GenerationKey codec.Hex    `fieldNumber:"3"`
}

type GenesisAsset struct {
	Accounts []*GenesisAssetAccount `fieldNumber:"1"`
}

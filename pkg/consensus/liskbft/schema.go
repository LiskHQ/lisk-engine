package liskbft

import "github.com/LiskHQ/lisk-engine/pkg/codec"

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

type GetValidatorInfoResponse struct {
	MaxHeightPrevoted uint32 `json:"maxHeightPrevoted" fieldNumber:"1"`
}

type IsBFTComplientRequest struct {
	GeneratorPublicKey codec.Hex `json:"generatorPublicKey" fieldNumber:"1"`
	Height             uint32    `json:"height" fieldNumber:"2"`
	MaxHeightGenerated uint32    `json:"maxHeightPreviouslyForged" fieldNumber:"3"`
	MaxHeightPrevoted  uint32    `json:"maxHeightPrevoted" fieldNumber:"4"`
}

type IsBFTComplientResponse struct {
	Valid bool `json:"valid" fieldNumber:"1"`
}

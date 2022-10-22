// Package codec_test is a test package for codec.
package codec_test

import "github.com/LiskHQ/lisk-engine/pkg/codec"

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

type Transaction struct {
	ID              codec.Hex   `json:"id"`
	Module          string      `json:"module" fieldNumber:"1"`
	Command         string      `json:"commandID" fieldNumber:"2"`
	Nonce           uint64      `json:"nonce,string" fieldNumber:"3"`
	Fee             uint64      `json:"fee,string" fieldNumber:"4"`
	SenderPublicKey codec.Hex   `json:"senderPublicKey" fieldNumber:"5"`
	Params          codec.Hex   `json:"params" fieldNumber:"6"`
	Signatures      []codec.Hex `json:"signatures" fieldNumber:"7"`
}

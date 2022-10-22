/*
gen generates encoding/decoding codes for struct using "fieldNumber" tag.
Below example will generate corresponding encoding/decoding functions.

Example:

	//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

	type Transaction struct {
		ID              codec.Hex
		size            int
		Module          string      `fieldNumber:"1"`
		Command         string      `fieldNumber:"2"`
		Nonce           uint64      `fieldNumber:"3"`
		Fee             uint64      `fieldNumber:"4"`
		SenderPublicKey codec.Hex   `fieldNumber:"5"`
		Params          codec.Hex   `fieldNumber:"6"`
		Signatures      []codec.Hex `fieldNumber:"7"`
	}
*/
package main

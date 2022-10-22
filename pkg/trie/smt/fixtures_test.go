package smt

import (
	"encoding/json"
	"os"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

type fixtureTestCase struct {
	Description string         `json:"description" fieldNumber:"1"`
	Input       *fixtureInput  `json:"input" fieldNumber:"2"`
	Output      *fixtureOutput `json:"output" fieldNumber:"3"`
}

type fixtureInput struct {
	Keys       []codec.Hex `json:"keys" fieldNumber:"1"`
	Values     []codec.Hex `json:"values" fieldNumber:"2"`
	DeleteKeys []codec.Hex `json:"deleteKeys" fieldNumber:"3"`
	QueryKeys  []codec.Hex `json:"queryKeys" fieldNumber:"4"`
}

type fixtureOutput struct {
	MerkleRoot codec.Hex           `json:"merkleRoot" fieldNumber:"1"`
	Proof      *fixtureOutputProof `json:"proof" fieldNumber:"2"`
}

type fixtureOutputProofQuery struct {
	Bitmap codec.Hex `json:"bitmap" fieldNumber:"1"`
	Key    codec.Hex `json:"key" fieldNumber:"2"`
	Value  codec.Hex `json:"value" fieldNumber:"3"`
}

type fixtureOutputProof struct {
	SiblingHashes []codec.Hex                `json:"siblingHashes" fieldNumber:"1"`
	Queries       []*fixtureOutputProofQuery `json:"queries" fieldNumber:"2"`
}

type fixture struct {
	Title     string             `json:"title" fieldNumber:"1"`
	Summary   string             `json:"summary" fieldNumber:"2"`
	TestCases []*fixtureTestCase `json:"testCases" fieldNumber:"3"`
}

func loadFixture(path string, fixture interface{}) error {
	file, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(file, fixture)
}

func loadEncodedFixture(path string, fixture codec.Decodable) error {
	file, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return fixture.Decode(file)
}

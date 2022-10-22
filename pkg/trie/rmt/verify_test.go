package rmt

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

func stringsToInts(arr []string) []uint64 {
	res := make([]uint64, len(arr))
	for i, val := range arr {
		num, err := strconv.ParseInt(val, 16, 64)
		if err != nil {
			panic(err)
		}
		res[i] = uint64(num)
	}
	return res
}

func TestVerifyProof(t *testing.T) {
	fixture := &ProofFixture{}
	if err := loadFixture("./fixtures/generate_verify_proof_fixtures.json", fixture); err != nil {
		t.Fatal(err)
	}

	for _, testCase := range fixture.TestCases {
		t.Log(testCase.Description)
		valid := VerifyProof(
			codec.HexArrayToBytesArray(testCase.Input.QueryHashes),
			&Proof{
				Size:          uint64(testCase.Output.Proof.Size),
				Idxs:          stringsToInts(testCase.Output.Proof.Idxs),
				SiblingHashes: codec.HexArrayToBytesArray(testCase.Output.Proof.SiblingHashes),
			},
			testCase.Output.MerkleRoot,
		)
		assert.True(t, valid)
	}
}

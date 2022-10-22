package rmt

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

func TestRootCalculation(t *testing.T) {
	fixture := &TransactionRootFixture{}
	if err := loadFixture("./fixtures/transaction_root_fixtures.json", fixture); err != nil {
		t.Fatal(err)
	}

	for _, tc := range fixture.TestCases {
		result := CalculateRoot(codec.HexArrayToBytesArray(tc.Input.TransactionIDs))
		assert.Equal(t, []byte(tc.Output.TransactionMerkleRoot), result)
	}
}

func TestCalculateRootFromUpdateData(t *testing.T) {
	fixture := &UpdateLeavesFixture{}
	if err := loadFixture("./fixtures/update_leaves_fixtures.json", fixture); err != nil {
		t.Fatal(err)
	}

	for _, testCase := range fixture.TestCases {
		proof := &Proof{
			Size:          uint64(testCase.Input.Proof.Size),
			Idxs:          stringsToInts(testCase.Input.Proof.Idxs),
			SiblingHashes: codec.HexArrayToBytesArray(testCase.Input.Proof.SiblingHashes),
		}
		result, err := CalculateRootFromUpdateData(
			codec.HexArrayToBytesArray(testCase.Input.UpdateValues),
			proof,
		)
		assert.NoError(t, err)
		assert.Equal(t, []byte(testCase.Output.FinalMerkleRoot), result)
	}
}

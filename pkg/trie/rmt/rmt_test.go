package rmt

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/db"
)

func TestRMTAppend(t *testing.T) {
	fixture := &TransactionRootFixture{}
	if err := loadFixture("./fixtures/transaction_root_fixtures.json", fixture); err != nil {
		t.Fatal(err)
	}

	for _, testCase := range fixture.TestCases {
		t.Log(testCase.Description)
		store, _ := db.NewInMemoryDB()
		rmt := NewRegularMerkleTree(store)
		for _, input := range testCase.Input.TransactionIDs {
			err := rmt.Append(input)
			assert.NoError(t, err)
		}

		assert.Equal(t, []byte(testCase.Output.TransactionMerkleRoot), rmt.Root())
	}
}

func TestGenerateProof(t *testing.T) {
	fixture := &ProofFixture{}
	if err := loadFixture("./fixtures/generate_verify_proof_fixtures.json", fixture); err != nil {
		t.Fatal(err)
	}
	for _, testCase := range fixture.TestCases {
		t.Log(testCase.Description)
		store, _ := db.NewInMemoryDB()
		rmt := NewRegularMerkleTree(store)
		for _, input := range testCase.Input.Values {
			err := rmt.Append(input)
			assert.NoError(t, err)
		}
		proof, err := rmt.GenerateProof(codec.HexArrayToBytesArray(testCase.Input.QueryHashes))
		assert.NoError(t, err)

		assert.Equal(t, uint64(testCase.Output.Proof.Size), proof.Size)
		assert.Equal(t, codec.HexArrayToBytesArray(testCase.Output.Proof.SiblingHashes), proof.SiblingHashes)
		assert.Equal(t, stringsToInts(testCase.Output.Proof.Idxs), proof.Idxs)
	}
}

func TestRightWitness(t *testing.T) {
	fixture := &TransactionRootFixture{}
	if err := loadFixture("./fixtures/transaction_root_fixtures.json", fixture); err != nil {
		t.Fatal(err)
	}

	for _, testCase := range fixture.TestCases {
		t.Log(testCase.Description)
		database, _ := db.NewInMemoryDB()
		fullTree := NewRegularMerkleTree(database)
		for _, input := range testCase.Input.TransactionIDs {
			err := fullTree.Append(input)
			assert.NoError(t, err)
		}
		eg := new(errgroup.Group)
		for i := 0; i < len(testCase.Input.TransactionIDs); i++ {
			i := i
			eg.Go(func() error {
				t.Logf("Testing index %d \n", i)
				witness, err := fullTree.GenerateRightWitness(uint64(i))
				assert.NoError(t, err)
				internalDB, _ := db.NewInMemoryDB()
				partialTree := NewRegularMerkleTree(internalDB)
				for j := 0; j < i; j++ {
					if err := partialTree.Append(testCase.Input.TransactionIDs[j]); err != nil {
						return err
					}
				}
				if !VerifyRightWitness(uint64(i), partialTree.AppendPath(), witness, fullTree.Root()) {
					return fmt.Errorf("invalid right wirness at i = %d", i)
				}
				return nil
			})
		}
		err := eg.Wait()
		assert.NoError(t, err)
	}
}

func TestUpdateData(t *testing.T) {
	fixture := &UpdateLeavesFixture{}
	if err := loadFixture("./fixtures/update_leaves_fixtures.json", fixture); err != nil {
		t.Fatal(err)
	}

	for _, testCase := range fixture.TestCases {
		store, _ := db.NewInMemoryDB()
		rmt := NewRegularMerkleTree(store)
		for _, input := range testCase.Input.Values {
			err := rmt.Append(input)
			assert.NoError(t, err)
		}
		err := rmt.Update(stringsToInts(testCase.Input.Proof.Idxs), codec.HexArrayToBytesArray(testCase.Input.UpdateValues))
		assert.NoError(t, err)
		assert.Equal(t, []byte(testCase.Output.FinalMerkleRoot), rmt.Root())
	}
}

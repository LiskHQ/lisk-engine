package smt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	"github.com/LiskHQ/lisk-engine/pkg/db"
)

func TestUpdateSMTFixture(t *testing.T) {
	updateFixture := &fixture{}
	err := loadFixture("./fixtures/smt_fixtures.json", updateFixture)
	assert.NoError(t, err)

	for _, testCase := range updateFixture.TestCases {
		t.Log(testCase.Description)
		database, _ := db.NewInMemoryDB()
		smt := NewTrie([]byte{}, 32)
		keys := make([][]byte, len(testCase.Input.Keys))
		values := make([][]byte, len(testCase.Input.Values))
		for i, key := range testCase.Input.Keys {
			keys[i] = key
			values[i] = testCase.Input.Values[i]
		}
		root, err := smt.Update(database, keys, values)
		assert.NoError(t, err)
		assert.Equal(t, []byte(testCase.Output.MerkleRoot), root)
	}
}

func TestUpdateTreeFixture(t *testing.T) {
	updateFixture := &fixture{}
	err := loadFixture("./fixtures/update_tree.json", updateFixture)
	assert.NoError(t, err)

	for _, testCase := range updateFixture.TestCases {
		t.Log(testCase.Description)
		database, _ := db.NewInMemoryDB()
		smt := NewTrie([]byte{}, 32)
		keys := make([][]byte, len(testCase.Input.Keys))
		values := make([][]byte, len(testCase.Input.Values))
		for i, key := range testCase.Input.Keys {
			keys[i] = key
			values[i] = testCase.Input.Values[i]
		}
		root, err := smt.Update(database, keys, values)
		assert.NoError(t, err)
		assert.Equal(t, []byte(testCase.Output.MerkleRoot), root)
	}
}

func TestRemoveTreeFixture(t *testing.T) {
	updateFixture := &fixture{}
	err := loadFixture("./fixtures/remove_tree.json", updateFixture)
	assert.NoError(t, err)

	for _, testCase := range updateFixture.TestCases {
		t.Log(testCase.Description)
		database, _ := db.NewInMemoryDB()
		smt := NewTrie([]byte{}, 32)
		keys := make([][]byte, len(testCase.Input.Keys)+len(testCase.Input.DeleteKeys))
		values := make([][]byte, len(testCase.Input.Values)+len(testCase.Input.DeleteKeys))
		for i, key := range testCase.Input.Keys {
			keys[i] = key
			values[i] = testCase.Input.Values[i]
		}
		for i, deleteKey := range testCase.Input.DeleteKeys {
			keys[i+len(testCase.Input.Keys)] = deleteKey
			values[i+len(testCase.Input.Keys)] = []byte{}
		}
		keys, values = UniqueAndSort(keys, values)
		root, err := smt.Update(database, keys, values)
		assert.NoError(t, err)
		assert.Equal(t, []byte(testCase.Output.MerkleRoot), root)
	}
}

func TestRemoveExtraTreeFixture(t *testing.T) {
	updateFixture := &fixture{}
	err := loadFixture("./fixtures/remove_extra_tree.json", updateFixture)
	assert.NoError(t, err)

	for _, testCase := range updateFixture.TestCases {
		t.Log(testCase.Description)
		database, _ := db.NewInMemoryDB()
		smt := NewTrie([]byte{}, 32)
		keys := make([][]byte, len(testCase.Input.Keys)+len(testCase.Input.DeleteKeys))
		values := make([][]byte, len(testCase.Input.Values)+len(testCase.Input.DeleteKeys))
		for i, key := range testCase.Input.Keys {
			keys[i] = key
			values[i] = testCase.Input.Values[i]
		}
		for i, deleteKey := range testCase.Input.DeleteKeys {
			keys[i+len(testCase.Input.Keys)] = deleteKey
			values[i+len(testCase.Input.Keys)] = []byte{}
		}
		keys, values = UniqueAndSort(keys, values)
		root, err := smt.Update(database, keys, values)
		assert.NoError(t, err)
		assert.Equal(t, []byte(testCase.Output.MerkleRoot), root)
	}
}

func TestGenerateProofFixture(t *testing.T) {
	updateFixture := &fixture{}
	err := loadFixture("./fixtures/smt_proof_fixtures.json", updateFixture)
	assert.NoError(t, err)

	for _, testCase := range updateFixture.TestCases {
		t.Log(testCase.Description)
		database, _ := db.NewInMemoryDB()
		smt := NewTrie([]byte{}, 32)
		keys := make([][]byte, len(testCase.Input.Keys)+len(testCase.Input.DeleteKeys))
		values := make([][]byte, len(testCase.Input.Values)+len(testCase.Input.DeleteKeys))
		for i, key := range testCase.Input.Keys {
			keys[i] = key
			values[i] = testCase.Input.Values[i]
		}
		for i, deleteKey := range testCase.Input.DeleteKeys {
			keys[i+len(testCase.Input.Keys)] = deleteKey
			values[i+len(testCase.Input.Keys)] = []byte{}
		}
		keys, values = UniqueAndSort(keys, values)

		root, err := smt.Update(database, keys, values)

		assert.NoError(t, err)
		assert.Equal(t, []byte(testCase.Output.MerkleRoot), root)

		queryKeys := make([][]byte, len(testCase.Input.QueryKeys))
		for i, k := range testCase.Input.QueryKeys {
			queryKeys[i] = k
		}

		proof, err := smt.Prove(database, queryKeys)
		assert.NoError(t, err)

		assert.Len(t, proof.Queries, len(testCase.Output.Proof.Queries))
		for i, q := range proof.Queries {
			assert.Equal(t, testCase.Output.Proof.Queries[i].Bitmap, q.Bitmap, "bitmap does not match")
			assert.Equal(t, testCase.Output.Proof.Queries[i].Key, q.Key, "key does not match")
			assert.Equal(t, testCase.Output.Proof.Queries[i].Value, q.Value, "value does not match")
		}
		assert.Equal(t, testCase.Output.Proof.SiblingHashes, proof.SiblingHashes, "sibling hash does not match")
		verified, err := Verify(queryKeys, proof, root, 32)
		assert.NoError(t, err)
		assert.True(t, verified)

		// Modified bitmap should fail (prepend 0 and append 0)
		zeroPrependedProof := proof.clone()
		for i, q := range zeroPrependedProof.Queries {
			q.Bitmap = append([]byte{0}, q.Bitmap...)
			zeroPrependedProof.Queries[i] = q
		}
		verified, err = Verify(queryKeys, zeroPrependedProof, root, 32)
		assert.NoError(t, err)
		assert.False(t, verified)

		zeroAppendedProof := proof.clone()
		for i, q := range zeroAppendedProof.Queries {
			q.Bitmap = append(q.Bitmap, 0)
			zeroAppendedProof.Queries[i] = q
		}
		verified, _ = Verify(queryKeys, zeroAppendedProof, root, 32)
		// Assert for NoError check can't be done here because the error is not nil in some cases
		// but the verification is still correct and must be false
		assert.False(t, verified)
	}
}

func TestGenerateInvalidProofFixture(t *testing.T) {
	updateFixture := &fixture{}
	err := loadFixture("./fixtures/smt_invalid_proof_fixtures.json", updateFixture)
	assert.NoError(t, err)

	for _, testCase := range updateFixture.TestCases {
		t.Log(testCase.Description)
		database, _ := db.NewInMemoryDB()
		smt := NewTrie([]byte{}, 32)
		keys := make([][]byte, len(testCase.Input.Keys)+len(testCase.Input.DeleteKeys))
		values := make([][]byte, len(testCase.Input.Values)+len(testCase.Input.DeleteKeys))
		for i, key := range testCase.Input.Keys {
			keys[i] = key
			values[i] = testCase.Input.Values[i]
		}
		for i, deleteKey := range testCase.Input.DeleteKeys {
			keys[i+len(testCase.Input.Keys)] = deleteKey
			values[i+len(testCase.Input.Keys)] = []byte{}
		}
		keys, values = UniqueAndSort(keys, values)

		root, err := smt.Update(database, keys, values)
		assert.NoError(t, err)
		assert.Equal(t, []byte(testCase.Output.MerkleRoot), root)

		queryKeys := make([][]byte, len(testCase.Input.QueryKeys))
		for i, k := range testCase.Input.QueryKeys {
			queryKeys[i] = k
		}

		// Create a proof with invalid sibling hashes
		proof := &Proof{
			SiblingHashes: testCase.Output.Proof.SiblingHashes,
		}
		proof.Queries = make([]*QueryProof, len(testCase.Output.Proof.Queries))
		for i, q := range testCase.Output.Proof.Queries {
			proof.Queries[i] = &QueryProof{
				Bitmap: q.Bitmap,
				Key:    q.Key,
				Value:  q.Value,
			}
		}

		verified, err := Verify(queryKeys, proof, testCase.Output.MerkleRoot, 32)
		assert.NotNil(t, err)
		assert.False(t, verified)
	}
}

func TestGenerateProofJumboFixture(t *testing.T) {
	updateFixture := fixture{}
	err := loadEncodedFixture("./fixtures/smt_jumbo_fixtures.json.encoded", &updateFixture)
	assert.NoError(t, err)
	eg, _ := errgroup.WithContext(context.Background())
	for _, testCase := range updateFixture.TestCases[:1] {
		testCase := testCase
		eg.Go(func() error {
			t.Log(testCase.Description)
			database, _ := db.NewInMemoryDB()
			smt := NewTrie([]byte{}, 32)
			keys := make([][]byte, len(testCase.Input.Keys)+len(testCase.Input.DeleteKeys))
			values := make([][]byte, len(testCase.Input.Values)+len(testCase.Input.DeleteKeys))
			for i, key := range testCase.Input.Keys {
				keys[i] = key
				values[i] = testCase.Input.Values[i]
			}
			for i, deleteKey := range testCase.Input.DeleteKeys {
				keys[i+len(testCase.Input.Keys)] = deleteKey
				values[i+len(testCase.Input.Keys)] = []byte{}
			}
			keys, values = UniqueAndSort(keys, values)

			root, err := smt.Update(database, keys, values)

			assert.NoError(t, err)
			assert.Equal(t, []byte(testCase.Output.MerkleRoot), root)

			queryKeys := make([][]byte, len(testCase.Input.QueryKeys))
			for i, k := range testCase.Input.QueryKeys {
				queryKeys[i] = k
			}

			proof, err := smt.Prove(database, queryKeys)
			assert.NoError(t, err)

			assert.Len(t, proof.Queries, len(testCase.Output.Proof.Queries))
			for i, q := range proof.Queries {
				assert.Equal(t, testCase.Output.Proof.Queries[i].Bitmap, q.Bitmap, "bitmap does not match")
				assert.Equal(t, testCase.Output.Proof.Queries[i].Key, q.Key, "key does not match")
				assert.Equal(t, testCase.Output.Proof.Queries[i].Value, q.Value, "value does not match")
			}
			assert.Equal(t, testCase.Output.Proof.SiblingHashes, proof.SiblingHashes, "sibling hash does not match")
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		assert.Nil(t, err)
	}
}

package smt

import (
	"errors"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
)

func Verify(
	queryKeys [][]byte,
	proof *Proof,
	root []byte,
	keyLength int,
) (bool, error) {
	if len(queryKeys) != len(proof.Queries) {
		return false, nil
	}
	for i, key := range queryKeys {
		if len(key) != keyLength {
			return false, nil
		}
		query := proof.Queries[i]
		// Bitmap must not include zero byte at the begging or the end
		if len(query.Bitmap) > 0 && (query.Bitmap[0] == 0 || query.Bitmap[len(query.Bitmap)-1] == 0) {
			return false, nil
		}
		if collection.Equal(key, query.Key) {
			continue
		}
		keyBinary := bytes.ToBools(key)
		queryKeyBinary := bytes.ToBools(query.Key)
		commonPrefix := collection.CommonPrefix(keyBinary, queryKeyBinary)
		binaryBitmap := stripPrefixFalse(bytes.ToBools(query.Bitmap))
		if len(binaryBitmap) > len(commonPrefix) {
			return false, nil
		}
	}
	filterMap := map[string]bool{}
	filteredProofs := []*QueryProof{}
	for _, query := range proof.Queries {
		binaryBitmap := stripPrefixFalse(bytes.ToBools(query.Bitmap))
		queryProof := newQueryProof(
			query.Key,
			query.Value,
			binaryBitmap,
			[][]byte{},
			[][]byte{},
		)
		uniqueKey := string(bytes.FromBools(queryProof.binaryPath()))
		if _, exist := filterMap[uniqueKey]; !exist {
			filteredProofs = append(filteredProofs, queryProof)
			filterMap[uniqueKey] = true
		}
	}
	calculatedRoot, err := CalculateRoot(proof.SiblingHashes, filteredProofs)
	if err != nil {
		return false, err
	}

	return bytes.Equal(root, calculatedRoot), nil
}

func CalculateRoot(siblingHashes []codec.Hex, queries QueryProofs) ([]byte, error) {
	queries.sort()
	nextSiblingHash := 0
	for len(queries) > 0 {
		query := queries[0]
		queries = queries[1:]

		if query.height() == 0 {
			return query.hash, nil
		}

		var siblingHash []byte
		if len(queries) > 0 && query.isSiblingOf(queries[0]) { //nolint:gocritic // prefer if/else
			sibling := queries[0]
			queries = queries[1:]
			siblingHash = sibling.hash
		} else if !query.binaryBitmap[0] {
			siblingHash = emptyHash
		} else if query.binaryBitmap[0] {
			if len(siblingHashes) == nextSiblingHash {
				return nil, errors.New("no more sibling hashes available")
			}
			siblingHash = siblingHashes[nextSiblingHash]
			nextSiblingHash += 1
		}
		if len(siblingHash) == 0 {
			return nil, errors.New("sibling hash is not set")
		}

		dir := query.binaryKey()[query.height()-1]
		if !dir {
			branchNode := newBranchNode(query.hash, siblingHash)
			query.hash = branchNode.hash
		} else {
			branchNode := newBranchNode(siblingHash, query.hash)
			query.hash = branchNode.hash
		}
		query.sliceBinaryBitmap(1)
		queries = insertAndFilterQueries(query, queries)
	}
	return nil, errors.New("fail to compute root")
}

package rmt

import "bytes"

func VerifyProof(
	queryHashes [][]byte,
	proof *Proof,
	rootHash []byte,
) bool {
	if proof.Size == 0 {
		return false
	}
	calculatedTree, err := calculatePathNodes(queryHashes, proof.Size, proof.Idxs, proof.SiblingHashes)
	if err != nil {
		return false
	}
	calculatedRoot, exist := calculatedTree[rootIndex]
	if !exist {
		return false
	}
	return bytes.Equal(calculatedRoot, rootHash)
}

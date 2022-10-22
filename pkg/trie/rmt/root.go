package rmt

import (
	"errors"
	"fmt"
	"math"

	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
)

// CalculateRoot returns a root hash with given data set.
func CalculateRoot(data [][]byte) []byte {
	result := make(chan []byte, 1)
	calculateRoot(data, result)
	return <-result
}

func calculateRoot(data [][]byte, result chan []byte) {
	if len(data) == 0 {
		result <- emptyHash
		return
	}
	if len(data) == 1 {
		result <- leafHash(data[0])
		return
	}

	divider := int(math.Pow(2, math.Floor(math.Log2(float64(len(data))-1))))
	leftCh := make(chan []byte, 1)
	rightCh := make(chan []byte, 1)
	go calculateRoot(data[:divider], leftCh)
	go calculateRoot(data[divider:], rightCh)
	leftRes := <-leftCh
	rightRes := <-rightCh
	result <- branchHash(append(leftRes, rightRes...))
}

type RootWithAppendPath struct {
	Root       []byte
	AppendPath [][]byte
	Size       uint64
}

func CalculateRootFromAppendPath(value []byte, appendPath [][]byte, size uint64) *RootWithAppendPath {
	newLeafHash := leafHash(value)
	currentHash := bytes.Copy(newLeafHash)
	binaryKeys := intToBinary(size)
	count := 0
	for i := 0; i < len(binaryKeys); i++ {
		if (size>>i)&1 == 1 {
			siblingHash := appendPath[count]
			currentHash = branchHash(append(siblingHash, currentHash...))
			count++
		}
	}
	newRoot := currentHash
	height := getHeight(size)
	subTreeIndex := uint64(0)
	for subTreeIndex < height && (size>>subTreeIndex)&1 == 1 {
		subTreeIndex++
	}
	currentHash = newLeafHash
	splicedPath := appendPath[subTreeIndex:]
	for _, sibling := range appendPath {
		currentHash = branchHash(append(sibling, currentHash...))
	}
	newAppendPath := append([][]byte{currentHash}, splicedPath...)

	return &RootWithAppendPath{
		Root:       newRoot,
		AppendPath: newAppendPath,
		Size:       size + 1,
	}
}

func getRootFromPath(paths [][]byte) []byte {
	currentHash := paths[0]
	for i := 1; i < len(paths); i++ {
		currentHash = branchHash(append(paths[i], currentHash...))
	}
	return currentHash
}

func CalculateRootFromRightWitness(
	nodeIndex uint64,
	appendPath [][]byte,
	rightWitness [][]byte,
) []byte {
	if len(appendPath) == 0 {
		return getRootFromPath(rightWitness)
	}
	if len(rightWitness) == 0 {
		return getRootFromPath(appendPath)
	}
	updatingAppendPath := appendPath
	updatingRightWitness := rightWitness

	layerIndex := 0
	incrementalIdx := nodeIndex
	firstAppendPath, updatingAppendPath := updatingAppendPath[0], updatingAppendPath[1:]
	firstRightWitness, updatingRightWitness := updatingRightWitness[0], updatingRightWitness[1:]
	currentHash := branchHash(append(firstAppendPath, firstRightWitness...))
	incrementalIdxInit := false

	for len(updatingAppendPath) > 0 || len(updatingRightWitness) > 0 {
		idxDigit := (nodeIndex >> layerIndex) & 1
		if len(updatingAppendPath) > 0 && idxDigit == 1 {
			if !incrementalIdxInit {
				incrementalIdx += 1 << layerIndex
				incrementalIdxInit = true
			} else {
				var leftHash []byte
				leftHash, updatingAppendPath = updatingAppendPath[0], updatingAppendPath[1:]
				currentHash = branchHash(append(leftHash, currentHash...))
			}
		}
		incrementalIdxDigit := (incrementalIdx >> layerIndex) & 1
		if len(updatingRightWitness) > 0 && incrementalIdxDigit == 1 {
			var rightHash []byte
			rightHash, updatingRightWitness = updatingRightWitness[0], updatingRightWitness[1:]
			currentHash = branchHash(append(currentHash, rightHash...))
			incrementalIdx += 1 << layerIndex
		}
		layerIndex++
	}
	return currentHash
}

func VerifyRightWitness(
	nodeIndex uint64,
	appendPath [][]byte,
	rightWitness [][]byte,
	root []byte,
) bool {
	calculatedRoot := CalculateRootFromRightWitness(
		nodeIndex,
		appendPath,
		rightWitness,
	)
	return bytes.Equal(calculatedRoot, root)
}

func CalculateRootFromUpdateData(
	updateData [][]byte,
	proof *Proof,
) ([]byte, error) {
	if proof.Size == 0 || len(proof.Idxs) == 0 {
		return nil, errors.New("invalid proof. size and index must not be empty")
	}
	if len(updateData) != len(proof.Idxs) {
		return nil, errors.New("invalid proof. Size of UpdateData must equal to indexes")
	}
	updateHashes := make([][]byte, len(updateData))
	for i, data := range updateData {
		leaf := leafHash(data)
		updateHashes[i] = leaf
	}
	idxs := make([]int, len(proof.Idxs))
	for i, idx := range proof.Idxs {
		idxs[i] = int(idx)
	}
	calculatedTree, err := calculatePathNodes(updateHashes, proof.Size, proof.Idxs, proof.SiblingHashes)
	if err != nil {
		return nil, err
	}
	root, exist := calculatedTree[rootIndex]
	if !exist {
		return nil, fmt.Errorf("invalid path calculated")
	}
	return root, nil
}

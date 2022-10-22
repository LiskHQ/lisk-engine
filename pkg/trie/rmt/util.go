package rmt

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

const (
	rootIndex = 2
)

func leafHash(msg []byte) []byte {
	return crypto.Hash(append([]byte{leafPrefix}, msg...))
}

func branchHash(msg []byte) []byte {
	return crypto.Hash(append([]byte{branchPrefix}, msg...))
}

func getHeight(size uint64) uint64 {
	return uint64(math.Ceil(math.Log2(float64(size)))) + 1
}

func getLayerStructure(size uint64) []int {
	structure := []int{}
	height := getHeight(size)
	for layer := uint64(0); layer < height; layer++ {
		max := size
		r := 0
		for j := uint64(0); j < layer; j++ {
			even := r%2 == 0
			r += int(max % 2)
			if even {
				max = uint64(math.Floor(float64(max) / 2))
			} else {
				max = uint64(math.Ceil(float64(max) / 2))
			}
		}
		structure = append(structure, int(max))
	}
	return structure
}

func areSameLayer(idx1, idx2 uint64) bool {
	sizeI := len(strconv.FormatInt(int64(idx1), 2))
	sizeJ := len(strconv.FormatInt(int64(idx2), 2))
	return sizeI == sizeJ
}

func length(idx uint64) uint64 {
	return uint64(len(strconv.FormatInt(int64(idx), 2)))
}

func areSiblings(idx1, idx2 uint64) bool {
	return idx1^idx2 == 1
}

func intToBytesWithoutLeadingZero(key uint64) []byte {
	res := make([]byte, 8)
	binary.BigEndian.PutUint64(res, key)
	index := 0
	for i, v := range res {
		if v != 0 {
			index = i
		}
	}
	return res[index:]
}

func bytesToBoolsWithSize(key []byte, size int) []bool {
	res := make([]bool, 8*len(key))
	for i, x := range key {
		for j := 0; j < 8; j++ {
			res[8*i+j] = (x<<uint(j))&0x80 == 0x80
		}
	}
	return res[len(res)-size:]
}

func intToBinary(key uint64) []bool {
	b := intToBytesWithoutLeadingZero(key)
	size := int(math.Floor(math.Log2(float64(key)))) + 1
	return bytesToBoolsWithSize(b, size)
}

type nodeLocation struct {
	nodeIndex  uint64
	layerIndex uint64
}

func newNodeLocationFromKey(key []byte) *nodeLocation {
	layerIndex := bytes.ToUint64(key[:8])
	nodeIndex := bytes.ToUint64(key[8:])
	return &nodeLocation{
		layerIndex: layerIndex,
		nodeIndex:  nodeIndex,
	}
}

func (l nodeLocation) key() []byte {
	layerIndex := bytes.FromUint64(l.layerIndex)
	nodeIndex := bytes.FromUint64(l.nodeIndex)
	return bytes.JoinSize(16, layerIndex, nodeIndex)
}

func (l *nodeLocation) index(height uint64) (uint64, error) {
	length := height - l.layerIndex
	if length <= 0 {
		return 0, fmt.Errorf("invalid height %d or layer index %d", height, l.layerIndex)
	}
	numStr := strconv.FormatInt(int64(l.nodeIndex), 2)
	for len(numStr) < int(length) {
		numStr = "0" + numStr
	}
	res, err := strconv.ParseInt("1"+numStr, 2, 32)
	return uint64(res), err
}

func newNodeLocation(index uint64, height uint64) (*nodeLocation, error) {
	numStr := strconv.FormatInt(int64(index), 2)
	if numStr[0] != '1' {
		return nil, errors.New("index must start from 1")
	}
	nodeIndexBinaryStr := numStr[1:]
	nodeIndex, err := strconv.ParseInt(nodeIndexBinaryStr, 2, 32)
	if err != nil {
		return nil, err
	}
	layerIndex := int(height) - len(nodeIndexBinaryStr)
	if layerIndex < 0 {
		return nil, fmt.Errorf("invalid height %d or layer index %d", height, layerIndex)
	}
	return &nodeLocation{
		nodeIndex:  uint64(nodeIndex),
		layerIndex: uint64(layerIndex),
	}, nil
}

func getRightSiblingInfo(nodeIndex uint64, layerIndex uint64, size uint64) (*nodeLocation, bool) {
	structure := getLayerStructure(size)
	siblingNodeIndex := ((nodeIndex >> 1) << 1) + ((nodeIndex + 1) % 2)
	siblingLayerIndex := layerIndex
	for siblingNodeIndex >= uint64(structure[siblingLayerIndex]) && siblingLayerIndex > 0 {
		siblingNodeIndex <<= 1
		siblingLayerIndex -= 1
	}
	if siblingNodeIndex >= size {
		return nil, false
	}
	return &nodeLocation{
		nodeIndex:  siblingNodeIndex,
		layerIndex: siblingLayerIndex,
	}, true
}

type indexes []uint64

func (idxs *indexes) sort() {
	original := *idxs
	sort.Slice(original, func(i, j int) bool {
		sizeI := len(strconv.FormatInt(int64(original[i]), 2))
		sizeJ := len(strconv.FormatInt(int64(original[j]), 2))
		if sizeI == sizeJ {
			return original[i] < original[j]
		}
		return original[i] > original[j]
	})
	*idxs = original
}

func (idxs *indexes) insert(idx uint64) {
	original := *idxs
	index := findInsertIndex(original, idx)
	if len(original) <= index {
		original = append(original, idx)
	} else if original[index] != idx {
		original = append(original, 0)
		copy(original[index+1:], original[index:])
		original[index] = idx
	}
	*idxs = original
}

func (idxs indexes) remove(val uint64) indexes {
	result := indexes{}
	for _, idx := range idxs {
		if idx != val {
			result = append(result, idx)
		}
	}
	return result
}

func (idxs indexes) findIndex(val uint64) int {
	for i, idx := range idxs {
		if idx == val {
			return i
		}
	}
	return -1
}

func isLeft(index uint64) bool {
	return (index & 1) == 0
}

func findInsertIndex(arr []uint64, idx uint64) int {
	low := -1
	high := len(arr)
	for 1+low < high {
		middle := low + ((high - low) >> 1)
		if arr[middle] == idx {
			return middle
		}
		middleVal := arr[middle]

		sizeI := len(strconv.FormatInt(int64(idx), 2))
		sizeJ := len(strconv.FormatInt(int64(middleVal), 2))
		if sizeI == sizeJ {
			if idx < middleVal {
				high = middle
			} else {
				low = middle
			}
		} else {
			if idx > middleVal {
				high = middle
			} else {
				low = middle
			}
		}
	}
	return high
}

func calculatePathNodes(queryHashes [][]byte, size uint64, idxs []uint64, siblingHashes [][]byte) (map[uint64][]byte, error) {
	copiedSiblings := make([][]byte, len(siblingHashes))
	copy(copiedSiblings, siblingHashes)
	if len(queryHashes) != len(idxs) {
		return nil, errors.New("size of queryHashes and indexes does not match")
	}
	if len(queryHashes) == 0 {
		return nil, errors.New("queryHashes and indexes cannot be empty")
	}
	result := map[uint64][]byte{}
	sortedIndexes := indexes{}
	for i, idx := range idxs {
		if idx == 0 {
			continue
		}
		sortedIndexes = append(sortedIndexes, idx)
		result[idx] = queryHashes[i]
	}
	sortedIndexes.sort()
	height := getHeight(size)
	parentCache := map[uint64][]byte{}
	for len(sortedIndexes) > 0 {
		idx := sortedIndexes[0]
		if idx == rootIndex {
			return result, nil
		}
		currentHash, exist := result[idx]
		if !exist {
			existInCache := false
			currentHash, existInCache = parentCache[idx]
			if !existInCache {
				return nil, fmt.Errorf("invalid state. Hash for index %d should exist", idx)
			}
		}
		parentIdx := idx >> 1
		currentLoc, err := newNodeLocation(idx, height)
		if err != nil {
			return nil, err
		}
		siblingLoc, siblingExist := getRightSiblingInfo(currentLoc.nodeIndex, currentLoc.layerIndex, size)
		if siblingExist {
			siblingIdx, err := siblingLoc.index(height)
			if err != nil {
				return nil, err
			}
			siblingHash, siblingHashExist := result[siblingIdx]
			if !siblingHashExist {
				siblingHash = copiedSiblings[0]
				copiedSiblings = copiedSiblings[1:]
			}
			var parentHash []byte
			if isLeft(idx) {
				parentHash = branchHash(append(currentHash, siblingHash...))
			} else {
				parentHash = branchHash(append(siblingHash, currentHash...))
			}
			if existingParentHash, exist := result[parentIdx]; exist && !bytes.Equal(existingParentHash, parentHash) {
				return nil, fmt.Errorf("invalid query hashes")
			}
			result[parentIdx] = parentHash
		} else {
			parentCache[parentIdx] = currentHash
		}
		sortedIndexes = sortedIndexes[1:]
		sortedIndexes.insert(parentIdx)
	}
	return result, nil
}

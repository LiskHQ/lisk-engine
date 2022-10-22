package smt

import (
	"sort"

	"github.com/LiskHQ/lisk-engine/pkg/collection"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
)

func calculateSubTree(
	nodes []*node,
	structure []uint8,
	height uint8,
	hasher nodeHasher,
	tempHolder []*nodeStructure,
) (*subTree, error) {
	if height == 0 {
		return newSubtreeFromData([]uint8{0}, nodes, hasher)
	}
	nextNodes := []*node{}
	nextStructure := []uint8{}
	i := 0
	for i < len(nodes) {
		if structure[i] != height {
			nextNodes = append(nextNodes, nodes[i])
			nextStructure = append(nextStructure, structure[i])
			i += 1
			continue
		}
		var parentNode *node
		if nodes[i].kind == nodeKindEmpty && nodes[i+1].kind == nodeKindEmpty { //nolint:gocritic // prefer if/else
			parentNode = nodes[i]
		} else if nodes[i].kind == nodeKindEmpty && nodes[i+1].kind == nodeKindLeaf {
			parentNode = nodes[i+1]
		} else if nodes[i].kind == nodeKindLeaf && nodes[i+1].kind == nodeKindEmpty {
			parentNode = nodes[i]
		} else {
			var leftNodes, rightNodes []*node
			var leftStructure, rightStructure []uint8
			if nodes[i].kind == nodeKindTemp {
				last := tempHolder[len(tempHolder)-1]
				tempHolder = tempHolder[:len(tempHolder)-1]
				leftNodes, leftStructure = last.nodes, last.structure
			} else {
				leftNodes, leftStructure = []*node{nodes[i]}, []uint8{structure[i]}
			}
			if nodes[i+1].kind == nodeKindTemp {
				last := tempHolder[len(tempHolder)-1]
				tempHolder = tempHolder[:len(tempHolder)-1]
				rightNodes, rightStructure = last.nodes, last.structure
			} else {
				rightNodes, rightStructure = []*node{nodes[i+1]}, []uint8{structure[i+1]}
			}
			tempStructure := append(leftStructure, rightStructure...) //nolint:gocritic // prefer if/else
			tempNodes := append(leftNodes, rightNodes...)             //nolint:gocritic // prefer if/else
			temp := newTempNode()
			parentNode = temp
			tempHolder = append([]*nodeStructure{{nodes: tempNodes, structure: tempStructure}}, tempHolder...)
		}
		nextNodes = append(nextNodes, parentNode)
		nextStructure = append(nextStructure, structure[i]-1)
		// used 2 nodes
		i += 2
	}
	if height == 1 {
		if nextNodes[0].kind == nodeKindTemp {
			return newSubtreeFromData(tempHolder[0].structure, tempHolder[0].nodes, hasher)
		}
		return newSubtreeFromData([]uint8{0}, nextNodes, hasher)
	}
	return calculateSubTree(
		nextNodes,
		nextStructure,
		height-1,
		hasher,
		tempHolder,
	)
}

type sortingKV struct {
	key   []byte
	value []byte
}

func UniqueAndSort(keys, values [][]byte) ([][]byte, [][]byte) {
	kvMap := map[string]int{}
	sortingKVs := []sortingKV{}
	index := 0
	for i, key := range keys {
		existingIndex, exist := kvMap[string(key)]
		if !exist {
			sortingKVs = append(sortingKVs, sortingKV{
				key:   key,
				value: values[i],
			})
			kvMap[string(key)] = index
			index += 1
			continue
		}
		// if already exist overwrite
		sortingKVs[existingIndex] = sortingKV{
			key:   key,
			value: values[i],
		}
	}

	sort.Slice(sortingKVs, func(i, j int) bool {
		return bytes.Compare(sortingKVs[i].key, sortingKVs[j].key) < 0
	})
	sortedKeys := make([][]byte, len(sortingKVs))
	sortedValues := make([][]byte, len(sortingKVs))
	for i, entry := range sortingKVs {
		sortedKeys[i] = entry.key
		sortedValues[i] = entry.value
	}
	return sortedKeys, sortedValues
}

func calculateQueryHashes(
	nodes []*node,
	structure []uint8,
	height uint8,
	targetID int,
	maxIndex int,
	siblingHashes [][]byte,
	ancestorHashes [][]byte,
	binaryBitmap []bool,
) ([][]byte, [][]byte, []bool) {
	if height == 0 {
		return siblingHashes, ancestorHashes, binaryBitmap
	}
	nextNodes := []*node{}
	nextStructure := []uint8{}
	nextSiblingHashes := [][]byte{}
	nextAncestorHashes := [][]byte{}
	nextBinaryBitmap := []bool{}
	nextTargetID := targetID
	i := 0
	for i < len(nodes) {
		if structure[i] != height {
			nextNodes = append(nextNodes, nodes[i])
			nextStructure = append(nextStructure, structure[i])
			i += 1
			continue
		}
		parentNode := newBranchNode(nodes[i].hash, nodes[i+1].hash)
		parentNode.index = maxIndex + i
		nextNodes = append(nextNodes, parentNode)
		nextStructure = append(nextStructure, structure[i]-1)

		if nextTargetID == nodes[i].index {
			nextAncestorHashes = append(nextAncestorHashes, parentNode.hash)
			nextTargetID = parentNode.index
			if nodes[i+1].kind == nodeKindEmpty {
				nextBinaryBitmap = append(nextBinaryBitmap, false)
			} else {
				nextBinaryBitmap = append(nextBinaryBitmap, true)
				nextSiblingHashes = append(nextSiblingHashes, nodes[i+1].hash)
			}
		} else if nextTargetID == nodes[i+1].index {
			nextAncestorHashes = append(nextAncestorHashes, parentNode.hash)
			nextTargetID = parentNode.index
			if nodes[i].kind == nodeKindEmpty {
				nextBinaryBitmap = append(nextBinaryBitmap, false)
			} else {
				nextBinaryBitmap = append(nextBinaryBitmap, true)
				nextSiblingHashes = append(nextSiblingHashes, nodes[i].hash)
			}
		}
		// used 2 nodes
		i += 2
	}
	nextSiblingHashes = collection.Reverse(nextSiblingHashes)
	nextSiblingHashes = append(nextSiblingHashes, siblingHashes...)
	nextAncestorHashes = collection.Reverse(nextAncestorHashes)
	nextAncestorHashes = append(nextAncestorHashes, ancestorHashes...)
	nextBinaryBitmap = append(binaryBitmap, nextBinaryBitmap...)
	return calculateQueryHashes(
		nextNodes,
		nextStructure,
		height-1,
		nextTargetID,
		maxIndex+i,
		nextSiblingHashes,
		nextAncestorHashes,
		nextBinaryBitmap,
	)
}

// insertAndFilterQueries insert query into queries.
func insertAndFilterQueries(query *QueryProof, queries []*QueryProof) []*QueryProof {
	if len(queries) == 0 {
		queries = append(queries, query)
		return queries
	}
	index := collection.BinarySearch(queries, func(val *QueryProof) bool {
		return (query.height() == val.height() && bytes.Compare(query.Key, val.Key) < 0) || query.height() > val.height()
	})
	if index == len(queries) {
		queries = append(queries, query)
		return queries
	}
	original := queries[index]
	if !collection.Equal(query.binaryPath(), original.binaryPath()) {
		queries = collection.Insert(queries, index, query)
	}
	return queries
}

func calculateSiblingHashes(
	queryProofs []*QueryProof,
	ancestorHashes [][]byte,
) [][]byte {
	siblingHashes := [][]byte{}
	for len(queryProofs) > 0 {
		query := queryProofs[0]
		queryProofs = queryProofs[1:]
		if query.height() == 0 {
			continue
		}
		if query.binaryBitmap[0] {
			lastIndex := len(query.siblingHashes) - 1
			nodeHash := query.siblingHashes[lastIndex]
			query.siblingHashes = query.siblingHashes[:lastIndex]
			if bytes.FindIndex(siblingHashes, nodeHash) == -1 && bytes.FindIndex(ancestorHashes, nodeHash) == -1 {
				siblingHashes = append(siblingHashes, nodeHash)
			}
		}
		query.sliceBinaryBitmap(1)
		queryProofs = insertAndFilterQueries(query, queryProofs)
	}
	return siblingHashes
}

func stripPrefixFalse(val []bool) []bool {
	result := []bool{}
	sawTrue := false
	for _, v := range val {
		if sawTrue || v {
			sawTrue = true
			result = append(result, v)
		}
	}
	return result
}

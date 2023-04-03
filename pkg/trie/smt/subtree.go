package smt

import (
	"errors"
	"fmt"

	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/collection/ints"
)

const (
	prefixIntLeafHash   uint8 = 0
	prefixIntBranchHash uint8 = 1
	prefixIntEmpty      uint8 = 2
)

type subTree struct {
	structure []uint8
	root      []byte
	nodes     []*node
}

type nodeHasher func(nodeHashes [][]byte, structure []uint8, height uint8) []byte

func newSubTree(data []byte, keyLength int, hasher nodeHasher) (*subTree, error) {
	if len(data) == 0 {
		return nil, errors.New("fail to create subtree. data size is zero")
	}
	nodeLength := int(data[0]) + 1
	structure := data[1 : nodeLength+1]
	nodeData := data[nodeLength+1:]
	nodes := []*node{}

	idx := 0

	for idx < len(nodeData) {
		switch nodeData[idx] {
		case prefixIntLeafHash:
			key := nodeData[idx+len(prefixLeafHash) : idx+len(prefixLeafHash)+keyLength]
			value := nodeData[idx+len(prefixLeafHash)+keyLength : idx+len(prefixLeafHash)+keyLength+hashSize]
			nodes = append(nodes, newLeafNode(key, value))
			idx += len(prefixLeafHash) + keyLength + hashSize
		case prefixIntBranchHash:
			nodeHash := nodeData[idx+len(prefixBranchHash) : idx+len(prefixBranchHash)+hashSize]
			nodes = append(nodes, newStubNode(nodeHash))
			idx += len(prefixBranchHash) + hashSize
		case prefixIntEmpty:
			nodes = append(nodes, newEmptyNode())
			idx += len(prefixEmpty)
		default:
			return nil, fmt.Errorf("fail to create subtree. invalid data prefix %d at index %d", nodeData[idx], idx)
		}
	}

	return newSubtreeFromData(structure, nodes, hasher)
}

func newSubtreeFromData(structure []uint8, nodes []*node, hasher nodeHasher) (*subTree, error) {
	height := ints.Max(structure...)
	hashes := make([][]byte, len(nodes))
	for i, n := range nodes {
		hashes[i] = n.hash
	}
	calculated := hasher(hashes, structure, height)
	return &subTree{
		structure: structure,
		root:      calculated,
		nodes:     nodes,
	}, nil
}

func newEmptySubTree() *subTree {
	emptyNode := newEmptyNode()
	return &subTree{
		structure: []byte{0},
		nodes:     []*node{emptyNode},
		root:      emptyNode.hash,
	}
}

func (t *subTree) encode() []byte {
	result := [][]byte{}
	// First byte is the length
	nodeLength := len(t.structure) - 1
	result = append(result, []byte{uint8(nodeLength)})
	// Seceond section holds structure
	result = append(result, t.structure)
	// Rest holds node hashes
	for _, n := range t.nodes {
		result = append(result, n.data)
	}
	return bytes.Join(result...)
}

func (t *subTree) clone() *subTree {
	nodes := make([]*node, len(t.nodes))
	for i, n := range t.nodes {
		nodes[i] = n.clone()
	}
	return &subTree{
		structure: t.structure,
		root:      t.root,
		nodes:     nodes,
	}
}

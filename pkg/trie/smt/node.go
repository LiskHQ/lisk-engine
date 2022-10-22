package smt

import (
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

type nodeKind int

const (
	nodeKindEmpty nodeKind = 0
	nodeKindLeaf  nodeKind = 1
	nodeKindStub  nodeKind = 2
	nodeKindTemp  nodeKind = 3
)

var (
	prefixLeafHash   = []byte{0}
	prefixBranchHash = []byte{1}
	prefixEmpty      = []byte{2}
)

type node struct {
	kind  nodeKind
	key   []byte
	data  []byte
	hash  []byte
	index int
}

func newEmptyNode() *node {
	return &node{
		kind:  nodeKindEmpty,
		hash:  emptyHash,
		data:  prefixEmpty,
		key:   []byte{},
		index: 0,
	}
}

func newLeafNode(key, value []byte) *node {
	data := bytes.Join(prefixLeafHash, key, value)
	hash := crypto.Hash(data)
	return &node{
		kind:  nodeKindLeaf,
		hash:  hash,
		data:  data,
		key:   key,
		index: 0,
	}
}

func newBranchNode(left, right []byte) *node {
	data := bytes.Join(prefixBranchHash, left, right)
	hash := crypto.Hash(data)
	return &node{
		kind:  nodeKindStub,
		hash:  hash,
		data:  data,
		key:   []byte{},
		index: 0,
	}
}

func newStubNode(nodeHash []byte) *node {
	data := bytes.Join(prefixBranchHash, nodeHash)
	return &node{
		kind:  nodeKindStub,
		hash:  nodeHash,
		data:  data,
		key:   []byte{},
		index: 0,
	}
}

func newTempNode() *node {
	return &node{
		kind:  nodeKindTemp,
		hash:  []byte{},
		data:  []byte{},
		key:   []byte{},
		index: 0,
	}
}

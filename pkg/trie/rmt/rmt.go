// Package rmt implements regular merkle tree following [LIP-0031].
//
// [LIP-0031]: https://github.com/LiskHQ/lips/blob/main/proposals/lip-0031.md
package rmt

import (
	"errors"
	"fmt"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

const (
	leafPrefix           byte = 0x00
	branchPrefix         byte = 0x01
	storePrefixInfo      byte = 0
	storePrefixHashToLoc byte = 1
	storePrefixLocToHash byte = 2
)

type info struct {
	root       []byte   `fieldNumber:"1"`
	appendPath [][]byte `fieldNumber:"2"`
	size       uint64   `fieldNumber:"3"`
}

var (
	emptyHash = crypto.Hash([]byte{})
)

// Database represens interface to access the data.
type Database interface {
	Get(key []byte) ([]byte, bool)
	Del(key []byte)
	Set(key, val []byte)
}

// RegularMerkleTree holds lisk markle tree.
type RegularMerkleTree struct {
	root       []byte
	appendPath [][]byte
	size       uint64
	db         Database
}

// NewRegularMerkleTree returns new tree with initial value.
func NewRegularMerkleTree(db Database) *RegularMerkleTree {
	tree := &RegularMerkleTree{
		root:       emptyHash,
		appendPath: [][]byte{},
		db:         db,
	}
	return tree
}

// NewRegularMerkleTreeWithPastData returns new tree with existing root and append path.
func NewRegularMerkleTreeWithPastData(db Database) (*RegularMerkleTree, error) {
	tree := &RegularMerkleTree{
		root:       []byte{},
		appendPath: [][]byte{},
		db:         db,
	}
	err := tree.loadInfo()
	return tree, err
}

// Root returns current root of the tree.
func (d *RegularMerkleTree) Root() []byte {
	return d.root
}

// AppendPath returns current append path of the tree.
func (d *RegularMerkleTree) AppendPath() [][]byte {
	return d.appendPath
}

// Size returns current data length of the tree.
func (d *RegularMerkleTree) Size() uint64 {
	return d.size
}

// Append adds value to the tree.
func (d *RegularMerkleTree) Append(value []byte) error {
	newLeafHash := leafHash(value)
	currentHash := bytes.Copy(newLeafHash)
	nodeLoc := &nodeLocation{
		nodeIndex:  d.size,
		layerIndex: 0,
	}
	d.saveNode(newLeafHash, nodeLoc)
	if d.size == 0 {
		d.appendPath = append(d.appendPath, currentHash)
		d.root = currentHash
		d.size++
		return nil
	}
	count := 0
	height := getHeight(d.size)
	for h := uint64(0); h < height; h++ {
		dir := (d.size >> h) & 1
		if dir == 1 {
			appendHash := d.appendPath[count]
			appendLoc, err := d.getLocation(appendHash)
			if err != nil {
				return err
			}
			nextLoc := &nodeLocation{
				nodeIndex:  appendLoc.nodeIndex >> 1,
				layerIndex: appendLoc.layerIndex + 1,
			}
			currentValue := append(appendHash, currentHash...) //nolint: gocritic // intentionally combining 2 slices
			currentHash = branchHash(currentValue)
			if nextLoc.layerIndex == height-1 {
				// add new node to layer above, with node.hash = currentHash
				if err := d.replaceNode(currentHash, nextLoc); err != nil {
					return err
				}
			} else {
				// set node.hash = hash, where node is the rightmost node in the layer above
				d.saveNode(currentHash, nextLoc)
			}
			count++
		}
	}
	d.root = bytes.Copy(currentHash)
	subTreeIndex := uint64(0)
	for subTreeIndex < height && (d.size>>subTreeIndex)&1 == 1 {
		subTreeIndex++
	}
	bottomPath, topPath := d.appendPath[:subTreeIndex], d.appendPath[subTreeIndex:]
	currentHash = bytes.Copy(newLeafHash)
	for _, h := range bottomPath {
		currentHash = branchHash(append(h, currentHash...))
	}
	d.appendPath = append([][]byte{currentHash}, topPath...)
	d.size++
	if err := d.saveInfo(); err != nil {
		return err
	}
	return nil
}

func (d *RegularMerkleTree) GenerateProof(queryHashes [][]byte) (*Proof, error) {
	if d.size == 0 {
		return &Proof{
			Size:          0,
			Idxs:          []uint64{},
			SiblingHashes: [][]byte{},
		}, nil
	}
	idxs, err := d.getIndexes(queryHashes)
	if err != nil {
		return nil, err
	}
	siblingHashes, err := d.getSiblingHashes(idxs)
	if err != nil {
		return nil, err
	}
	return &Proof{
		Size:          d.size,
		SiblingHashes: siblingHashes,
		Idxs:          idxs,
	}, nil
}

func (d *RegularMerkleTree) GenerateRightWitness(nodeIndex uint64) ([][]byte, error) {
	if nodeIndex > d.size {
		return nil, errors.New("index out of range")
	}
	if d.size == 0 {
		return [][]byte{}, nil
	}
	height := getHeight(d.size)
	currentLoc := &nodeLocation{
		nodeIndex:  nodeIndex,
		layerIndex: 0,
	}
	if currentLoc.layerIndex == 0 && currentLoc.nodeIndex == 0 {
		return d.appendPath, nil
	}
	leftTreeLastIndex := nodeIndex - 1
	incrementalIdx := nodeIndex
	rightWitness := [][]byte{}
	for layerIdx := uint64(0); layerIdx < height; layerIdx++ {
		digit := (incrementalIdx >> layerIdx) & 1
		if digit == 0 {
			continue
		}
		nodeIdx := leftTreeLastIndex >> layerIdx
		siblingLoc, exist := getRightSiblingInfo(nodeIdx, layerIdx, d.size)
		if !exist {
			break
		}
		siblingHash, exist := d.getHash(siblingLoc)
		if !exist {
			return nil, fmt.Errorf("sibling at %s does not exist", codec.Hex(siblingHash))
		}

		rightWitness = append(rightWitness, siblingHash)
		incrementalIdx += 1 << layerIdx
	}
	return rightWitness, nil
}

func (d *RegularMerkleTree) Update(idxs []uint64, updateData [][]byte) error {
	height := getHeight(d.size)
	for _, idx := range idxs {
		if length(idx) != height+1 {
			return errors.New("index must be at leaf")
		}
	}
	updateHashes := make([][]byte, len(updateData))
	for i, data := range updateData {
		updateHashes[i] = leafHash(data)
	}
	siblingHashes, err := d.getSiblingHashes(idxs)
	if err != nil {
		return err
	}
	calculatedTree, err := calculatePathNodes(updateHashes, d.size, idxs, siblingHashes)
	if err != nil {
		return err
	}
	for idx, hashedData := range calculatedTree {
		loc, err := newNodeLocation(idx, height)
		if err != nil {
			return err
		}
		d.saveNode(hashedData, loc)
	}
	nextRoot, exist := calculatedTree[rootIndex]
	if !exist {
		return errors.New("root must exist")
	}
	d.root = nextRoot
	if err := d.saveInfo(); err != nil {
		return err
	}
	return nil
}

func (d *RegularMerkleTree) getHash(loc *nodeLocation) ([]byte, bool) {
	return d.db.Get(append([]byte{storePrefixHashToLoc}, loc.key()...))
}

func (d *RegularMerkleTree) getLocation(hash []byte) (*nodeLocation, error) {
	locKey, exist := d.db.Get(append([]byte{storePrefixHashToLoc}, hash...))
	if !exist {
		return nil, fmt.Errorf("location for hash %s does not exist", codec.Hex(hash))
	}
	return newNodeLocationFromKey(locKey), nil
}

func (d *RegularMerkleTree) replaceNode(hash []byte, loc *nodeLocation) error {
	prevValue, exist := d.getHash(loc)
	if !exist {
		return fmt.Errorf("hash %s does not exist", codec.Hex(hash))
	}
	d.db.Del(prevValue)
	d.saveNode(hash, loc)
	return nil
}

func (d *RegularMerkleTree) saveNode(hash []byte, loc *nodeLocation) {
	d.db.Set(append([]byte{storePrefixHashToLoc}, hash...), loc.key())
	d.db.Set(append([]byte{storePrefixHashToLoc}, loc.key()...), hash)
}

func (d *RegularMerkleTree) loadInfo() error {
	infoByte, exist := d.db.Get([]byte{storePrefixInfo})
	if !exist {
		return fmt.Errorf("tree information does not exist")
	}
	info := &info{}
	if err := info.Decode(infoByte); err != nil {
		return err
	}
	d.root = info.root
	d.appendPath = info.appendPath
	d.size = info.size
	return nil
}

func (d *RegularMerkleTree) saveInfo() error {
	info := &info{
		root:       d.root,
		size:       d.size,
		appendPath: d.appendPath,
	}
	infoByte, err := info.Encode()
	if err != nil {
		return err
	}
	d.db.Set([]byte{storePrefixInfo}, infoByte)
	return nil
}

func (d *RegularMerkleTree) getIndexes(queryHashes [][]byte) ([]uint64, error) {
	res := make([]uint64, len(queryHashes))
	height := getHeight(d.size)
	for i, query := range queryHashes {
		loc, err := d.getLocation(query)
		if err != nil {
			res[i] = 0
			continue
		}
		idx, err := loc.index(height)
		if err != nil {
			return nil, err
		}
		res[i] = idx
	}
	return res, nil
}

func (d *RegularMerkleTree) getSiblingHashes(idxs []uint64) ([][]byte, error) {
	sortedIdxs := indexes{}
	originalIdxs := make(indexes, len(idxs))
	for i, idx := range idxs {
		originalIdxs[i] = idx
		if idx != 0 {
			sortedIdxs = append(sortedIdxs, idx)
		}
	}
	height := getHeight(d.size)
	sortedIdxs.sort()
	siblingHashes := [][]byte{}

	for len(sortedIdxs) > 0 {
		currentIdx := sortedIdxs[0]

		if isLeft(currentIdx) &&
			len(sortedIdxs) > 1 &&
			areSameLayer(currentIdx, sortedIdxs[1]) &&
			areSiblings(currentIdx, sortedIdxs[1]) {
			sortedIdxs = sortedIdxs[2:]
			parentIdx := currentIdx >> 1
			sortedIdxs.insert(parentIdx)
			continue
		}

		if currentIdx == rootIndex {
			return siblingHashes, nil
		}

		currentLoc, err := newNodeLocation(currentIdx, height)
		if err != nil {
			return nil, err
		}
		siblingLoc, siblingExist := getRightSiblingInfo(currentLoc.nodeIndex, currentLoc.layerIndex, d.size)
		if siblingExist {
			siblingIdx, err := siblingLoc.index(height)
			if err != nil {
				return nil, err
			}
			if originalIdxs.findIndex(siblingIdx) > -1 {
				sortedIdxs = sortedIdxs.remove(currentIdx)
				parentIdx := currentIdx >> 1
				sortedIdxs.insert(parentIdx)
				continue
			}
			siblingHash, exist := d.getHash(siblingLoc)
			if !exist {
				return nil, fmt.Errorf("sibling at %s does not exist", codec.Hex(siblingHash))
			}
			siblingHashes = append(siblingHashes, siblingHash)
		}
		sortedIdxs = sortedIdxs.remove(currentIdx)
		parentIdx := currentIdx >> 1
		sortedIdxs.insert(parentIdx)
	}
	return siblingHashes, nil
}

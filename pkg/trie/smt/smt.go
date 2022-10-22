// package smt implements sparse merkle tree following [LIP-0039].
//
// [LIP-0039]: https://github.com/LiskHQ/lips/blob/main/proposals/lip-0039.md
package smt

import (
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/collection/ints"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

type DBReader interface {
	Get(key []byte) ([]byte, error)
}

type DBWriter interface {
	Set(key, val []byte) error
	Del(key []byte) error
}

type DBReadWriter interface {
	DBReader
	DBWriter
}

type trie struct {
	root             []byte
	keyLength        int
	subtreeHeight    uint8
	maxNumberOfNodes int
	hasher           nodeHasher
}

const (
	DefaultKeyLength     = 32
	DefaultSubtreeHeight = 8
)

var (
	emptyHash = crypto.Hash([]byte{})
	hashSize  = len(emptyHash)
)

type nodeStructure struct {
	nodes     []*node
	structure []uint8
}

func NewTrie(root []byte, keyLength int) *trie {
	trieRoot := root
	if len(trieRoot) == 0 {
		trieRoot = emptyHash
	}
	keyLengthWithDefault := keyLength
	if keyLength == 0 {
		keyLength = DefaultKeyLength
	}
	return &trie{
		root:             trieRoot,
		keyLength:        keyLengthWithDefault,
		subtreeHeight:    DefaultSubtreeHeight,
		maxNumberOfNodes: 1 << DefaultSubtreeHeight,
		hasher:           treeHasher,
	}
}

func (t *trie) SetSubtreeHeight(subtreeHeight uint8) {
	t.subtreeHeight = subtreeHeight
	t.maxNumberOfNodes = 1 << t.subtreeHeight
}

func (t *trie) Update(db DBReadWriter, keys [][]byte, values [][]byte) ([]byte, error) {
	if len(keys) != len(values) {
		return nil, errors.New("length of keys and values must be equal")
	}
	if len(keys) == 0 {
		return t.root, nil
	}
	// update keys to be unique
	keyMap := map[string]bool{}
	uniqueKeys := [][]byte{}
	uniqueValues := [][]byte{}
	for i, k := range keys {
		if _, exist := keyMap[string(k)]; !exist {
			uniqueKeys = append(uniqueKeys, k)
			uniqueValues = append(uniqueValues, values[i])
			keyMap[string(k)] = true
		}
	}

	// Maybe this should be done outside
	root, err := t.getSubtree(db, t.root)
	if err != nil {
		return nil, err
	}
	newRoot, err := t.updateSubtree(db, uniqueKeys, uniqueValues, root, 0)
	if err != nil {
		return nil, err
	}
	t.root = newRoot.root
	return t.root, nil
}

func (t *trie) Prove(db DBReader, queryKeys [][]byte) (*Proof, error) {
	if len(queryKeys) == 0 {
		return &Proof{
			SiblingHashes: []codec.Hex{},
			Queries:       []*QueryProof{},
		}, nil
	}
	root, err := t.getSubtree(db, t.root)
	if err != nil {
		return nil, err
	}
	queryProofs := make(QueryProofs, len(queryKeys))
	eg := new(errgroup.Group)
	for i, key := range queryKeys {
		i, key := i, key // https://golang.org/doc/faq#closures_and_goroutines
		eg.Go(func() error {
			queryProof, err := t.generateQueryProof(db, root, key, 0)
			if err != nil {
				return err
			}
			queryProofs[i] = queryProof
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	queries := make([]*QueryProof, len(queryProofs))
	ancestorHashes := [][]byte{}
	for i, queryProof := range queryProofs {
		ancestorHashes = append(ancestorHashes, queryProof.ancestorHashes...)
		queries[i] = queryProof.copy()
	}

	queryProofs.sort()
	siblingHashes := calculateSiblingHashes(queryProofs, ancestorHashes)

	return &Proof{
		Queries:       queries,
		SiblingHashes: codec.BytesArrayToHexArray(siblingHashes),
	}, nil
}

func (t *trie) getSubtree(db DBReader, nodeHash []byte) (*subTree, error) {
	if len(nodeHash) == 0 || bytes.Equal(nodeHash, emptyHash) {
		return newEmptySubTree(), nil
	}
	encodedValue, err := db.Get(nodeHash)
	if err != nil {
		return nil, err
	}
	return newSubTree(encodedValue, t.keyLength, t.hasher)
}

func (t *trie) updateSubtree(db DBReadWriter, keys, values [][]byte, currentSubtree *subTree, height int) (*subTree, error) {
	if len(keys) == 0 {
		return currentSubtree, nil
	}

	// divide keys and values to different bins
	keyBins := make([][][]byte, t.maxNumberOfNodes)
	valueBins := make([][][]byte, t.maxNumberOfNodes)
	b := height / 8
	for i, key := range keys {
		binIndex, err := t.getBinIndex(key, height, b)
		if err != nil {
			return nil, err
		}
		keyBins[binIndex] = append(keyBins[binIndex], key)
		valueBins[binIndex] = append(valueBins[binIndex], values[i])
	}

	newNodes := []*node{}
	newStructure := []uint8{}

	binOffset := 0
	for i, currentNode := range currentSubtree.nodes {
		h := currentSubtree.structure[i]
		newOffset := 1 << (t.subtreeHeight - h)
		targetKeys := keyBins[binOffset : binOffset+newOffset]
		targetValues := valueBins[binOffset : binOffset+newOffset]
		lengthBins := make([]int, len(targetKeys))
		sum := 0
		for i, key := range targetKeys {
			sum += len(key)
			lengthBins[i] = sum
		}
		resultChan := make(chan updateNodeResult)
		go t.updateNode(db, targetKeys, targetValues, lengthBins, 0, currentNode, height, h, resultChan)
		result := <-resultChan
		if result.err != nil {
			return nil, result.err
		}
		newNodes = append(newNodes, result.data.nodes...)
		newStructure = append(newStructure, result.data.structure...)
		binOffset += newOffset
	}

	if binOffset != t.maxNumberOfNodes {
		return nil, fmt.Errorf("bin offset should end with %d but received %d", t.maxNumberOfNodes, binOffset)
	}

	maxStructure := ints.Max(newStructure...)
	newSubtree, err := calculateSubTree(
		newNodes,
		newStructure,
		maxStructure,
		t.hasher,
		[]*nodeStructure{},
	)
	if err != nil {
		return nil, err
	}

	encodedSubtree := newSubtree.encode()
	if err := db.Set(newSubtree.root, encodedSubtree); err != nil {
		return nil, err
	}

	return newSubtree, nil
}

type updateNodeResult struct {
	data *nodeStructure
	err  error
}

func newUpdateNodeResult(nodes []*node, structure []uint8) updateNodeResult {
	return updateNodeResult{
		data: &nodeStructure{
			nodes:     nodes,
			structure: structure,
		},
		err: nil,
	}
}

func newErrUpdateNodeResult(err error) updateNodeResult {
	return updateNodeResult{
		data: nil,
		err:  err,
	}
}

func (t *trie) updateNode(db DBReadWriter, keyBins, valueBins [][][]byte, lengthBins []int, lengthBase int, currentNode *node, height int, structurePos uint8, result chan updateNodeResult) {
	totalData := lengthBins[len(lengthBins)-1] - lengthBase
	if totalData == 0 {
		result <- newUpdateNodeResult([]*node{currentNode}, []uint8{structurePos})
		return
	}
	if totalData == 1 {
		index := -1
		for i, length := range lengthBins {
			if length == lengthBase+1 {
				index = i
				break
			}
		}
		if index == -1 {
			result <- newErrUpdateNodeResult(errors.New("invalid index for length bins"))
			return
		}
		if currentNode.kind == nodeKindEmpty {
			if len(valueBins[index][0]) != 0 {
				newLeaf := newLeafNode(keyBins[index][0], valueBins[index][0])
				result <- newUpdateNodeResult([]*node{newLeaf}, []uint8{structurePos})
				return
			}
			result <- newUpdateNodeResult([]*node{currentNode}, []uint8{structurePos})
			return
		}
		if currentNode.kind == nodeKindLeaf && bytes.Equal(currentNode.key, keyBins[index][0]) {
			if len(valueBins[index][0]) != 0 {
				newLeaf := newLeafNode(keyBins[index][0], valueBins[index][0])
				result <- newUpdateNodeResult([]*node{newLeaf}, []uint8{structurePos})
				return
			}
			result <- newUpdateNodeResult([]*node{newEmptyNode()}, []uint8{structurePos})
			return
		}
	}
	if structurePos == t.subtreeHeight {
		if len(keyBins) != 1 || len(valueBins) != 1 {
			result <- newErrUpdateNodeResult(errors.New("invalid key/value length"))
			return
		}
		var bottomSubTree *subTree
		var subtreeErr error
		switch currentNode.kind {
		case nodeKindStub:
			bottomSubTree, subtreeErr = t.getSubtree(db, currentNode.hash)
			if subtreeErr != nil {
				result <- newErrUpdateNodeResult(subtreeErr)
				return
			}
			if err := db.Del(currentNode.hash); err != nil {
				result <- newErrUpdateNodeResult(err)
				return
			}
		case nodeKindEmpty:
			bottomSubTree, subtreeErr = t.getSubtree(db, currentNode.hash)
			if subtreeErr != nil {
				result <- newErrUpdateNodeResult(subtreeErr)
				return
			}
		case nodeKindLeaf:
			bottomSubTree, subtreeErr = newSubtreeFromData([]uint8{0}, []*node{currentNode}, t.hasher)
			if subtreeErr != nil {
				result <- newErrUpdateNodeResult(subtreeErr)
				return
			}
		default:
			result <- newErrUpdateNodeResult(errors.New("invalid index node kind"))
			return
		}

		newSubtree, err := t.updateSubtree(
			db,
			keyBins[0],
			valueBins[0],
			bottomSubTree,
			height+int(structurePos),
		)
		if err != nil {
			result <- newErrUpdateNodeResult(err)
			return
		}
		if len(newSubtree.nodes) == 1 {
			result <- newUpdateNodeResult([]*node{newSubtree.nodes[0]}, []uint8{structurePos})
			return
		}
		newBranch := newStubNode(newSubtree.root)
		result <- newUpdateNodeResult([]*node{newBranch}, []uint8{structurePos})
		return
	}

	var leftNode, rightNode *node
	switch currentNode.kind {
	case nodeKindEmpty:
		leftNode, rightNode = newEmptyNode(), newEmptyNode()
	case nodeKindLeaf:
		if bytes.IsBitSet(currentNode.key, height+int(structurePos)) {
			leftNode, rightNode = newEmptyNode(), currentNode
		} else {
			leftNode, rightNode = currentNode, newEmptyNode()
		}
	default:
		result <- newErrUpdateNodeResult(errors.New("invalid node kind"))
		return
	}
	splitIndex := len(keyBins) / 2
	leftResultChan := make(chan updateNodeResult)
	rightResultChan := make(chan updateNodeResult)
	go t.updateNode(
		db,
		keyBins[:splitIndex],
		valueBins[:splitIndex],
		lengthBins[:splitIndex],
		lengthBase,
		leftNode,
		height,
		structurePos+1,
		leftResultChan,
	)
	go t.updateNode(
		db,
		keyBins[splitIndex:],
		valueBins[splitIndex:],
		lengthBins[splitIndex:],
		lengthBins[splitIndex-1],
		rightNode,
		height,
		structurePos+1,
		rightResultChan,
	)
	leftResult := <-leftResultChan
	if leftResult.err != nil {
		result <- newErrUpdateNodeResult(leftResult.err)
		return
	}
	rightResult := <-rightResultChan
	if rightResult.err != nil {
		result <- newErrUpdateNodeResult(rightResult.err)
		return
	}
	result <- newUpdateNodeResult(
		append(leftResult.data.nodes, rightResult.data.nodes...),
		append(leftResult.data.structure, rightResult.data.structure...),
	)
}

func (t *trie) generateQueryProof(db DBReader, currentSubtree *subTree, queryKey []byte, height int) (*QueryProof, error) {
	if len(queryKey) != t.keyLength {
		return nil, fmt.Errorf("query key must have key length %d", t.keyLength)
	}
	b := height / 8
	binIndex, err := t.getBinIndex(queryKey, height, b)
	if err != nil {
		return nil, err
	}
	for i, node := range currentSubtree.nodes {
		node.index = i
	}
	binOffset := 0
	var currentNode *node
	queryHeight := uint8(0)
	for i, node := range currentSubtree.nodes {
		queryHeight = currentSubtree.structure[i]
		currentNode = node
		newOffset := 1 << (t.subtreeHeight - queryHeight)
		if binOffset <= binIndex && binIndex < binOffset+newOffset {
			break
		}
		binOffset += newOffset
	}
	if currentNode == nil {
		return nil, errors.New("current node is not selected")
	}
	maxStructure := ints.Max(currentSubtree.structure...)

	siblingHashes, ancesancestorHashes, binaryBitmap := calculateQueryHashes(
		currentSubtree.nodes,
		currentSubtree.structure,
		maxStructure,
		currentNode.index,
		len(currentSubtree.nodes),
		[][]byte{},
		[][]byte{},
		[]bool{},
	)

	if currentNode.kind == nodeKindEmpty {
		return newQueryProof(
			queryKey,
			[]byte{},
			binaryBitmap,
			ancesancestorHashes,
			siblingHashes,
		), nil
	}

	if currentNode.kind == nodeKindLeaf {
		ancesancestorHashes = append(ancesancestorHashes, currentNode.hash)
		return newQueryProof(
			currentNode.key,
			currentNode.data[len(prefixLeafHash)+t.keyLength:],
			binaryBitmap,
			ancesancestorHashes,
			siblingHashes,
		), nil
	}

	lowerSubtree, err := t.getSubtree(db, currentNode.hash)
	if err != nil {
		return nil, err
	}

	lowerQueryProof, err := t.generateQueryProof(
		db,
		lowerSubtree,
		queryKey,
		height+int(queryHeight),
	)
	if err != nil {
		return nil, err
	}
	return newQueryProof(
		lowerQueryProof.Key,
		lowerQueryProof.Value,
		append(lowerQueryProof.binaryBitmap, binaryBitmap...),
		append(ancesancestorHashes, lowerQueryProof.ancestorHashes...),
		append(siblingHashes, lowerQueryProof.siblingHashes...),
	), nil
}

func (t *trie) getBinIndex(key []byte, height, b int) (int, error) {
	if t.subtreeHeight == 4 {
		switch height % 8 {
		case 0:
			// take left side
			return int(key[b]) >> 4, nil
		case 4:
			// take right side
			return int(key[b]) >> 15, nil
		default:
			return 0, errors.New("invalid bin index")
		}
	}
	if t.subtreeHeight == 8 {
		return int(key[b]), nil
	}
	return 0, fmt.Errorf("unsupported Subtree height %d", t.subtreeHeight)
}

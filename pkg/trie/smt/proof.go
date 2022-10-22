package smt

import (
	"sort"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

// Proof holds a data for inclusion / exclusion proof.
type Proof struct {
	SiblingHashes []codec.Hex   `json:"siblingHashes" fieldNumber:"1"`
	Queries       []*QueryProof `json:"queries" fieldNumber:"2"`
}

func (p *Proof) Copy() *Proof {
	siblingHashes := make([]codec.Hex, len(p.SiblingHashes))
	for i, sh := range p.SiblingHashes {
		siblingHashes[i] = bytes.Copy(sh)
	}
	return &Proof{
		SiblingHashes: siblingHashes,
		Queries:       p.Queries,
	}
}

// QueryProof holds a query for the proof.
type QueryProof struct {
	Key            codec.Hex `json:"key" fieldNumber:"1"`
	Value          codec.Hex `json:"value" fieldNumber:"2"`
	Bitmap         codec.Hex `json:"bitmap" fieldNumber:"3"`
	binaryBitmap   []bool
	siblingHashes  [][]byte
	ancestorHashes [][]byte
	hash           []byte
}

type QueryProofs []*QueryProof

func (q *QueryProofs) sort() {
	original := *q
	sort.Slice(original, func(i, j int) bool {
		if original[i].height() != original[j].height() {
			return original[i].height() > original[j].height()
		}
		return bytes.Compare(original[i].Key, original[j].Key) < 0
	})
	*q = original
}

func newQueryProof(
	key []byte,
	value []byte,
	binaryBitmap []bool,
	ancestorHashes [][]byte,
	siblingHashes [][]byte,
) *QueryProof {
	hash := emptyHash
	if len(value) != 0 {
		leafNode := newLeafNode(key, value)
		hash = leafNode.hash
	}
	bitmap := bytes.FromBools(binaryBitmap)
	return &QueryProof{
		Key:            key,
		Value:          value,
		Bitmap:         bitmap,
		hash:           hash,
		ancestorHashes: ancestorHashes,
		siblingHashes:  siblingHashes,
		binaryBitmap:   binaryBitmap,
	}
}

func (p *QueryProof) copy() *QueryProof {
	return &QueryProof{
		Key:            collection.Copy(p.Key),
		Value:          collection.Copy(p.Value),
		Bitmap:         collection.Copy(p.Bitmap),
		hash:           collection.Copy(p.hash),
		ancestorHashes: collection.Copy(p.ancestorHashes),
		siblingHashes:  collection.Copy(p.siblingHashes),
		binaryBitmap:   collection.Copy(p.binaryBitmap),
	}
}

func (p *QueryProof) height() int {
	return len(p.binaryBitmap)
}

func (p *QueryProof) sliceBinaryBitmap(index int) {
	p.binaryBitmap = p.binaryBitmap[index:]
	p.Bitmap = bytes.FromBools(p.binaryBitmap)
}

func (p *QueryProof) binaryPath() []bool {
	return p.binaryKey()[:p.height()]
}

func (p *QueryProof) binaryKey() []bool {
	return bytes.ToBools(p.Key)
}

func (p *QueryProof) isSiblingOf(query *QueryProof) bool {
	if len(p.binaryBitmap) != len(query.binaryBitmap) {
		return false
	}
	if !collection.Equal(p.binaryKey()[:p.height()-1], query.binaryKey()[:query.height()-1]) {
		return false
	}

	if !p.binaryKey()[p.height()-1] && query.binaryKey()[p.height()-1] {
		return true
	}
	if p.binaryKey()[p.height()-1] && !query.binaryKey()[p.height()-1] {
		return true
	}

	return false
}

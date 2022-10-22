// Package contradiction package provides functions to determin contradiction between 2 block headers as defined in [LIP-0014].
//
// [LIP-0014]: https://github.com/LiskHQ/lips/blob/main/proposals/lip-0014.md
package contradiction

import (
	"bytes"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
)

type BFTPartialBlockHeader interface {
	Height() uint32
	GeneratorAddress() []byte
	MaxHeightGenerated() uint32
	MaxHeightPrevoted() uint32
}

type BFTBlockHeader interface {
	BFTPartialBlockHeader
	ID() []byte
}

type bftBlockHeader struct {
	id                 []byte
	height             uint32
	generatorAddress   []byte
	maxHeightGenerated uint32
	maxHeightPrevoted  uint32
}

func (h *bftBlockHeader) ID() []byte                 { return h.id }
func (h *bftBlockHeader) Height() uint32             { return h.height }
func (h *bftBlockHeader) GeneratorAddress() []byte   { return h.generatorAddress }
func (h *bftBlockHeader) MaxHeightGenerated() uint32 { return h.maxHeightGenerated }
func (h *bftBlockHeader) MaxHeightPrevoted() uint32  { return h.maxHeightPrevoted }

func NewBFTBlockHeader(header blockchain.SealedBlockHeader) BFTBlockHeader {
	return &bftBlockHeader{
		id:                 header.ID(),
		height:             header.Height(),
		generatorAddress:   header.GeneratorAddress(),
		maxHeightGenerated: header.MaxHeightGenerated(),
		maxHeightPrevoted:  header.MaxHeightPrevoted(),
	}
}

func AreDistinctHeadersContradicting(b1, b2 BFTPartialBlockHeader) bool {
	earlierBlock, laterBlock := b1, b2

	higherMaxHeightPreviouslyForged := earlierBlock.MaxHeightGenerated() > laterBlock.MaxHeightGenerated()
	sameMaxHeightPreviouslyForged := earlierBlock.MaxHeightGenerated() == laterBlock.MaxHeightGenerated()
	higherMaxHeightPrevoted := earlierBlock.MaxHeightPrevoted() > laterBlock.MaxHeightPrevoted()
	sameMaxHeightPrevoted := earlierBlock.MaxHeightPrevoted() == laterBlock.MaxHeightPrevoted()
	higherHeight := earlierBlock.Height() > laterBlock.Height()
	if higherMaxHeightPreviouslyForged ||
		(sameMaxHeightPreviouslyForged && higherMaxHeightPrevoted) ||
		(sameMaxHeightPreviouslyForged && sameMaxHeightPrevoted && higherHeight) {
		earlierBlock, laterBlock = laterBlock, earlierBlock
	}

	if !bytes.Equal(earlierBlock.GeneratorAddress(), laterBlock.GeneratorAddress()) {
		return false
	}
	// Violation of the fork choice rule as validator moved to different chain
	// without strictly larger maxHeightPreviouslyForged or larger height as
	// justification. This in particular happens, if a validator is double forging.
	if earlierBlock.MaxHeightPrevoted() == laterBlock.MaxHeightPrevoted() &&
		earlierBlock.Height() >= laterBlock.Height() {
		return true
	}
	if earlierBlock.Height() > laterBlock.MaxHeightGenerated() {
		return true
	}
	if earlierBlock.MaxHeightPrevoted() > laterBlock.MaxHeightPrevoted() {
		return true
	}
	return false
}

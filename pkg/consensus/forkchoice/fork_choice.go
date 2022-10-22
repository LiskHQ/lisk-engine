// Package forkchoice implements block comparison logic for the fork choice mechanism as defined in [LIP-0014].
//
// [LIP-0014]: https://github.com/LiskHQ/lips/blob/main/proposals/lip-0014.md
package forkchoice

import (
	"bytes"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/consensus/validator"
)

type forkChoice struct {
	lastHeader             *blockchain.BlockHeader
	currentHeader          *blockchain.BlockHeader
	lastBlockReceivedAt    *time.Time
	currentBlockReceivedAt time.Time
	slot                   *validator.BlockSlot
}

func NewForkChoice(lastHeader, currentHeader *blockchain.BlockHeader, slot *validator.BlockSlot, lastBlockReceivedAt *time.Time) (*forkChoice, error) {
	// if last block version is not 2 (ex: genesis block), use zero value
	return &forkChoice{
		lastHeader:             lastHeader,
		currentHeader:          currentHeader,
		slot:                   slot,
		lastBlockReceivedAt:    lastBlockReceivedAt,
		currentBlockReceivedAt: time.Now(),
	}, nil
}

func (c *forkChoice) IsValidBlock() bool {
	return c.lastHeader.Height+1 == c.currentHeader.Height && bytes.Equal(c.lastHeader.ID, c.currentHeader.PreviousBlockID)
}

func (c *forkChoice) IsIdenticalBlock() bool {
	return bytes.Equal(c.lastHeader.ID, c.currentHeader.ID)
}

func (c *forkChoice) IsDoubleForging() bool {
	return c.isDuplicateBlock() &&
		bytes.Equal(c.lastHeader.GeneratorAddress, c.currentHeader.GeneratorAddress)
}

func (c *forkChoice) IsTieBreak() bool {
	return c.isDuplicateBlock() &&
		c.slot.GetSlotNumber(c.lastHeader.Timestamp) < c.slot.GetSlotNumber(c.currentHeader.Timestamp) &&
		!c.receivedLastBlockWithinForgingSlot() &&
		c.receivedBlockWithinForgingSlot()
}

func (c *forkChoice) IsDifferentChain() bool {
	return IsDifferentChain(c.lastHeader.MaxHeightPrevoted, c.currentHeader.MaxHeightPrevoted, c.lastHeader.Height, c.currentHeader.Height)
}

func (c *forkChoice) isDuplicateBlock() bool {
	return c.lastHeader.Height == c.currentHeader.Height &&
		c.lastHeader.MaxHeightPrevoted == c.currentHeader.MaxHeightPrevoted &&
		bytes.Equal(c.lastHeader.PreviousBlockID, c.currentHeader.PreviousBlockID)
}

func (c *forkChoice) receivedBlockWithinForgingSlot() bool {
	return c.slot.GetSlotNumber(uint32(c.currentBlockReceivedAt.Unix())) == c.slot.GetSlotNumber(c.currentHeader.Timestamp)
}

func (c *forkChoice) receivedLastBlockWithinForgingSlot() bool {
	// if receivedAt is nil, the block comes from syncing
	if c.lastBlockReceivedAt == nil {
		return true
	}
	return c.slot.GetSlotNumber(uint32(c.lastBlockReceivedAt.Unix())) == c.slot.GetSlotNumber(c.lastHeader.Timestamp)
}

func IsDifferentChain(lastMaxHeightPrevoted, maxHeightPrevoted, lastHeight, height uint32) bool {
	return lastMaxHeightPrevoted < maxHeightPrevoted ||
		(lastHeight < height && lastMaxHeightPrevoted == maxHeightPrevoted)
}

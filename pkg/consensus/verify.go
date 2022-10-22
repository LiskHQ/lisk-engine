package consensus

import (
	"bytes"
	"fmt"
	"time"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/db/diffdb"
)

func (c *Executer) verifyBlock(consensusStore *diffdb.Database, block *blockchain.Block) error {
	blockHeader := block.Header
	// check version
	if blockHeader.Version != 2 {
		return fmt.Errorf("block header version %d must be %d", blockHeader.Version, 2)
	}
	lastBlockHeader := c.chain.LastBlock().Header
	// check height
	if blockHeader.Height != lastBlockHeader.Height+1 {
		return fmt.Errorf("block header %d is not consecutive from last block height %d", blockHeader.Height, lastBlockHeader.Height)
	}
	// check previousBlockID
	if !bytes.Equal(lastBlockHeader.ID, blockHeader.PreviousBlockID) {
		return fmt.Errorf("invalid previous block id %s", blockHeader.PreviousBlockID)
	}
	// Check timestamp
	now := uint32(time.Now().Unix())
	blockSlot := c.blockSlot.GetSlotNumber(blockHeader.Timestamp)
	currentBlockSlot := c.blockSlot.GetSlotNumber(now)
	lastBlockSlot := c.blockSlot.GetSlotNumber(lastBlockHeader.Timestamp)
	if blockSlot > currentBlockSlot {
		return fmt.Errorf("future block with timestamp %d is invalid", blockHeader.Timestamp)
	}
	if blockSlot <= lastBlockSlot {
		return fmt.Errorf("block with timestamp %d is invalid because less or equal to last block slot", blockHeader.Timestamp)
	}
	// check generator address
	generators, err := c.liskBFT.API().GetGeneratorKeys(consensusStore, blockHeader.Height)
	if err != nil {
		return err
	}
	generator, err := generators.AtTimestamp(c.blockSlot, blockHeader.Timestamp)
	if err != nil {
		return err
	}
	if !bytes.Equal(generator.Address(), blockHeader.GeneratorAddress) {
		return fmt.Errorf("invalid block generator. Expected %s but received %s", generator.Address(), blockHeader.GeneratorAddress)
	}
	// check max height prevoted and max height previously forged
	maxHeightPrevoted, _, _, err := c.liskBFT.API().GetBFTHeights(consensusStore)
	if err != nil {
		return err
	}
	if blockHeader.MaxHeightPrevoted != maxHeightPrevoted {
		return fmt.Errorf("invalid maxHeight prevoted. Expected %d but received %d", maxHeightPrevoted, blockHeader.MaxHeightPrevoted)
	}
	contradicting, err := c.liskBFT.API().IsHeaderContradictingChain(consensusStore, blockHeader.Readonly())
	if err != nil {
		return err
	}
	if contradicting {
		return fmt.Errorf("received contradicting block header by %s. Height %d MaxHeightPrevoted %d and MaxHeightGenerated %d",
			blockHeader.GeneratorAddress,
			blockHeader.Height,
			blockHeader.MaxHeightPrevoted,
			blockHeader.MaxHeightGenerated,
		)
	}
	if err := c.verifyAggregateCommit(consensusStore, blockHeader.AggregateCommit); err != nil {
		return err
	}
	// check signature
	valid, err := blockHeader.VerifySignature(c.chain.ChainID(), generator.GeneratorKey())
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("invalid signature %s received from %s", blockHeader.Signature, lastBlockHeader.GeneratorAddress)
	}
	return nil
}

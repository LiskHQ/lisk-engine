package blockchain

import (
	"errors"
	"fmt"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/collection/ints"
	"github.com/LiskHQ/lisk-engine/pkg/db"
)

// ChainConfig is for config value of chain.
type ChainConfig struct {
	ChainID               codec.Hex
	MaxTransactionsLength uint32
	MaxBlockCache         int
	KeepEventsForHeights  int
}

// Chain holds information of a particular chain.
type Chain struct {
	maxTransactionsLength uint32
	maxBlockCache         int
	chainID               codec.Hex
	keepEventsForHeights  int
	// Dynamic values
	database     *db.DB
	dataAccess   *DataAccess
	genesisBlock *Block
}

// NewChain returns new chain with config.
func NewChain(cfg *ChainConfig) *Chain {
	chain := &Chain{
		chainID:               cfg.ChainID,
		maxTransactionsLength: cfg.MaxTransactionsLength,
		maxBlockCache:         cfg.MaxBlockCache,
		keepEventsForHeights:  cfg.KeepEventsForHeights,
	}
	return chain
}

// Init the chain.
func (c *Chain) Init(genesisBlock *Block, db *db.DB) {
	c.database = db
	c.dataAccess = NewDataAccess(c.database, c.maxBlockCache, c.keepEventsForHeights)
	c.genesisBlock = genesisBlock
}

// LastBlock returns latest block on the chain.
func (c *Chain) LastBlock() *Block {
	return c.dataAccess.CachedLastBlock()
}

// GetLastNBlock returns latest N blocks exists.
func (c *Chain) GetLastNBlocks(n int) ([]*Block, error) {
	lastBlock := c.dataAccess.CachedLastBlock()
	startHeight := c.genesisBlock.Header.Height
	if lastBlock.Header.Height > startHeight+uint32(n)-1 {
		startHeight = lastBlock.Header.Height - uint32(n) + 1
	}
	return c.dataAccess.GetBlocksBetweenHeight(startHeight, lastBlock.Header.Height)
}

// ChainID is getter for network identifier for the chain.
func (c *Chain) ChainID() []byte {
	return c.chainID
}

// DataAccess returns access to DB.
func (c *Chain) DataAccess() *DataAccess {
	return c.dataAccess
}

// GenesisBlockExist returns true if matching genesis block exist.
func (c *Chain) GenesisBlockExist(block *Block) (bool, error) {
	existingBlock, err := c.dataAccess.GetBlockByHeight(block.Header.Height)
	if err != nil {
		if errors.Is(err, db.ErrDataNotFound) {
			return false, nil
		}
		return false, err
	}
	if bytes.Equal(existingBlock.Header.ID, block.Header.ID) {
		return true, nil
	}
	return false, fmt.Errorf("blockID %s do not match with existing genesis block %s", block.Header.ID, existingBlock.Header.ID)
}

// AddBlock adds a block on top of the current chain.
func (c *Chain) AddBlock(batch *db.Batch, block *Block, events []*Event, finalizedHeight uint32, removeTemp bool) error {
	if err := c.dataAccess.saveBlock(batch, block, events, finalizedHeight, removeTemp); err != nil {
		return err
	}
	if err := c.database.Write(batch); err != nil {
		return err
	}
	return c.dataAccess.Cache(block)
}

// RemoveBlock removes last block from the top of the current chain.
func (c *Chain) RemoveBlock(batch *db.Batch, saveTemp bool) error {
	lastBlock := c.dataAccess.CachedLastBlock()
	if lastBlock.Header.Height == c.genesisBlock.Header.Height {
		return fmt.Errorf("genesis block cannot be removed")
	}
	if err := c.dataAccess.removeBlock(batch, lastBlock, saveTemp); err != nil {
		return err
	}
	if err := c.database.Write(batch); err != nil {
		return err
	}
	c.dataAccess.RemoveCache()
	return nil
}

// PrepareCache block cache.
func (c *Chain) PrepareCache() error {
	lastBlock, err := c.dataAccess.getLastBlock()
	if err != nil {
		return nil
	}
	if lastBlock.Header.Height == 0 && c.dataAccess.Cached(0) {
		return nil
	}
	if lastBlock.Header.Height > 0 {
		minCacheHeight := ints.Max(0, int(lastBlock.Header.Height)-c.maxBlockCache)
		blocks, err := c.dataAccess.GetBlocksBetweenHeight(uint32(minCacheHeight), lastBlock.Header.Height-1)
		if err != nil {
			return err
		}
		for _, block := range blocks {
			if err := c.dataAccess.Cache(block); err != nil {
				return err
			}
		}
	}
	if err := c.dataAccess.Cache(lastBlock); err != nil {
		return err
	}
	return nil
}

package blockchain

import (
	"fmt"
	"sync"
)

type blockCache struct {
	cachedBlocks  map[string]*Block
	heightIndex   map[uint32]string
	size          int
	maxSize       int
	currentHeight uint32
	mutex         *sync.RWMutex
}

func newBlockCache(maxSize int) *blockCache {
	return &blockCache{
		mutex:        new(sync.RWMutex),
		size:         0,
		maxSize:      maxSize,
		cachedBlocks: make(map[string]*Block),
		heightIndex:  make(map[uint32]string),
	}
}

func (c *blockCache) last() (*Block, bool) {
	c.mutex.RLocker().Lock()
	defer c.mutex.RLocker().Unlock()
	return c.getByHeight(c.currentHeight)
}

func (c *blockCache) get(id []byte) (*Block, bool) {
	c.mutex.RLocker().Lock()
	defer c.mutex.RLocker().Unlock()
	block, exist := c.cachedBlocks[string(id)]
	return block, exist
}

func (c *blockCache) getByHeight(height uint32) (*Block, bool) {
	c.mutex.RLocker().Lock()
	defer c.mutex.RLocker().Unlock()
	id, exist := c.heightIndex[height]
	if !exist {
		return nil, false
	}
	block, exist := c.cachedBlocks[id]
	return block, exist
}

func (c *blockCache) push(block *Block) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.size != 0 && block.Header.Height != c.currentHeight+1 {
		return fmt.Errorf("height %d cannot be added since current height is %d", block.Header.Height, c.currentHeight)
	}
	// remove the oldest one if max size
	if c.size >= c.maxSize {
		oldestHeight := c.currentHeight - uint32(c.maxSize) + 1
		id, exist := c.heightIndex[oldestHeight]
		if !exist {
			return fmt.Errorf("oldest height %d does not exist in cache", oldestHeight)
		}
		delete(c.heightIndex, oldestHeight)
		delete(c.cachedBlocks, id)
		c.size--
	}
	id := string(block.Header.ID)
	c.cachedBlocks[id] = block
	c.heightIndex[block.Header.Height] = id
	c.currentHeight = block.Header.Height
	c.size++
	return nil
}

func (c *blockCache) pop() *Block {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.size == 0 {
		return nil
	}
	id := c.heightIndex[c.currentHeight]
	block := c.cachedBlocks[id]
	delete(c.heightIndex, c.currentHeight)
	delete(c.cachedBlocks, id)
	c.size--
	c.currentHeight--
	return block
}

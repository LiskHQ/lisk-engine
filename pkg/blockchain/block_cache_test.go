package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlockCache(t *testing.T) {
	blocks := []*Block{
		createRandomBlock(1),
		createRandomBlock(2),
		createRandomBlock(3),
		createRandomBlock(4),
	}

	cache := newBlockCache(3)

	for _, block := range blocks {
		cache.push(block)
	}

	lastBlock, exist := cache.last()
	assert.Equal(t, blocks[3], lastBlock)
	assert.True(t, exist)
	assert.Equal(t, 3, cache.size)
	assert.Equal(t, uint32(4), cache.currentHeight)

	cache.pop()

	lastBlock, exist = cache.last()
	assert.Equal(t, blocks[2], lastBlock)
	assert.True(t, exist)
	assert.Equal(t, 2, cache.size)
	assert.Equal(t, uint32(3), cache.currentHeight)

	assert.EqualError(t, cache.push(createRandomBlock(100)), "height 100 cannot be added since current height is 3")
}

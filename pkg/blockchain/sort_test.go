package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlockSort(t *testing.T) {
	blocks := []*Block{
		{
			Header: &BlockHeader{
				Height: 13,
			},
		},
		{
			Header: &BlockHeader{
				Height: 12,
			},
		},
		{
			Header: &BlockHeader{
				Height: 11,
			},
		},
	}
	SortBlockByHeightAsc(blocks)
	assert.Equal(t, uint32(11), blocks[0].Header.Height)
	assert.Equal(t, uint32(13), blocks[2].Header.Height)

	SortBlockByHeightDesc(blocks)
	assert.Equal(t, uint32(13), blocks[0].Header.Height)
	assert.Equal(t, uint32(11), blocks[2].Header.Height)
}

package blockchain

import "sort"

// SortBlockByHeightAsc sorts block with height asc.
func SortBlockByHeightAsc(blocks []*Block) {
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Header.Height < blocks[j].Header.Height
	})
}

// SortBlockByHeightDesc sorts block with height desc.
func SortBlockByHeightDesc(blocks []*Block) {
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Header.Height > blocks[j].Header.Height
	})
}

package validator

import (
	"math"
)

type BlockSlot struct {
	genesisTimestamp uint32
	blockTime        uint32
}

func NewBlockSlot(genesisTimestamp, blockTime uint32) *BlockSlot {
	return &BlockSlot{
		genesisTimestamp: genesisTimestamp,
		blockTime:        blockTime,
	}
}

func (a *BlockSlot) GetSlotNumber(unixTime uint32) int {
	elapsed := unixTime - a.genesisTimestamp
	return int(math.Floor(float64(elapsed) / float64(a.blockTime)))
}

func (a *BlockSlot) GetSlotTime(slot int) uint32 {
	offset := slot * int(a.blockTime)
	return a.genesisTimestamp + uint32(offset)
}

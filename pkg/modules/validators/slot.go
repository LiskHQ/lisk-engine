package validators

import (
	"math"
	"time"
)

// CurrentSlotNumber returns the slotnumber with current time.
func CurrentSlotNumber(genesisTime, interval uint32) int {
	t := time.Now().Unix()
	elapsed := t - int64(genesisTime)
	return int(math.Floor(float64(elapsed) / float64(interval)))
}

// NextSlotNumber returns the next slot number with current time.
func NextSlotNumber(genesisTime, interval uint32) int {
	return CurrentSlotNumber(genesisTime, interval) + 1
}

// SlotNumber returns slot allocated to the unix time given.
func SlotNumber(genesisTime, interval uint32, unixTime int64) int {
	elapsed := unixTime - int64(genesisTime)
	return int(math.Floor(float64(elapsed) / float64(interval)))
}

// SlotTime returns beginning time of the slot.
func SlotTime(genesisTime, interval uint32, slot int) uint32 {
	offset := slot * int(interval)
	return genesisTime + uint32(offset)
}

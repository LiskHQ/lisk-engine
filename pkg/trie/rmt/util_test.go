package rmt

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBinaryExpansion(t *testing.T) {
	assert.Equal(t, []bool{true, true}, intToBinary(3))
	assert.Equal(t, []bool{true, false, false}, intToBinary(4))
	assert.Equal(t, []bool{true, false, true}, intToBinary(5))
	assert.Equal(t, []bool{true, true, false}, intToBinary(6))
	assert.Equal(t, []bool{true, false, false, false}, intToBinary(8))
}

func byteStringToInt(str string) uint64 {
	v, err := strconv.ParseInt(str, 2, 64)
	if err != nil {
		panic(err)
	}
	return uint64(v)
}

func byteStringsToInts(arr []string) []uint64 {
	res := make([]uint64, len(arr))
	for i, str := range arr {
		res[i] = byteStringToInt(str)
	}
	return res
}

func TestIndexesSort(t *testing.T) {
	idxs := indexes(byteStringsToInts([]string{"1001", "100001"}))
	idxs.sort()
	assert.Equal(t, indexes(byteStringsToInts([]string{"100001", "1001"})), idxs)

	idxs = indexes(byteStringsToInts([]string{"100101", "100001"}))
	idxs.sort()
	assert.Equal(t, indexes(byteStringsToInts([]string{"100001", "100101"})), idxs)

	idxs = indexes(byteStringsToInts([]string{"100101", "1001", "100001"}))
	idxs.sort()
	assert.Equal(t, indexes(byteStringsToInts([]string{"100001", "100101", "1001"})), idxs)
}

func TestIndexesInsert(t *testing.T) {
	idxs := indexes(byteStringsToInts([]string{"100001", "100101", "1001"}))
	idxs.insert(byteStringToInt("100001"))
	assert.Equal(t, indexes(byteStringsToInts([]string{"100001", "100101", "1001"})), idxs)

	idxs = indexes(byteStringsToInts([]string{"100001", "100101", "1001"}))
	idxs.insert(byteStringToInt("1001"))
	assert.Equal(t, indexes(byteStringsToInts([]string{"100001", "100101", "1001"})), idxs)

	idxs = indexes(byteStringsToInts([]string{"100001", "100101", "1001"}))
	idxs.insert(byteStringToInt("100011"))
	assert.Equal(t, indexes(byteStringsToInts([]string{"100001", "100011", "100101", "1001"})), idxs)

	idxs = indexes(byteStringsToInts([]string{"100001", "100101", "1001"}))
	idxs.insert(byteStringToInt("1000011"))
	assert.Equal(t, indexes(byteStringsToInts([]string{"1000011", "100001", "100101", "1001"})), idxs)

	idxs = indexes(byteStringsToInts([]string{"100001", "100101", "1001"}))
	idxs.insert(byteStringToInt("10"))
	assert.Equal(t, indexes(byteStringsToInts([]string{"100001", "100101", "1001", "10"})), idxs)
}

func TestNodeLocation(t *testing.T) {
	loc, err := newNodeLocation(8, 3)
	assert.NoError(t, err)
	assert.Equal(t, loc.nodeIndex, uint64(0))
	assert.Equal(t, loc.layerIndex, uint64(0))
}

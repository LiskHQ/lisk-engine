package bytes

import (
	"bytes"
	"sort"
)

// SortBytesLexicographically is type for sorging bytes lexicographically.
type SortBytesLexicographically [][]byte

func (b SortBytesLexicographically) Len() int {
	return len(b)
}

func (b SortBytesLexicographically) Less(i, j int) bool {
	return bytes.Compare(b[i], b[j]) < 0
}

func (b SortBytesLexicographically) Swap(i, j int) {
	b[j], b[i] = b[i], b[j]
}

// Sort the input in lexographical order.
func Sort(values [][]byte) {
	sort.Sort(SortBytesLexicographically(values))
}

// IsSorted checks if the value is sorted.
func IsSorted(values [][]byte) bool {
	return sort.IsSorted(SortBytesLexicographically(values))
}

// Reverse input bytes and return a new slice.
func Reverse(bytes []byte) []byte {
	copied := Copy(bytes)
	for i := len(copied)/2 - 1; i >= 0; i-- {
		opp := len(copied) - 1 - i
		copied[i], copied[opp] = copied[opp], copied[i]
	}
	return copied
}

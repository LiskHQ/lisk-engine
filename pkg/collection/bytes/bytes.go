// Package bytes provides utility functions for byte slices.
package bytes

import "bytes"

// Equal reports whether a and b are the same length and contain the same bytes. A nil argument is equivalent to an empty slice.
//
// Equal is alias for the function from standard library "bytes".
func Equal(a, b []byte) bool {
	return bytes.Equal(a, b)
}

// Compare returns an integer comparing two byte slices lexicographically. The result will be 0 if a == b, -1 if a < b, and +1 if a > b. A nil argument is equivalent to an empty slice.
//
// Compare is alias for the function from standard library "bytes".
func Compare(a, b []byte) int {
	return bytes.Compare(a, b)
}

// Repeat returns a new byte slice consisting of count copies of b.
//
// It panics if count is negative or if the result of (len(b) * count) overflows.
// Repeat is alias for the function from standard library "bytes".
func Repeat(b []byte, count int) []byte {
	return bytes.Repeat(b, count)
}

// NewReader returns a new Reader reading from b.
//
// NewReader is alias for the function from standard library "bytes".
func NewReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}

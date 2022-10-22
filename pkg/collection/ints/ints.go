// Package ints provides utility functions for int slices.
package ints

// Integer is type for generic for any integers.
type Integer interface {
	~int | ~uint | ~int8 | ~uint8 | ~int16 | ~uint16 | ~int32 | ~uint32 | ~int64 | ~uint64
}

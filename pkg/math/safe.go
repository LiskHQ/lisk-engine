package math

import "math/bits"

// SafeSub returns x-y and checks for overflow.
func SafeSub(x, y uint64) (uint64, bool) {
	diff, borrowOut := bits.Sub64(x, y, 0)
	return diff, borrowOut == 0
}

// SafeAdd returns x+y and checks for overflow.
func SafeAdd(x, y uint64) (uint64, bool) {
	sum, carryOut := bits.Add64(x, y, 0)
	return sum, carryOut == 0
}

// SafeMul returns x*y and checks for overflow.
func SafeMul(x, y uint64) (uint64, bool) {
	hi, lo := bits.Mul64(x, y)
	return lo, hi == 0
}

type UInt interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// SubWithMin returns x - y and return min if less than zero.
func SafeSubWithMin[T UInt](x, y, min T) T {
	if x < y {
		return min
	}
	return x - y
}

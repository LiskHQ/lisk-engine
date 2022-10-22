package bytes

// Join joins multiple different bytes.
func Join(s ...[]byte) []byte {
	n := 0
	for _, v := range s {
		n += len(v)
	}

	b, i := make([]byte, n), 0
	for _, v := range s {
		i += copy(b[i:], v)
	}
	return b
}

// JoinSize joins bytes with predefined size. (faster than join).
func JoinSize(size int, s ...[]byte) []byte {
	b, i := make([]byte, size), 0
	for _, v := range s {
		i += copy(b[i:], v)
	}
	return b
}

// JoinSlice joins 2 bytes slices provided and returns a new slice.
// Each element of slice T is copied but each byte in the slice is not copied.
func JoinSlice[T ~[]byte](a, b []T) []T {
	result := make([]T, len(a)+len(b))
	copy(result[:len(a)], a)
	copy(result[len(a):], b)
	return result
}

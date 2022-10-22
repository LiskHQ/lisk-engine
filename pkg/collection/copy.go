package collection

// Copy returns a new splice of the input using standard "copy" function.
func Copy[T any](val []T) []T {
	dest := make([]T, len(val))
	copy(dest, val)
	return dest
}

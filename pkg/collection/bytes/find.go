package bytes

// FindIndex returns the index in the slice if target bytes is in the values provided. It returns -1 if it does not exist.
func FindIndex[T ~[]byte](values []T, target T) int {
	for i, v := range values {
		if Equal(v, target) {
			return i
		}
	}
	return -1
}

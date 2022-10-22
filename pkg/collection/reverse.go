package collection

// Reverse the input slice and return a new slice.
func Reverse[T any](value []T) []T {
	copied := make([]T, len(value))
	copy(copied, value)
	for i := len(copied)/2 - 1; i >= 0; i-- {
		opp := len(copied) - 1 - i
		copied[i], copied[opp] = copied[opp], copied[i]
	}
	return copied
}

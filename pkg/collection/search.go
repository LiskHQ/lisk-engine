package collection

// BinarySearch returns the index where the target should exist based on the less function provided.
// less function should return true when value is less.
func BinarySearch[T any](list []T, less func(T) bool) int {
	low := -1
	high := len(list)

	for 1+low < high {
		mid := low + ((high - low) >> 1)
		if less(list[mid]) {
			high = mid
		} else {
			low = mid
		}
	}

	return high
}

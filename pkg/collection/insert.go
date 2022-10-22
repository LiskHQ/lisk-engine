package collection

// Insert a value into the specified index and return a new slice.
// Elements after the index will be shifted.
func Insert[T any](list []T, index int, val T) []T {
	result := make([]T, len(list)+1)
	for i := 0; i < index; i++ {
		result[i] = list[i]
	}
	result[index] = val

	for i := index + 1; i < len(result); i++ {
		result[i] = list[i-1]
	}

	return result
}

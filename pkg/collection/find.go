package collection

// Find returns first value of slice when predicate returns true. It returns default value of T if no match is found.
func Find[T any](list []T, predicate func(T) bool) T {
	var result T
	for _, val := range list {
		if predicate(val) {
			return val
		}
	}
	return result
}

// FindIndex returns first index when predicate returns true. It returns "-1" if no match is found.
func FindIndex[T any](list []T, predicate func(T) bool) int {
	for i, val := range list {
		if predicate(val) {
			return i
		}
	}
	return -1
}

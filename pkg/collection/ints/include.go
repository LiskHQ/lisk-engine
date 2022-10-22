package ints

// Include returns true if the list contains target integer.
func Include[T Integer](list []T, target T) bool {
	for _, val := range list {
		if val == target {
			return true
		}
	}
	return false
}

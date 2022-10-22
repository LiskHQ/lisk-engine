package ints

// Unique takes integer slice and return unique set of integer.
func Unique[T Integer](list []T) []T {
	temp := map[T]bool{}
	for _, val := range list {
		temp[val] = true
	}
	result := []T{}
	for key := range temp {
		result = append(result, key)
	}
	return result
}

// Unique takes integer slice and return true if it contains unique set.
func IsUnique[T Integer](list []T) bool {
	return len(Unique(list)) == len(list)
}

package bytes

// Unique retuens uniqe value of the input.
func Unique[T ~[]byte](values []T) []T {
	uniqueVal := map[string]bool{}
	for _, val := range values {
		if _, exist := uniqueVal[string(val)]; !exist {
			uniqueVal[string(val)] = true
		}
	}
	result := make([]T, len(uniqueVal))
	index := 0
	for key := range uniqueVal {
		result[index] = []byte(key)
		index++
	}
	return result
}

// IsUnique returns true if the elements of the slice are unique.
func IsUnique[T ~[]byte](values []T) bool {
	return len(Unique(values)) == len(values)
}

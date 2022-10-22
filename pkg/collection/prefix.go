package collection

// CommonPrefix returns a new slice with common elements at the beginning of the both inputs.
func CommonPrefix[T comparable](a, b []T) []T {
	longer, shorter := a, b
	if len(longer) < len(shorter) {
		longer, shorter = shorter, longer
	}
	result := []T{}
	for i, val := range shorter {
		if val != longer[i] {
			return result
		}
		result = append(result, val)
	}
	return result
}

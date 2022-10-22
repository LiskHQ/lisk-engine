package bytes

// Copy is utility function to copy the bytes to a new slice.
func Copy(val []byte) []byte {
	dest := make([]byte, len(val))
	copy(dest, val)
	return dest
}

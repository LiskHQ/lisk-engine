package bytes

// IsBitSet takes in byte array, and check if bitwise index is 0 or 1.
//
// []byte{255, 0, 1} and index is 8, it looks at index 1 in the slice and determin if 0 bit is true/false.
func IsBitSet(bits []byte, index int) bool {
	return ((bits[index/8] << (index % 8)) & 0x80) == 0x80
}

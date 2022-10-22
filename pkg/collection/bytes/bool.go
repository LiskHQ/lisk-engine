package bytes

// FromBools takes a slice of bool and converts to byte slice.
func FromBools(input []bool) []byte {
	res := make([]byte, (len(input)+7)/8)
	targetSize := len(input)
	if targetSize%8 != 0 {
		targetSize += 8 - targetSize%8
	}
	target := make([]bool, targetSize)
	diff := len(target) - len(input)
	copy(target[diff:], input)
	for i, x := range target {
		if x {
			res[i/8] |= 0x80 >> uint(i%8)
		}
	}
	return res
}

// ToBools takes byte slice and converted to corresponding boolean slice.
func ToBools(val []byte) []bool {
	res := make([]bool, 8*len(val))
	for i, x := range val {
		for j := 0; j < 8; j++ {
			res[8*i+j] = (x<<uint(j))&0x80 == 0x80
		}
	}
	return res
}

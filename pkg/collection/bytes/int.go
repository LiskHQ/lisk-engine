package bytes

import "encoding/binary"

// FromUint32 converts uint32 to byte slice with length 4.
func FromUint32(val uint32) []byte {
	heightBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(heightBytes, val)
	return heightBytes
}

// FromUint64 converts uint64 to byte slice with length 8.
func FromUint64(val uint64) []byte {
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, val)
	return heightBytes
}

// FromUint16 converts uint16 to byte slice with length 2.
func FromUint16(val uint16) []byte {
	heightBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(heightBytes, val)
	return heightBytes
}

// ToUint32 converts byte slice to uint32. bytes[4:] will be ignored.
func ToUint32(val []byte) uint32 {
	return binary.BigEndian.Uint32(val)
}

// ToUint64 converts byte slice to uint64. bytes[8:] will be ignored.
func ToUint64(val []byte) uint64 {
	return binary.BigEndian.Uint64(val)
}

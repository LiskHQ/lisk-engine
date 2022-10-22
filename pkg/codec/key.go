package codec

import "encoding/binary"

const (
	wireType0 = 0
	wireType2 = 2
)

// returns field number, wire type, and error from the key.
func readKey(val int) (int, int, error) {
	wireType := val & 7
	if wireType != wireType0 && wireType != wireType2 {
		return 0, 0, ErrInvalidData
	}
	fieldNumber := val >> 3
	return fieldNumber, wireType, nil
}

func getKey(wireType int, fieldNumber int) ([]byte, int) {
	key := (fieldNumber << 3) | wireType
	vint := make([]byte, binary.MaxVarintLen32)
	size := binary.PutUvarint(vint, uint64(key))

	return vint, size
}

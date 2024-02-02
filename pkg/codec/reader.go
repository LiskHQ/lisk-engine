package codec

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const (
	msg8Bit  = 0x80
	rest8Bit = 0x7f
)

// Reader is responsible for reading data in protobuf protocol.
type Reader struct {
	index int
	end   int
	data  []byte
}

// NewReader returns reader with the data given.
func NewReader(data []byte) *Reader {
	return &Reader{
		data:  data,
		index: 0,
		end:   len(data),
	}
}

// ReadUInt reads uint if the field number matches.
func (r *Reader) ReadUInt(fieldNumber int, strict bool) (uint64, error) {
	ok, err := r.check(fieldNumber, wireType0)
	if err != nil {
		if !r.strictError(err) {
			return 0, err
		}
		if strict {
			return 0, err
		}
	}

	if !ok {
		return 0, nil
	}
	return r.readUInt()
}

// ReadUInt32 reads uint32 if the field number matches.
func (r *Reader) ReadUInt32(fieldNumber int, strict bool) (uint32, error) {
	val, err := r.ReadUInt(fieldNumber, strict)
	return uint32(val), err
}

// ReadUInts reads []uint if the field number matches.
func (r *Reader) ReadUInts(fieldNumber int) ([]uint64, error) {
	ok, err := r.check(fieldNumber, wireType2)
	if err != nil && !r.strictError(err) {
		return []uint64{}, err
	}
	if !ok {
		return []uint64{}, nil
	}
	arrayLength, err := r.readUInt()
	if err != nil {
		return []uint64{}, err
	}
	end := r.index + int(arrayLength)
	result := []uint64{}
	for r.index < end {
		val, err := r.readUInt()
		if err != nil {
			return []uint64{}, err
		}
		result = append(result, val)
	}
	return result, nil
}

// ReadUInt32s reads []uint32 if the field number matches.
func (r *Reader) ReadUInt32s(fieldNumber int) ([]uint32, error) {
	ok, err := r.check(fieldNumber, wireType2)
	if err != nil && !r.strictError(err) {
		return []uint32{}, err
	}
	if !ok {
		return []uint32{}, nil
	}
	arrayLength, err := r.readUInt()
	if err != nil {
		return []uint32{}, err
	}
	end := r.index + int(arrayLength)
	result := []uint32{}
	for r.index < end {
		val, err := r.readUInt()
		if err != nil {
			return []uint32{}, err
		}
		result = append(result, uint32(val))
	}
	return result, nil
}

// ReadInt reads int if the field number matches.
func (r *Reader) ReadInt(fieldNumber int, strict bool) (int64, error) {
	ok, err := r.check(fieldNumber, wireType0)
	if err != nil {
		if !r.strictError(err) {
			return 0, err
		}
		if strict {
			return 0, err
		}
	}
	if !ok {
		return 0, nil
	}
	return r.readInt()
}

// ReadInt reads int if the field number matches.
func (r *Reader) ReadInt32(fieldNumber int, strict bool) (int32, error) {
	ok, err := r.check(fieldNumber, wireType0)
	if err != nil {
		if !r.strictError(err) {
			return 0, err
		}
		if strict {
			return 0, err
		}
	}
	if !ok {
		return 0, nil
	}
	res, err := r.readInt()
	if err != nil {
		return 0, err
	}
	return int32(res), nil
}

// ReadInts reads []int if the field number matches.
func (r *Reader) ReadInts(fieldNumber int) ([]int64, error) {
	ok, err := r.check(fieldNumber, wireType2)
	if err != nil && !r.strictError(err) {
		return []int64{}, err
	}
	if !ok {
		return []int64{}, nil
	}
	arrayLength, err := r.readUInt()
	if err != nil {
		return []int64{}, err
	}
	end := r.index + int(arrayLength)
	result := []int64{}
	for r.index < end {
		val, err := r.readInt()
		if err != nil {
			return []int64{}, err
		}
		result = append(result, val)
	}
	return result, nil
}

// ReadBool reads int if the field number matches.
func (r *Reader) ReadBool(fieldNumber int, strict bool) (bool, error) {
	ok, err := r.check(fieldNumber, wireType0)
	if err != nil {
		if !r.strictError(err) {
			return false, err
		}
		if strict {
			return false, err
		}
	}
	if !ok {
		return false, nil
	}
	return r.readBool()
}

// ReadBools reads []bool if the field number matches.
func (r *Reader) ReadBools(fieldNumber int) ([]bool, error) {
	ok, err := r.check(fieldNumber, wireType2)
	if err != nil && !r.strictError(err) {
		return []bool{}, err
	}
	if !ok {
		return []bool{}, nil
	}
	arrayLength, err := r.readUInt()
	if err != nil {
		return []bool{}, err
	}
	end := r.index + int(arrayLength)
	result := []bool{}
	for r.index < end {
		val, err := r.readBool()
		if err != nil {
			return []bool{}, err
		}
		result = append(result, val)
	}
	return result, nil
}

// ReadBytes reads []byte if the field number matches.
func (r *Reader) ReadBytes(fieldNumber int, strict bool) ([]byte, error) {
	ok, err := r.check(fieldNumber, wireType2)
	if err != nil {
		if !r.strictError(err) {
			return []byte{}, err
		}
		if strict {
			return []byte{}, err
		}
	}
	if !ok {
		return []byte{}, nil
	}
	return r.readBytes()
}

// ReadBytesArray reads [][]byte if the field number matches.
func (r *Reader) ReadBytesArray(fieldNumber int) ([][]byte, error) {
	result := [][]byte{}
	for r.index < r.end {
		ok, err := r.check(fieldNumber, wireType2)
		if err != nil && !r.strictError(err) {
			return result, err
		}
		if !ok {
			return result, nil
		}
		val, err := r.readBytes()
		if err != nil {
			return result, err
		}
		result = append(result, val)
	}
	return result, nil
}

// ReadString reads string if the field number matches.
func (r *Reader) ReadString(fieldNumber int, strict bool) (string, error) {
	ok, err := r.check(fieldNumber, wireType2)
	if err != nil {
		if !r.strictError(err) {
			return "", err
		}
		if strict {
			return "", err
		}
	}
	if !ok {
		return "", nil
	}
	return r.readString()
}

// ReadStrings reads []string if the field number matches.
func (r *Reader) ReadStrings(fieldNumber int) ([]string, error) {
	result := []string{}
	for r.index < r.end {
		ok, err := r.check(fieldNumber, wireType2)
		if err != nil && !r.strictError(err) {
			return result, err
		}
		if !ok {
			return result, nil
		}
		val, err := r.readString()
		if err != nil {
			return result, err
		}
		result = append(result, val)
	}
	return result, nil
}

// ReadDecodable reads struct if the field number matches.
func (r *Reader) ReadDecodable(fieldNumber int, creator func() DecodableReader, strict bool) (DecodableReader, error) {
	ok, err := r.check(fieldNumber, wireType2)
	if err != nil {
		if !r.strictError(err) {
			return nil, err
		}
		if strict {
			return nil, err
		}
	}
	if !ok {
		return creator(), nil
	}
	decodableSize, err := r.readUInt()
	if err != nil {
		return nil, err
	}
	decodableReader := &Reader{
		data:  r.data,
		index: r.index,
		end:   r.index + int(decodableSize),
	}
	val := creator()
	if err := val.DecodeFromReader(decodableReader); err != nil {
		return nil, err
	}
	r.index = decodableReader.index
	return val, nil
}

// ReadDecodables reads array of struct if the field number matches.
func (r *Reader) ReadDecodables(fieldNumber int, creator func() DecodableReader) ([]interface{}, error) {
	result := []interface{}{}
	for r.index < r.end {
		ok, err := r.check(fieldNumber, wireType2)
		if err != nil && !r.strictError(err) {
			return result, err
		}
		if !ok {
			return result, nil
		}
		decodableSize, err := r.readUInt()
		if err != nil {
			return result, err
		}
		decodableReader := &Reader{
			data:  r.data,
			index: r.index,
			end:   r.index + int(decodableSize),
		}
		val := creator()
		if err := val.DecodeFromReader(decodableReader); err != nil {
			return nil, err
		}
		r.index = decodableReader.index
		result = append(result, val)
	}
	return result, nil
}

func (r *Reader) HasUnreadBytes() bool {
	return r.index != r.end
}

func (r *Reader) readUInt() (uint64, error) {
	result, size, err := readUint(r.data, r.index)
	if err != nil {
		return 0, err
	}

	r.index += size
	return result, nil
}

func (r *Reader) readInt() (int64, error) {
	res, err := r.readUInt()
	if err != nil {
		return 0, err
	}
	if res%2 == 0 {
		return int64(res / 2), nil
	}
	return -1 * int64((res+1)/2), nil
}

func (r *Reader) readBytes() ([]byte, error) {
	size, err := r.readUInt()
	if err != nil {
		return nil, err
	}
	remaining := len(r.data) - r.index
	if size > uint64(remaining) {
		return nil, fmt.Errorf("invalid byte size %d. Remaining data length is %d", size, remaining)
	}
	result := make([]byte, int(size))
	copy(result, r.data[r.index:r.index+int(size)])
	r.index += int(size)
	return result, nil
}

func (r *Reader) readString() (string, error) {
	result, err := r.readBytes()
	if err != nil {
		return "", err
	}
	if !utf8.Valid(result) {
		return "", errors.New("invalid byte for UTF-8 is included")
	}
	if !isNormalizedString(result) {
		return "", errors.New("UTF-8 is not normalized")
	}
	return string(result), nil
}

func (r *Reader) readBool() (bool, error) {
	target := r.data[r.index]
	if target != 0x00 && target != 0x01 {
		return false, ErrInvalidData
	}
	result := target != 0x00
	r.index++
	return result, nil
}

func (r *Reader) peekKey() (int, int, error) {
	result, size, err := readUint(r.data, r.index)
	return int(result), size, err
}

func (r *Reader) check(fieldNumber, wireType int) (bool, error) {
	if r.index >= r.end {
		return false, ErrFieldNumberNotFound
	}
	key, size, err := r.peekKey()
	if err != nil {
		return false, err
	}
	nextFieldNumber, nextWireType, err := readKey(key)
	if err != nil {
		return false, err
	}
	if nextFieldNumber != fieldNumber {
		return false, ErrUnexpectedFieldNumber
	}
	if nextWireType != wireType {
		return false, ErrInvalidData
	}
	r.index += size
	return true, nil
}

func readUint(data []byte, offset int) (uint64, int, error) {
	result := uint64(0)
	index := offset
	for shift := 0; shift < 64; shift += 7 {
		if index >= len(data) {
			return 0, 0, ErrInvalidData
		}
		bit := uint64(data[index])
		index++
		if index == offset+10 && bit > 0x01 {
			return 0, 0, ErrOutOfRange
		}
		result |= (bit & uint64(rest8Bit)) << shift
		if (bit & uint64(msg8Bit)) == 0 {
			var err error
			if varintShortestSize(result) != index-offset {
				err = ErrUnnecessaryLeadingBytes
			}
			return result, index - offset, err
		}
	}
	return 0, 0, ErrNoTerminate
}

func (r *Reader) strictError(err error) bool {
	return errors.Is(err, ErrFieldNumberNotFound) || errors.Is(err, ErrUnexpectedFieldNumber)
}

func varintShortestSize(data uint64) int {
	switch {
	case data < (1 << 7):
		return 1
	case data < (1 << (7 * 2)):
		return 2
	case data < (1 << (7 * 3)):
		return 3
	case data < (1 << (7 * 4)):
		return 4
	case data < (1 << (7 * 5)):
		return 5
	case data < (1 << (7 * 6)):
		return 6
	case data < (1 << (7 * 7)):
		return 7
	case data < (1 << (7 * 8)):
		return 8
	case data < (1 << (7 * 9)):
		return 9
	default:
		return 10
	}
}

func isNormalizedString(data []byte) bool {
	return norm.NFC.IsNormal(data)
}

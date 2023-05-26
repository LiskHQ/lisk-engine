package codec

import (
	"encoding/binary"

	"golang.org/x/text/unicode/norm"
)

// Writer is responsible for writing data in protobuf protocol.
type Writer struct {
	result []byte
	size   int
}

// NewWriter returns a new instances of a writer.
func NewWriter() *Writer {
	return &Writer{
		result: []byte{},
	}
}

// WriteBytes writes a bytes to result.
func (w *Writer) WriteBytes(fieldNumber int, data []byte) {
	w.writeKey(wireType2, fieldNumber)
	w.writeBytes(data)
}

// WriteBytesArray writes bytes array to result.
func (w *Writer) WriteBytesArray(fieldNumber int, data [][]byte) {
	if len(data) == 0 {
		return
	}
	for _, val := range data {
		w.WriteBytes(fieldNumber, val)
	}
}

// WriteString writes a string to result.
func (w *Writer) WriteString(fieldNumber int, data string) {
	w.WriteBytes(fieldNumber, []byte(norm.NFC.String(data)))
}

// WriteStrings writes strings to result.
func (w *Writer) WriteStrings(fieldNumber int, data []string) {
	if len(data) == 0 {
		return
	}
	for _, val := range data {
		w.WriteString(fieldNumber, val)
	}
}

// WriteBool writes a boolean to result.
func (w *Writer) WriteBool(fieldNumber int, data bool) {
	w.writeKey(wireType0, fieldNumber)
	w.writeBool(data)
}

// WriteBools writes booleans to result.
func (w *Writer) WriteBools(fieldNumber int, data []bool) {
	if len(data) == 0 {
		return
	}
	w.writeKey(wireType2, fieldNumber)
	writer := NewWriter()
	for _, val := range data {
		writer.writeBool(val)
	}
	w.writeBytes(writer.Result())
}

// WriteUInt writes uint to result.
func (w *Writer) WriteUInt(fieldNumber int, data uint64) {
	w.writeKey(wireType0, fieldNumber)
	w.writeUInt(data)
}

// WriteUInt32 writes uint to result.
func (w *Writer) WriteUInt32(fieldNumber int, data uint32) {
	w.WriteUInt(fieldNumber, uint64(data))
}

// WriteUInts writes uint to result.
func (w *Writer) WriteUInts(fieldNumber int, data []uint64) {
	if len(data) == 0 {
		return
	}
	w.writeKey(wireType2, fieldNumber)
	writer := NewWriter()
	for _, val := range data {
		writer.writeUInt(val)
	}
	w.writeBytes(writer.Result())
}

// WriteUInt32s writes uint to result.
func (w *Writer) WriteUInt32s(fieldNumber int, data []uint32) {
	if len(data) == 0 {
		return
	}
	w.writeKey(wireType2, fieldNumber)
	writer := NewWriter()
	for _, val := range data {
		writer.writeUInt(uint64(val))
	}
	w.writeBytes(writer.Result())
}

// WriteInt writes int to result.
func (w *Writer) WriteInt(fieldNumber int, data int64) {
	w.writeKey(wireType0, fieldNumber)
	w.writeInt(data)
}

// WriteInt32 writes int to result.
func (w *Writer) WriteInt32(fieldNumber int, data int32) {
	w.WriteInt(fieldNumber, int64(data))
}

// WriteInts writes int to result.
func (w *Writer) WriteInts(fieldNumber int, data []int64) {
	if len(data) == 0 {
		return
	}
	w.writeKey(wireType2, fieldNumber)
	writer := NewWriter()
	for _, val := range data {
		writer.writeInt(val)
	}
	w.writeBytes(writer.Result())
}

// WriteInt32s writes int to result.
func (w *Writer) WriteInt32s(fieldNumber int, data []int32) {
	if len(data) == 0 {
		return
	}
	w.writeKey(wireType2, fieldNumber)
	writer := NewWriter()
	for _, val := range data {
		writer.writeInt(int64(val))
	}
	w.writeBytes(writer.Result())
}

// WriteEncodable writes encodable struct to result.
func (w *Writer) WriteEncodable(fieldNumber int, data Encodable) {
	if data == nil {
		return
	}
	w.writeKey(wireType2, fieldNumber)
	result := data.Encode()
	w.writeBytes(result)
}

// Result returns the written bytes.
func (w *Writer) Result() []byte {
	return w.result
}

// Size returns written size.
func (w *Writer) Size() int {
	return w.size
}

// writeKey writes a key to the byte.
func (w *Writer) writeKey(wireType int, fieldNumber int) {
	vint, size := getKey(wireType, fieldNumber)
	w.size += size
	w.result = append(w.result, vint[0:size]...)
}

// writeBytes writes a bytes to result without key.
func (w *Writer) writeBytes(data []byte) {
	size := len(data)
	w.writeUInt(uint64(size))
	w.size += size
	w.result = append(w.result, data...)
}

// writeUInt writes uint to result without key.
func (w *Writer) writeUInt(data uint64) {
	vint := make([]byte, binary.MaxVarintLen64)
	size := binary.PutUvarint(vint, data)
	w.size = size
	w.result = append(w.result, vint[0:size]...)
}

// writeInt writes int to result without key.
func (w *Writer) writeInt(data int64) {
	vint := make([]byte, binary.MaxVarintLen64)
	size := binary.PutVarint(vint, data)
	w.size = size
	w.result = append(w.result, vint[0:size]...)
}

// writeBool writes bool to result without key.
func (w *Writer) writeBool(data bool) {
	b := make([]byte, 1)
	if data {
		b[0] = 0x01
	} else {
		b[0] = 0x00
	}
	w.size++
	w.result = append(w.result, b...)
}

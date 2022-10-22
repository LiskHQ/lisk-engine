package codec

import "encoding/binary"

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
func (w *Writer) WriteBytes(fieldNumber int, data []byte) error {
	if err := w.writeKey(wireType2, fieldNumber); err != nil {
		return err
	}
	return w.writeBytes(data)
}

// WriteBytesArray writes bytes array to result.
func (w *Writer) WriteBytesArray(fieldNumber int, data [][]byte) error {
	if len(data) == 0 {
		return nil
	}
	for _, val := range data {
		if err := w.WriteBytes(fieldNumber, val); err != nil {
			return nil
		}
	}
	return nil
}

// WriteString writes a string to result.
func (w *Writer) WriteString(fieldNumber int, data string) error {
	return w.WriteBytes(fieldNumber, []byte(data))
}

// WriteStrings writes strings to result.
func (w *Writer) WriteStrings(fieldNumber int, data []string) error {
	if len(data) == 0 {
		return nil
	}
	for _, val := range data {
		if err := w.WriteString(fieldNumber, val); err != nil {
			return nil
		}
	}
	return nil
}

// WriteBool writes a boolean to result.
func (w *Writer) WriteBool(fieldNumber int, data bool) error {
	if err := w.writeKey(wireType0, fieldNumber); err != nil {
		return err
	}
	return w.writeBool(data)
}

// WriteBools writes booleans to result.
func (w *Writer) WriteBools(fieldNumber int, data []bool) error {
	if len(data) == 0 {
		return nil
	}
	if err := w.writeKey(wireType2, fieldNumber); err != nil {
		return err
	}
	writer := NewWriter()
	for _, val := range data {
		if err := writer.writeBool(val); err != nil {
			return err
		}
	}
	if err := w.writeBytes(writer.Result()); err != nil {
		return err
	}
	return nil
}

// WriteUInt writes uint to result.
func (w *Writer) WriteUInt(fieldNumber int, data uint64) error {
	if err := w.writeKey(wireType0, fieldNumber); err != nil {
		return err
	}
	return w.writeUInt(data)
}

// WriteUInt32 writes uint to result.
func (w *Writer) WriteUInt32(fieldNumber int, data uint32) error {
	return w.WriteUInt(fieldNumber, uint64(data))
}

// WriteUInts writes uint to result.
func (w *Writer) WriteUInts(fieldNumber int, data []uint64) error {
	if len(data) == 0 {
		return nil
	}
	if err := w.writeKey(wireType2, fieldNumber); err != nil {
		return err
	}
	writer := NewWriter()
	for _, val := range data {
		if err := writer.writeUInt(val); err != nil {
			return err
		}
	}
	if err := w.writeBytes(writer.Result()); err != nil {
		return err
	}
	return nil
}

// WriteUInt32s writes uint to result.
func (w *Writer) WriteUInt32s(fieldNumber int, data []uint32) error {
	if len(data) == 0 {
		return nil
	}
	if err := w.writeKey(wireType2, fieldNumber); err != nil {
		return err
	}
	writer := NewWriter()
	for _, val := range data {
		if err := writer.writeUInt(uint64(val)); err != nil {
			return err
		}
	}
	if err := w.writeBytes(writer.Result()); err != nil {
		return err
	}
	return nil
}

// WriteInt writes int to result.
func (w *Writer) WriteInt(fieldNumber int, data int64) error {
	if err := w.writeKey(wireType0, fieldNumber); err != nil {
		return err
	}
	return w.writeInt(data)
}

// WriteInt32 writes int to result.
func (w *Writer) WriteInt32(fieldNumber int, data int32) error {
	return w.WriteInt(fieldNumber, int64(data))
}

// WriteInts writes int to result.
func (w *Writer) WriteInts(fieldNumber int, data []int64) error {
	if len(data) == 0 {
		return nil
	}
	if err := w.writeKey(wireType2, fieldNumber); err != nil {
		return err
	}
	writer := NewWriter()
	for _, val := range data {
		if err := writer.writeInt(val); err != nil {
			return err
		}
	}
	if err := w.writeBytes(writer.Result()); err != nil {
		return err
	}
	return nil
}

// WriteInt32s writes int to result.
func (w *Writer) WriteInt32s(fieldNumber int, data []int32) error {
	if len(data) == 0 {
		return nil
	}
	if err := w.writeKey(wireType2, fieldNumber); err != nil {
		return err
	}
	writer := NewWriter()
	for _, val := range data {
		if err := writer.writeInt(int64(val)); err != nil {
			return err
		}
	}
	if err := w.writeBytes(writer.Result()); err != nil {
		return err
	}
	return nil
}

// WriteEncodable writes encodable struct to result.
func (w *Writer) WriteEncodable(fieldNumber int, data Encodable) error {
	if data == nil {
		return nil
	}
	if err := w.writeKey(wireType2, fieldNumber); err != nil {
		return err
	}
	result, err := data.Encode()
	if err != nil {
		return err
	}
	return w.writeBytes(result)
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
func (w *Writer) writeKey(wireType int, fieldNumber int) error {
	vint, size := getKey(wireType, fieldNumber)
	w.size += size
	w.result = append(w.result, vint[0:size]...)
	return nil
}

// writeBytes writes a bytes to result without key.
func (w *Writer) writeBytes(data []byte) error {
	size := len(data)
	if err := w.writeUInt(uint64(size)); err != nil {
		return err
	}
	w.size += size
	w.result = append(w.result, data...)
	return nil
}

// writeUInt writes uint to result without key.
func (w *Writer) writeUInt(data uint64) error {
	vint := make([]byte, binary.MaxVarintLen64)
	size := binary.PutUvarint(vint, data)
	w.size = size
	w.result = append(w.result, vint[0:size]...)
	return nil
}

// writeInt writes int to result without key.
func (w *Writer) writeInt(data int64) error {
	vint := make([]byte, binary.MaxVarintLen64)
	size := binary.PutVarint(vint, data)
	w.size = size
	w.result = append(w.result, vint[0:size]...)
	return nil
}

// writeBool writes bool to result without key.
func (w *Writer) writeBool(data bool) error {
	b := make([]byte, 1)
	if data {
		b[0] = 0x01
	} else {
		b[0] = 0x00
	}
	w.size++
	w.result = append(w.result, b...)
	return nil
}

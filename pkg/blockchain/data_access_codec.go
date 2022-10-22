// Code generated by github.com/LiskHQ/lisk-engine/pkg/codec/gen; DO NOT EDIT.

package blockchain

import (
	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

func (e *bytesList) Encode() ([]byte, error) {
	writer := codec.NewWriter()
	if err := writer.WriteBytesArray(1, e.items); err != nil {
		return nil, err
	}
	return writer.Result(), nil
}

func (e *bytesList) MustEncode() []byte {
	encoded, err := e.Encode()
	if err != nil {
		panic(err)
	}
	return encoded
}

func (e *bytesList) Decode(data []byte) error {
	reader := codec.NewReader(data)
	return e.DecodeFromReader(reader)
}

func (e *bytesList) MustDecode(data []byte) {
	if err := e.Decode(data); err != nil {
		panic(err)
	}
}

func (e *bytesList) DecodeStrict(data []byte) error {
	reader := codec.NewReader(data)
	if err := e.DecodeStrictFromReader(reader); err != nil {
		return err
	}
	if reader.HasUnreadBytes() {
		return codec.ErrUnreadBytes
	}
	return nil
}

func (e *bytesList) DecodeFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytesArray(1)
		if err != nil {
			return err
		}
		e.items = val
	}
	return nil
}

func (e *bytesList) DecodeStrictFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytesArray(1)
		if err != nil {
			return err
		}
		e.items = val
	}
	return nil
}

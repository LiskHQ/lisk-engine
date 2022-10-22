// Code generated by github.com/LiskHQ/lisk-engine/pkg/codec/gen; DO NOT EDIT.

package sync

import (
	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

func (e *getHighestCommonBlockRequest) Encode() ([]byte, error) {
	writer := codec.NewWriter()
	if err := writer.WriteBytesArray(1, e.ids); err != nil {
		return nil, err
	}
	return writer.Result(), nil
}

func (e *getHighestCommonBlockRequest) MustEncode() []byte {
	encoded, err := e.Encode()
	if err != nil {
		panic(err)
	}
	return encoded
}

func (e *getHighestCommonBlockRequest) Decode(data []byte) error {
	reader := codec.NewReader(data)
	return e.DecodeFromReader(reader)
}

func (e *getHighestCommonBlockRequest) MustDecode(data []byte) {
	if err := e.Decode(data); err != nil {
		panic(err)
	}
}

func (e *getHighestCommonBlockRequest) DecodeStrict(data []byte) error {
	reader := codec.NewReader(data)
	if err := e.DecodeStrictFromReader(reader); err != nil {
		return err
	}
	if reader.HasUnreadBytes() {
		return codec.ErrUnreadBytes
	}
	return nil
}

func (e *getHighestCommonBlockRequest) DecodeFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytesArray(1)
		if err != nil {
			return err
		}
		e.ids = val
	}
	return nil
}

func (e *getHighestCommonBlockRequest) DecodeStrictFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytesArray(1)
		if err != nil {
			return err
		}
		e.ids = val
	}
	return nil
}

func (e *getHighestCommonBlockResponse) Encode() ([]byte, error) {
	writer := codec.NewWriter()
	if err := writer.WriteBytes(1, e.id); err != nil {
		return nil, err
	}
	return writer.Result(), nil
}

func (e *getHighestCommonBlockResponse) MustEncode() []byte {
	encoded, err := e.Encode()
	if err != nil {
		panic(err)
	}
	return encoded
}

func (e *getHighestCommonBlockResponse) Decode(data []byte) error {
	reader := codec.NewReader(data)
	return e.DecodeFromReader(reader)
}

func (e *getHighestCommonBlockResponse) MustDecode(data []byte) {
	if err := e.Decode(data); err != nil {
		panic(err)
	}
}

func (e *getHighestCommonBlockResponse) DecodeStrict(data []byte) error {
	reader := codec.NewReader(data)
	if err := e.DecodeStrictFromReader(reader); err != nil {
		return err
	}
	if reader.HasUnreadBytes() {
		return codec.ErrUnreadBytes
	}
	return nil
}

func (e *getHighestCommonBlockResponse) DecodeFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytes(1, false)
		if err != nil {
			return err
		}
		e.id = val
	}
	return nil
}

func (e *getHighestCommonBlockResponse) DecodeStrictFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytes(1, true)
		if err != nil {
			return err
		}
		e.id = val
	}
	return nil
}

func (e *getBlocksFromIDRequest) Encode() ([]byte, error) {
	writer := codec.NewWriter()
	if err := writer.WriteBytes(1, e.blockID); err != nil {
		return nil, err
	}
	return writer.Result(), nil
}

func (e *getBlocksFromIDRequest) MustEncode() []byte {
	encoded, err := e.Encode()
	if err != nil {
		panic(err)
	}
	return encoded
}

func (e *getBlocksFromIDRequest) Decode(data []byte) error {
	reader := codec.NewReader(data)
	return e.DecodeFromReader(reader)
}

func (e *getBlocksFromIDRequest) MustDecode(data []byte) {
	if err := e.Decode(data); err != nil {
		panic(err)
	}
}

func (e *getBlocksFromIDRequest) DecodeStrict(data []byte) error {
	reader := codec.NewReader(data)
	if err := e.DecodeStrictFromReader(reader); err != nil {
		return err
	}
	if reader.HasUnreadBytes() {
		return codec.ErrUnreadBytes
	}
	return nil
}

func (e *getBlocksFromIDRequest) DecodeFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytes(1, false)
		if err != nil {
			return err
		}
		e.blockID = val
	}
	return nil
}

func (e *getBlocksFromIDRequest) DecodeStrictFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytes(1, true)
		if err != nil {
			return err
		}
		e.blockID = val
	}
	return nil
}

func (e *getBlocksFromIDResponse) Encode() ([]byte, error) {
	writer := codec.NewWriter()
	if err := writer.WriteBytesArray(1, e.blocks); err != nil {
		return nil, err
	}
	return writer.Result(), nil
}

func (e *getBlocksFromIDResponse) MustEncode() []byte {
	encoded, err := e.Encode()
	if err != nil {
		panic(err)
	}
	return encoded
}

func (e *getBlocksFromIDResponse) Decode(data []byte) error {
	reader := codec.NewReader(data)
	return e.DecodeFromReader(reader)
}

func (e *getBlocksFromIDResponse) MustDecode(data []byte) {
	if err := e.Decode(data); err != nil {
		panic(err)
	}
}

func (e *getBlocksFromIDResponse) DecodeStrict(data []byte) error {
	reader := codec.NewReader(data)
	if err := e.DecodeStrictFromReader(reader); err != nil {
		return err
	}
	if reader.HasUnreadBytes() {
		return codec.ErrUnreadBytes
	}
	return nil
}

func (e *getBlocksFromIDResponse) DecodeFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytesArray(1)
		if err != nil {
			return err
		}
		e.blocks = val
	}
	return nil
}

func (e *getBlocksFromIDResponse) DecodeStrictFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytesArray(1)
		if err != nil {
			return err
		}
		e.blocks = val
	}
	return nil
}

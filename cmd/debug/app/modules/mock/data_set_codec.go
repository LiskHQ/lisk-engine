// Code generated by github.com/LiskHQ/lisk-engine/pkg/codec/gen; DO NOT EDIT.

package mock

import (
	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

func (e *Pair) Encode() ([]byte, error) {
	writer := codec.NewWriter()
	if err := writer.WriteBytes(1, e.Key); err != nil {
		return nil, err
	}
	if err := writer.WriteBytes(2, e.Value); err != nil {
		return nil, err
	}
	return writer.Result(), nil
}

func (e *Pair) MustEncode() []byte {
	encoded, err := e.Encode()
	if err != nil {
		panic(err)
	}
	return encoded
}

func (e *Pair) Decode(data []byte) error {
	reader := codec.NewReader(data)
	return e.DecodeFromReader(reader)
}

func (e *Pair) MustDecode(data []byte) {
	if err := e.Decode(data); err != nil {
		panic(err)
	}
}

func (e *Pair) DecodeStrict(data []byte) error {
	reader := codec.NewReader(data)
	if err := e.DecodeStrictFromReader(reader); err != nil {
		return err
	}
	if reader.HasUnreadBytes() {
		return codec.ErrUnreadBytes
	}
	return nil
}

func (e *Pair) DecodeFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytes(1, false)
		if err != nil {
			return err
		}
		e.Key = val
	}
	{
		val, err := reader.ReadBytes(2, false)
		if err != nil {
			return err
		}
		e.Value = val
	}
	return nil
}

func (e *Pair) DecodeStrictFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytes(1, true)
		if err != nil {
			return err
		}
		e.Key = val
	}
	{
		val, err := reader.ReadBytes(2, true)
		if err != nil {
			return err
		}
		e.Value = val
	}
	return nil
}

func (e *DataSetParams) Encode() ([]byte, error) {
	writer := codec.NewWriter()
	{
		for _, val := range e.Pairs {
			if val != nil {
				if err := writer.WriteEncodable(1, val); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := writer.WriteBool(2, e.Hash); err != nil {
		return nil, err
	}
	return writer.Result(), nil
}

func (e *DataSetParams) MustEncode() []byte {
	encoded, err := e.Encode()
	if err != nil {
		panic(err)
	}
	return encoded
}

func (e *DataSetParams) Decode(data []byte) error {
	reader := codec.NewReader(data)
	return e.DecodeFromReader(reader)
}

func (e *DataSetParams) MustDecode(data []byte) {
	if err := e.Decode(data); err != nil {
		panic(err)
	}
}

func (e *DataSetParams) DecodeStrict(data []byte) error {
	reader := codec.NewReader(data)
	if err := e.DecodeStrictFromReader(reader); err != nil {
		return err
	}
	if reader.HasUnreadBytes() {
		return codec.ErrUnreadBytes
	}
	return nil
}

func (e *DataSetParams) DecodeFromReader(reader *codec.Reader) error {
	{
		vals, err := reader.ReadDecodables(1, func() codec.DecodableReader { return new(Pair) })
		if err != nil {
			return err
		}
		r := make([]*Pair, len(vals))
		for i, v := range vals {
			r[i] = v.(*Pair)
		}
		e.Pairs = r
	}
	{
		val, err := reader.ReadBool(2, false)
		if err != nil {
			return err
		}
		e.Hash = val
	}
	return nil
}

func (e *DataSetParams) DecodeStrictFromReader(reader *codec.Reader) error {
	{
		vals, err := reader.ReadDecodables(1, func() codec.DecodableReader { return new(Pair) })
		if err != nil {
			return err
		}
		r := make([]*Pair, len(vals))
		for i, v := range vals {
			r[i] = v.(*Pair)
		}
		e.Pairs = r
	}
	{
		val, err := reader.ReadBool(2, true)
		if err != nil {
			return err
		}
		e.Hash = val
	}
	return nil
}
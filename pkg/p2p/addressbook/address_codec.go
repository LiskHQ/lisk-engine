// Code generated by github.com/LiskHQ/lisk-engine/pkg/codec/gen; DO NOT EDIT.

package addressbook

import (
	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

func (e *Addresses) Encode() ([]byte, error) {
	writer := codec.NewWriter()
	{
		for _, val := range e.Addresses {
			if val != nil {
				if err := writer.WriteEncodable(1, val); err != nil {
					return nil, err
				}
			}
		}
	}
	return writer.Result(), nil
}

func (e *Addresses) MustEncode() []byte {
	encoded, err := e.Encode()
	if err != nil {
		panic(err)
	}
	return encoded
}

func (e *Addresses) Decode(data []byte) error {
	reader := codec.NewReader(data)
	return e.DecodeFromReader(reader)
}

func (e *Addresses) MustDecode(data []byte) {
	if err := e.Decode(data); err != nil {
		panic(err)
	}
}

func (e *Addresses) DecodeStrict(data []byte) error {
	reader := codec.NewReader(data)
	if err := e.DecodeStrictFromReader(reader); err != nil {
		return err
	}
	if reader.HasUnreadBytes() {
		return codec.ErrUnreadBytes
	}
	return nil
}

func (e *Addresses) DecodeFromReader(reader *codec.Reader) error {
	{
		vals, err := reader.ReadDecodables(1, func() codec.DecodableReader { return new(Address) })
		if err != nil {
			return err
		}
		r := make([]*Address, len(vals))
		for i, v := range vals {
			r[i] = v.(*Address)
		}
		e.Addresses = r
	}
	return nil
}

func (e *Addresses) DecodeStrictFromReader(reader *codec.Reader) error {
	{
		vals, err := reader.ReadDecodables(1, func() codec.DecodableReader { return new(Address) })
		if err != nil {
			return err
		}
		r := make([]*Address, len(vals))
		for i, v := range vals {
			r[i] = v.(*Address)
		}
		e.Addresses = r
	}
	return nil
}

func (e *Address) Encode() ([]byte, error) {
	writer := codec.NewWriter()
	if err := writer.WriteString(1, e.ip); err != nil {
		return nil, err
	}
	if err := writer.WriteUInt32(2, e.port); err != nil {
		return nil, err
	}
	return writer.Result(), nil
}

func (e *Address) MustEncode() []byte {
	encoded, err := e.Encode()
	if err != nil {
		panic(err)
	}
	return encoded
}

func (e *Address) Decode(data []byte) error {
	reader := codec.NewReader(data)
	return e.DecodeFromReader(reader)
}

func (e *Address) MustDecode(data []byte) {
	if err := e.Decode(data); err != nil {
		panic(err)
	}
}

func (e *Address) DecodeStrict(data []byte) error {
	reader := codec.NewReader(data)
	if err := e.DecodeStrictFromReader(reader); err != nil {
		return err
	}
	if reader.HasUnreadBytes() {
		return codec.ErrUnreadBytes
	}
	return nil
}

func (e *Address) DecodeFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadString(1, false)
		if err != nil {
			return err
		}
		e.ip = val
	}
	{
		val, err := reader.ReadUInt32(2, false)
		if err != nil {
			return err
		}
		e.port = val
	}
	return nil
}

func (e *Address) DecodeStrictFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadString(1, true)
		if err != nil {
			return err
		}
		e.ip = val
	}
	{
		val, err := reader.ReadUInt32(2, true)
		if err != nil {
			return err
		}
		e.port = val
	}
	return nil
}

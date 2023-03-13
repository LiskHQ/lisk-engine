// Code generated by github.com/LiskHQ/lisk-engine/pkg/codec/gen; DO NOT EDIT.

package mock

import (
	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

func (e *UpdateValidatorsParams) Encode() ([]byte, error) {
	writer := codec.NewWriter()
	{
		for _, val := range e.Keys {
			if val != nil {
				if err := writer.WriteEncodable(1, val); err != nil {
					return nil, err
				}
			}
		}
	}
	return writer.Result(), nil
}

func (e *UpdateValidatorsParams) MustEncode() []byte {
	encoded, err := e.Encode()
	if err != nil {
		panic(err)
	}
	return encoded
}

func (e *UpdateValidatorsParams) Decode(data []byte) error {
	reader := codec.NewReader(data)
	return e.DecodeFromReader(reader)
}

func (e *UpdateValidatorsParams) MustDecode(data []byte) {
	if err := e.Decode(data); err != nil {
		panic(err)
	}
}

func (e *UpdateValidatorsParams) DecodeStrict(data []byte) error {
	reader := codec.NewReader(data)
	if err := e.DecodeStrictFromReader(reader); err != nil {
		return err
	}
	if reader.HasUnreadBytes() {
		return codec.ErrUnreadBytes
	}
	return nil
}

func (e *UpdateValidatorsParams) DecodeFromReader(reader *codec.Reader) error {
	{
		vals, err := reader.ReadDecodables(1, func() codec.DecodableReader { return new(ValidatorKey) })
		if err != nil {
			return err
		}
		r := make([]*ValidatorKey, len(vals))
		for i, v := range vals {
			r[i] = v.(*ValidatorKey)
		}
		e.Keys = r
	}
	return nil
}

func (e *UpdateValidatorsParams) DecodeStrictFromReader(reader *codec.Reader) error {
	{
		vals, err := reader.ReadDecodables(1, func() codec.DecodableReader { return new(ValidatorKey) })
		if err != nil {
			return err
		}
		r := make([]*ValidatorKey, len(vals))
		for i, v := range vals {
			r[i] = v.(*ValidatorKey)
		}
		e.Keys = r
	}
	return nil
}
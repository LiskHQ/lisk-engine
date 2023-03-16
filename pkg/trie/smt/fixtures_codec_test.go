// Code generated by github.com/LiskHQ/lisk-engine/pkg/codec/gen; DO NOT EDIT.

package smt

import (
	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

func (e *fixtureTestCase) Encode() []byte {
	writer := codec.NewWriter()
	writer.WriteString(1, e.Description)
	if e.Input != nil {
		writer.WriteEncodable(2, e.Input)
	}
	if e.Output != nil {
		writer.WriteEncodable(3, e.Output)
	}
	return writer.Result()
}

func (e *fixtureTestCase) Decode(data []byte) error {
	reader := codec.NewReader(data)
	return e.DecodeFromReader(reader)
}

func (e *fixtureTestCase) MustDecode(data []byte) {
	if err := e.Decode(data); err != nil {
		panic(err)
	}
}

func (e *fixtureTestCase) DecodeStrict(data []byte) error {
	reader := codec.NewReader(data)
	if err := e.DecodeStrictFromReader(reader); err != nil {
		return err
	}
	if reader.HasUnreadBytes() {
		return codec.ErrUnreadBytes
	}
	return nil
}

func (e *fixtureTestCase) DecodeFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadString(1, false)
		if err != nil {
			return err
		}
		e.Description = val
	}
	{
		val, err := reader.ReadDecodable(2, func() codec.DecodableReader { return new(fixtureInput) }, false)
		if err != nil {
			return err
		}
		e.Input = val.(*fixtureInput)
	}
	{
		val, err := reader.ReadDecodable(3, func() codec.DecodableReader { return new(fixtureOutput) }, false)
		if err != nil {
			return err
		}
		e.Output = val.(*fixtureOutput)
	}
	return nil
}

func (e *fixtureTestCase) DecodeStrictFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadString(1, true)
		if err != nil {
			return err
		}
		e.Description = val
	}
	{
		val, err := reader.ReadDecodable(2, func() codec.DecodableReader { return new(fixtureInput) }, true)
		if err != nil {
			return err
		}
		e.Input = val.(*fixtureInput)
	}
	{
		val, err := reader.ReadDecodable(3, func() codec.DecodableReader { return new(fixtureOutput) }, true)
		if err != nil {
			return err
		}
		e.Output = val.(*fixtureOutput)
	}
	return nil
}

func (e *fixtureInput) Encode() []byte {
	writer := codec.NewWriter()
	writer.WriteBytesArray(1, codec.HexArrayToBytesArray(e.Keys))
	writer.WriteBytesArray(2, codec.HexArrayToBytesArray(e.Values))
	writer.WriteBytesArray(3, codec.HexArrayToBytesArray(e.DeleteKeys))
	writer.WriteBytesArray(4, codec.HexArrayToBytesArray(e.QueryKeys))
	return writer.Result()
}

func (e *fixtureInput) Decode(data []byte) error {
	reader := codec.NewReader(data)
	return e.DecodeFromReader(reader)
}

func (e *fixtureInput) MustDecode(data []byte) {
	if err := e.Decode(data); err != nil {
		panic(err)
	}
}

func (e *fixtureInput) DecodeStrict(data []byte) error {
	reader := codec.NewReader(data)
	if err := e.DecodeStrictFromReader(reader); err != nil {
		return err
	}
	if reader.HasUnreadBytes() {
		return codec.ErrUnreadBytes
	}
	return nil
}

func (e *fixtureInput) DecodeFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytesArray(1)
		if err != nil {
			return err
		}
		e.Keys = codec.BytesArrayToHexArray(val)
	}
	{
		val, err := reader.ReadBytesArray(2)
		if err != nil {
			return err
		}
		e.Values = codec.BytesArrayToHexArray(val)
	}
	{
		val, err := reader.ReadBytesArray(3)
		if err != nil {
			return err
		}
		e.DeleteKeys = codec.BytesArrayToHexArray(val)
	}
	{
		val, err := reader.ReadBytesArray(4)
		if err != nil {
			return err
		}
		e.QueryKeys = codec.BytesArrayToHexArray(val)
	}
	return nil
}

func (e *fixtureInput) DecodeStrictFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytesArray(1)
		if err != nil {
			return err
		}
		e.Keys = codec.BytesArrayToHexArray(val)
	}
	{
		val, err := reader.ReadBytesArray(2)
		if err != nil {
			return err
		}
		e.Values = codec.BytesArrayToHexArray(val)
	}
	{
		val, err := reader.ReadBytesArray(3)
		if err != nil {
			return err
		}
		e.DeleteKeys = codec.BytesArrayToHexArray(val)
	}
	{
		val, err := reader.ReadBytesArray(4)
		if err != nil {
			return err
		}
		e.QueryKeys = codec.BytesArrayToHexArray(val)
	}
	return nil
}

func (e *fixtureOutput) Encode() []byte {
	writer := codec.NewWriter()
	writer.WriteBytes(1, e.MerkleRoot)
	if e.Proof != nil {
		writer.WriteEncodable(2, e.Proof)
	}
	return writer.Result()
}

func (e *fixtureOutput) Decode(data []byte) error {
	reader := codec.NewReader(data)
	return e.DecodeFromReader(reader)
}

func (e *fixtureOutput) MustDecode(data []byte) {
	if err := e.Decode(data); err != nil {
		panic(err)
	}
}

func (e *fixtureOutput) DecodeStrict(data []byte) error {
	reader := codec.NewReader(data)
	if err := e.DecodeStrictFromReader(reader); err != nil {
		return err
	}
	if reader.HasUnreadBytes() {
		return codec.ErrUnreadBytes
	}
	return nil
}

func (e *fixtureOutput) DecodeFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytes(1, false)
		if err != nil {
			return err
		}
		e.MerkleRoot = val
	}
	{
		val, err := reader.ReadDecodable(2, func() codec.DecodableReader { return new(fixtureOutputProof) }, false)
		if err != nil {
			return err
		}
		e.Proof = val.(*fixtureOutputProof)
	}
	return nil
}

func (e *fixtureOutput) DecodeStrictFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytes(1, true)
		if err != nil {
			return err
		}
		e.MerkleRoot = val
	}
	{
		val, err := reader.ReadDecodable(2, func() codec.DecodableReader { return new(fixtureOutputProof) }, true)
		if err != nil {
			return err
		}
		e.Proof = val.(*fixtureOutputProof)
	}
	return nil
}

func (e *fixtureOutputProofQuery) Encode() []byte {
	writer := codec.NewWriter()
	writer.WriteBytes(1, e.Bitmap)
	writer.WriteBytes(2, e.Key)
	writer.WriteBytes(3, e.Value)
	return writer.Result()
}

func (e *fixtureOutputProofQuery) Decode(data []byte) error {
	reader := codec.NewReader(data)
	return e.DecodeFromReader(reader)
}

func (e *fixtureOutputProofQuery) MustDecode(data []byte) {
	if err := e.Decode(data); err != nil {
		panic(err)
	}
}

func (e *fixtureOutputProofQuery) DecodeStrict(data []byte) error {
	reader := codec.NewReader(data)
	if err := e.DecodeStrictFromReader(reader); err != nil {
		return err
	}
	if reader.HasUnreadBytes() {
		return codec.ErrUnreadBytes
	}
	return nil
}

func (e *fixtureOutputProofQuery) DecodeFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytes(1, false)
		if err != nil {
			return err
		}
		e.Bitmap = val
	}
	{
		val, err := reader.ReadBytes(2, false)
		if err != nil {
			return err
		}
		e.Key = val
	}
	{
		val, err := reader.ReadBytes(3, false)
		if err != nil {
			return err
		}
		e.Value = val
	}
	return nil
}

func (e *fixtureOutputProofQuery) DecodeStrictFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytes(1, true)
		if err != nil {
			return err
		}
		e.Bitmap = val
	}
	{
		val, err := reader.ReadBytes(2, true)
		if err != nil {
			return err
		}
		e.Key = val
	}
	{
		val, err := reader.ReadBytes(3, true)
		if err != nil {
			return err
		}
		e.Value = val
	}
	return nil
}

func (e *fixtureOutputProof) Encode() []byte {
	writer := codec.NewWriter()
	writer.WriteBytesArray(1, codec.HexArrayToBytesArray(e.SiblingHashes))
	{
		for _, val := range e.Queries {
			if val != nil {
				writer.WriteEncodable(2, val)
			}
		}
	}
	return writer.Result()
}

func (e *fixtureOutputProof) Decode(data []byte) error {
	reader := codec.NewReader(data)
	return e.DecodeFromReader(reader)
}

func (e *fixtureOutputProof) MustDecode(data []byte) {
	if err := e.Decode(data); err != nil {
		panic(err)
	}
}

func (e *fixtureOutputProof) DecodeStrict(data []byte) error {
	reader := codec.NewReader(data)
	if err := e.DecodeStrictFromReader(reader); err != nil {
		return err
	}
	if reader.HasUnreadBytes() {
		return codec.ErrUnreadBytes
	}
	return nil
}

func (e *fixtureOutputProof) DecodeFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytesArray(1)
		if err != nil {
			return err
		}
		e.SiblingHashes = codec.BytesArrayToHexArray(val)
	}
	{
		vals, err := reader.ReadDecodables(2, func() codec.DecodableReader { return new(fixtureOutputProofQuery) })
		if err != nil {
			return err
		}
		r := make([]*fixtureOutputProofQuery, len(vals))
		for i, v := range vals {
			r[i] = v.(*fixtureOutputProofQuery)
		}
		e.Queries = r
	}
	return nil
}

func (e *fixtureOutputProof) DecodeStrictFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadBytesArray(1)
		if err != nil {
			return err
		}
		e.SiblingHashes = codec.BytesArrayToHexArray(val)
	}
	{
		vals, err := reader.ReadDecodables(2, func() codec.DecodableReader { return new(fixtureOutputProofQuery) })
		if err != nil {
			return err
		}
		r := make([]*fixtureOutputProofQuery, len(vals))
		for i, v := range vals {
			r[i] = v.(*fixtureOutputProofQuery)
		}
		e.Queries = r
	}
	return nil
}

func (e *fixture) Encode() []byte {
	writer := codec.NewWriter()
	writer.WriteString(1, e.Title)
	writer.WriteString(2, e.Summary)
	{
		for _, val := range e.TestCases {
			if val != nil {
				writer.WriteEncodable(3, val)
			}
		}
	}
	return writer.Result()
}

func (e *fixture) Decode(data []byte) error {
	reader := codec.NewReader(data)
	return e.DecodeFromReader(reader)
}

func (e *fixture) MustDecode(data []byte) {
	if err := e.Decode(data); err != nil {
		panic(err)
	}
}

func (e *fixture) DecodeStrict(data []byte) error {
	reader := codec.NewReader(data)
	if err := e.DecodeStrictFromReader(reader); err != nil {
		return err
	}
	if reader.HasUnreadBytes() {
		return codec.ErrUnreadBytes
	}
	return nil
}

func (e *fixture) DecodeFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadString(1, false)
		if err != nil {
			return err
		}
		e.Title = val
	}
	{
		val, err := reader.ReadString(2, false)
		if err != nil {
			return err
		}
		e.Summary = val
	}
	{
		vals, err := reader.ReadDecodables(3, func() codec.DecodableReader { return new(fixtureTestCase) })
		if err != nil {
			return err
		}
		r := make([]*fixtureTestCase, len(vals))
		for i, v := range vals {
			r[i] = v.(*fixtureTestCase)
		}
		e.TestCases = r
	}
	return nil
}

func (e *fixture) DecodeStrictFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadString(1, true)
		if err != nil {
			return err
		}
		e.Title = val
	}
	{
		val, err := reader.ReadString(2, true)
		if err != nil {
			return err
		}
		e.Summary = val
	}
	{
		vals, err := reader.ReadDecodables(3, func() codec.DecodableReader { return new(fixtureTestCase) })
		if err != nil {
			return err
		}
		r := make([]*fixtureTestCase, len(vals))
		for i, v := range vals {
			r[i] = v.(*fixtureTestCase)
		}
		e.TestCases = r
	}
	return nil
}

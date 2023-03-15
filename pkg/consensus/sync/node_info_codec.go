// Code generated by github.com/LiskHQ/lisk-engine/pkg/codec/gen; DO NOT EDIT.

package sync

import (
	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

func (e *NodeInfo) Encode() []byte {
	writer := codec.NewWriter()
	writer.WriteUInt32(1, e.height)
	writer.WriteUInt32(2, e.maxHeightPrevoted)
	writer.WriteUInt32(3, e.blockVersion)
	writer.WriteBytes(4, e.lastBlockID)
	return writer.Result()
}

func (e *NodeInfo) Decode(data []byte) error {
	reader := codec.NewReader(data)
	return e.DecodeFromReader(reader)
}

func (e *NodeInfo) MustDecode(data []byte) {
	if err := e.Decode(data); err != nil {
		panic(err)
	}
}

func (e *NodeInfo) DecodeStrict(data []byte) error {
	reader := codec.NewReader(data)
	if err := e.DecodeStrictFromReader(reader); err != nil {
		return err
	}
	if reader.HasUnreadBytes() {
		return codec.ErrUnreadBytes
	}
	return nil
}

func (e *NodeInfo) DecodeFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadUInt32(1, false)
		if err != nil {
			return err
		}
		e.height = val
	}
	{
		val, err := reader.ReadUInt32(2, false)
		if err != nil {
			return err
		}
		e.maxHeightPrevoted = val
	}
	{
		val, err := reader.ReadUInt32(3, false)
		if err != nil {
			return err
		}
		e.blockVersion = val
	}
	{
		val, err := reader.ReadBytes(4, false)
		if err != nil {
			return err
		}
		e.lastBlockID = val
	}
	return nil
}

func (e *NodeInfo) DecodeStrictFromReader(reader *codec.Reader) error {
	{
		val, err := reader.ReadUInt32(1, true)
		if err != nil {
			return err
		}
		e.height = val
	}
	{
		val, err := reader.ReadUInt32(2, true)
		if err != nil {
			return err
		}
		e.maxHeightPrevoted = val
	}
	{
		val, err := reader.ReadUInt32(3, true)
		if err != nil {
			return err
		}
		e.blockVersion = val
	}
	{
		val, err := reader.ReadBytes(4, true)
		if err != nil {
			return err
		}
		e.lastBlockID = val
	}
	return nil
}

package rpc

import (
	"encoding/json"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

type EventContent interface {
	Event() string
	Data() codec.EncodeDecodable
	JSONData() ([]byte, error)
	CodecData() ([]byte, error)
}

func NewEventContent(event string, data codec.EncodeDecodable) EventContent {
	return &eventContent{
		event: event,
		data:  data,
	}
}

type eventContent struct {
	event string
	data  codec.EncodeDecodable
}

func (e *eventContent) Event() string {
	return e.event
}

func (e *eventContent) Data() codec.EncodeDecodable {
	return e.data
}

func (e *eventContent) JSONData() ([]byte, error) {
	return json.Marshal(e.data)
}

func (e *eventContent) CodecData() ([]byte, error) {
	return e.data.Encode()
}

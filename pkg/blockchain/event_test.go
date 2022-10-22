package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

func createRandomEvent(height, index uint32) *Event {
	return NewEventFromValues(
		codec.Hex(crypto.RandomBytes(1)).String(),
		codec.Hex(crypto.RandomBytes(1)).String(),
		crypto.RandomBytes(int(EventMaxSizeBytes)),
		[]codec.Hex{crypto.RandomBytes(32), crypto.RandomBytes(100), crypto.RandomBytes(2), crypto.RandomBytes(34)},
		height,
		index,
	)
}

func TestEventValidate(t *testing.T) {
	event := createRandomEvent(5, 0)
	event.Data = crypto.RandomBytes(int(EventMaxSizeBytes) + 1)
	assert.EqualError(t,
		event.Validate(),
		"event exceeds max data size 1024")

	event = createRandomEvent(5, 0)
	event.Topics = append(event.Topics, crypto.RandomBytes(2))
	assert.EqualError(t,
		event.Validate(),
		"event exceeds max topics length 4")

	event = createRandomEvent(5, 0)
	event.Topics = []codec.Hex{}
	assert.EqualError(t,
		event.Validate(),
		"topics must have at least one element")
}

func TestEventKeyPairs(t *testing.T) {
	event := createRandomEvent(5, 0)
	keyPairs := event.KeyPairs()
	assert.Len(t, keyPairs, len(event.Topics))
	keys := make([][]byte, len(keyPairs))
	values := make([][]byte, len(keyPairs))
	for i, kv := range keyPairs {
		keys[i] = kv.Key
		values[i] = kv.Value
	}
	assert.Len(t, bytes.Unique(values), 1)
	assert.Len(t, bytes.Unique(keys), len(event.Topics))
}

func TestEventRoot(t *testing.T) {
	event := createRandomEvent(5, 0)
	root, err := CalculateEventRoot([]*Event{event})
	assert.NoError(t, err)
	assert.NotEqual(t, emptyHash, root)
}

func TestEvents_UpdateIndex(t *testing.T) {
	events := Events{
		{
			Module: "sample",
			Name:   "new",
			Data:   crypto.RandomBytes(20),
			Topics: []codec.Hex{},
			Height: 3,
			Index:  0,
		},
		{
			Module: "sample",
			Name:   "new",
			Data:   crypto.RandomBytes(20),
			Topics: []codec.Hex{},
			Height: 3,
			Index:  0,
		},
		{
			Module: "sample",
			Name:   "new",
			Data:   crypto.RandomBytes(20),
			Topics: []codec.Hex{},
			Height: 3,
			Index:  1,
		},
	}
	for _, event := range events {
		event.UpdateID()
	}
	originalID := bytes.Copy(events[2].ID)

	events.UpdateIndex()

	assert.NotEqual(t, originalID, events[2].ID)
	for i, event := range events {
		assert.Equal(t, uint32(i), event.Index)
	}
}

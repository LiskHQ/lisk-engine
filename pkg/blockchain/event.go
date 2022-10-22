package blockchain

import (
	"errors"
	"fmt"
	"math"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/LiskHQ/lisk-engine/pkg/db"
	"github.com/LiskHQ/lisk-engine/pkg/trie/smt"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

const (
	eventTopicHashLengthBytes  uint32 = 8
	eventIndexLengthBits       uint32 = 30
	eventTopicIndexLengthBits  uint32 = 2
	EventMaxSizeBytes          uint32 = 1024
	EventTotalIndexLengthBytes uint32 = (eventIndexLengthBits + eventTopicIndexLengthBits) / 8
	EventIDLengthBytes         uint32 = 4 + EventTotalIndexLengthBytes
	binaryBase                        = 2
)

var (
	EventNameDefault       = "commandExecutionResult"
	EventMaxTopicsPerEvent = uint32(math.Pow(binaryBase, float64(eventTopicIndexLengthBits)))
	MaxEventsPerBlock      = uint32(math.Pow(binaryBase, float64(eventIndexLengthBits)))
)

func NewEvent(val []byte) (*Event, error) {
	e := &Event{}
	if err := e.Decode(val); err != nil {
		return nil, err
	}
	return e, nil
}

func NewEventFromValues(module string, name string, data []byte, topics []codec.Hex, height, index uint32) *Event {
	e := &Event{
		Module: module,
		Name:   name,
		Data:   data,
		Topics: topics,
		Height: height,
		Index:  index,
	}
	return e
}

type StandardTransactionEvent struct {
	Success bool `fieldNumber:"1"`
}

func NewStandardTransactionEventData(success bool) []byte {
	e := &StandardTransactionEvent{
		Success: success,
	}
	return e.MustEncode()
}

type Event struct {
	ID     codec.Hex   `json:"id"`
	Module string      `fieldNumber:"1" json:"module"`
	Name   string      `fieldNumber:"2" json:"name"`
	Data   codec.Hex   `fieldNumber:"3" json:"data"`
	Topics []codec.Hex `fieldNumber:"4" json:"topics"`
	Height uint32      `fieldNumber:"5" json:"height"`
	Index  uint32      `fieldNumber:"6" json:"index"`
}

type EventKeyPair struct {
	Key   codec.Hex
	Value codec.Hex
}

func (e *Event) UpdateID() []byte {
	heightBytes := bytes.FromUint32(e.Height)
	idBytes := bytes.FromUint32(e.Index << eventTopicIndexLengthBits)
	return bytes.JoinSize(int(EventTotalIndexLengthBytes), heightBytes, idBytes)
}

func (e *Event) KeyPairs() []EventKeyPair {
	result := make([]EventKeyPair, len(e.Topics))
	value := e.MustEncode()
	for i, topic := range e.Topics {
		indexBytes := bytes.FromUint32((e.Index << eventTopicIndexLengthBits) + uint32(i))
		topicBytes := crypto.Hash(topic)[:eventTopicHashLengthBytes]
		key := bytes.Join(topicBytes, indexBytes)
		result[i] = EventKeyPair{
			Key:   key,
			Value: value,
		}
	}
	return result
}

func (e *Event) Validate() error {
	if !alphanumericRegex.MatchString(e.Module) {
		return fmt.Errorf("module %s does not satisfy alphanumeric", e.Module)
	}
	if !alphanumericRegex.MatchString(e.Name) {
		return fmt.Errorf("name %s does not satisfy alphanumeric", e.Name)
	}
	if len(e.Data) > int(EventMaxSizeBytes) {
		return fmt.Errorf("event exceeds max data size %d", EventMaxSizeBytes)
	}
	if len(e.Topics) == 0 {
		return errors.New("topics must have at least one element")
	}
	if len(e.Topics) > int(EventMaxTopicsPerEvent) {
		return fmt.Errorf("event exceeds max topics length %d", EventMaxTopicsPerEvent)
	}
	return nil
}

type Events []*Event

func (e *Events) UpdateIndex() {
	for i, event := range *e {
		event.Index = uint32(i)
		event.UpdateID()
	}
}

func CalculateEventRoot(events []*Event) ([]byte, error) {
	inmemoryDB, err := db.NewInMemoryDB()
	if err != nil {
		return nil, err
	}
	tree := smt.NewTrie(emptyHash, int(eventTopicHashLengthBytes+EventTotalIndexLengthBytes))
	keys := [][]byte{}
	values := [][]byte{}
	for _, event := range events {
		for _, kv := range event.KeyPairs() {
			keys = append(keys, kv.Key)
			values = append(values, kv.Value)
		}
	}

	root, err := tree.Update(inmemoryDB, keys, values)
	if err != nil {
		return nil, err
	}
	return root, nil
}

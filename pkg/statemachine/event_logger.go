package statemachine

import (
	"errors"

	"github.com/LiskHQ/lisk-engine/pkg/blockchain"
	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

var (
	EventTopicInitGenesisState          = []byte{0}
	EventTopicFinalizeGenesisState      = []byte{1}
	EventTopicBeforeTransactionsExecute = []byte{2}
	EventTopicAfterTransactionsExecute  = []byte{3}
)

type loggedEvent struct {
	event    *blockchain.Event
	noRevert bool
}

type EventLogger struct {
	events        []loggedEvent
	snapshotIndex int
	height        uint32
	defaultTopic  []byte
}

func NewEventLogger(height uint32) *EventLogger {
	return &EventLogger{
		events:        []loggedEvent{},
		height:        height,
		snapshotIndex: -1,
	}
}

func (l *EventLogger) Add(module, name string, data []byte, topics []codec.Hex) error {
	event, err := l.createEvent(module, name, data, topics)
	if err != nil {
		return err
	}
	l.events = append(l.events, loggedEvent{
		event:    event,
		noRevert: true,
	})
	return nil
}

func (l *EventLogger) AddUnrevertible(module, name string, data []byte, topics []codec.Hex) error {
	event, err := l.createEvent(module, name, data, topics)
	if err != nil {
		return err
	}
	l.events = append(l.events, loggedEvent{
		event:    event,
		noRevert: true,
	})
	return nil
}

func (l *EventLogger) createEvent(module, name string, data []byte, topics []codec.Hex) (*blockchain.Event, error) {
	if l.defaultTopic == nil {
		return nil, errors.New("default topic must be set before adding a new event")
	}
	topicsWithDefault := []codec.Hex{l.defaultTopic}
	if len(topics) > 0 {
		topicsWithDefault = append(topicsWithDefault, topics...)
	}
	event := blockchain.NewEventFromValues(module, name, data, topicsWithDefault, l.height, uint32(len(l.events)))
	if err := event.Validate(); err != nil {
		return nil, err
	}
	return event, nil
}

func (l *EventLogger) SetDefaultTopic(topic []byte) {
	l.defaultTopic = topic
}

func (l *EventLogger) CreateSnapshot() {
	l.snapshotIndex = len(l.events)
}

func (l *EventLogger) RestoreSnapshot() {
	if l.snapshotIndex < 0 {
		return
	}
	existingEvents, newEvents := l.events[:l.snapshotIndex], l.events[l.snapshotIndex:]
	noRevertingEvents := []loggedEvent{}
	for _, event := range newEvents {
		if event.noRevert {
			event.event.Index = uint32(l.snapshotIndex + len(noRevertingEvents))
			noRevertingEvents = append(noRevertingEvents, event)
		}
	}
	existingEvents = append(existingEvents, noRevertingEvents...)
	l.events = existingEvents
	l.snapshotIndex = -1
}

func (l *EventLogger) Events() []*blockchain.Event {
	events := make([]*blockchain.Event, len(l.events))
	for i, loggedEvent := range l.events {
		events[i] = loggedEvent.event
	}
	return events
}

// Package event implements Pub-Sub using channel.
package event

import (
	"errors"
	"sync"
)

var (
	ErrEventNotFound = errors.New("event not found")
)

type EventEmitter struct {
	events  map[string][]chan interface{}
	rwMutex *sync.RWMutex
}

func New() *EventEmitter {
	return &EventEmitter{
		events:  make(map[string][]chan interface{}),
		rwMutex: &sync.RWMutex{},
	}
}

func (ee *EventEmitter) On(event string, out chan interface{}) {
	ee.rwMutex.Lock()
	ee.events[event] = append(ee.events[event], out)
	ee.rwMutex.Unlock()
}

func (ee *EventEmitter) Subscribe(event string) chan interface{} {
	ee.rwMutex.Lock()
	out := make(chan interface{})
	ee.events[event] = append(ee.events[event], out)
	ee.rwMutex.Unlock()
	return out
}

func (ee *EventEmitter) Publish(event string, message interface{}) {
	ee.rwMutex.Lock()
	defer ee.rwMutex.Unlock()
	outs, exist := ee.events[event]
	if !exist {
		return
	}
	for _, out := range outs {
		out <- message
	}
}

func (ee *EventEmitter) Emit(event string, message interface{}) {
	ee.rwMutex.Lock()
	defer ee.rwMutex.Unlock()
	outs, exist := ee.events[event]
	if !exist {
		return
	}
	for _, out := range outs {
		out <- message
	}
}

func (ee *EventEmitter) Close() error {
	ee.rwMutex.Lock()
	defer ee.rwMutex.Unlock()
	for event, outs := range ee.events {
		for _, out := range outs {
			close(out)
		}
		delete(ee.events, event)
	}
	return nil
}

func (ee *EventEmitter) UnsubscribeAll(event string) error {
	ee.rwMutex.Lock()
	defer ee.rwMutex.Unlock()
	outs, exist := ee.events[event]
	if !exist {
		return ErrEventNotFound
	}
	for _, out := range outs {
		close(out)
	}
	delete(ee.events, event)
	return nil
}

func (ee *EventEmitter) Unsubscribe(event string, deletingOut chan interface{}) error {
	ee.rwMutex.Lock()
	defer ee.rwMutex.Unlock()
	outs, exist := ee.events[event]
	if !exist {
		return ErrEventNotFound
	}
	newOuts := []chan interface{}{}
	for _, out := range outs {
		if out == deletingOut {
			close(out)
		} else {
			newOuts = append(newOuts, out)
		}
	}
	ee.events[event] = newOuts
	return nil
}

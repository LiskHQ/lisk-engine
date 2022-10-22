package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testData struct {
	data string
}

func TestEventSubscribe(t *testing.T) {
	emitter := New()
	defer emitter.Close()

	receiver := emitter.Subscribe("block_received")
	wait := make(chan testData)

	go func() {
		received := <-receiver
		converted, ok := received.(testData)
		assert.Equal(t, true, ok)
		wait <- converted
	}()
	emitter.Publish("block_received", testData{data: "test"})

	result := <-wait
	assert.Equal(t, "test", result.data)

	emitter.Unsubscribe("block_received", receiver)
	emitter.Publish("block_received", testData{data: "test"})
	assert.Len(t, emitter.events["block_received"], 0)
}

func TestEventOn(t *testing.T) {
	emitter := New()
	defer emitter.Close()

	wait := make(chan interface{})
	emitter.On("block_received", wait)

	go func() {
		emitter.Emit("block_received", testData{data: "test-on"})
	}()

	result := <-wait
	converted, ok := result.(testData)
	assert.Equal(t, true, ok)
	assert.Equal(t, "test-on", converted.data)

	err := emitter.UnsubscribeAll("block_received")
	emitter.Emit("block_received", testData{data: "test-on"})
	assert.NoError(t, err)

	assert.Len(t, emitter.events["block_received"], 0)
}

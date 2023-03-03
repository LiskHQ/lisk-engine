package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvent_NewEvent(t *testing.T) {
	assert := assert.New(t)

	e := NewEvent(testPeerID, testEvent, []byte(testData))
	assert.Equal(testPeerID, e.peerID)
	assert.Equal(testEvent, e.topic)
	assert.Equal([]byte(testData), e.data)
}

func TestEvent_Getters(t *testing.T) {
	assert := assert.New(t)

	e := NewEvent(testPeerID, testEvent, []byte(testData))
	assert.Equal(testPeerID, e.PeerID())
	assert.Equal(testEvent, e.Topic())
	assert.Equal([]byte(testData), e.Data())
}

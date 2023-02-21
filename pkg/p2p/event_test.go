package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvent_NewEvent(t *testing.T) {
	assert := assert.New(t)

	e := newEvent(testPeerID, testEvent, []byte(testData))
	assert.Equal(testPeerID, e.peerID)
	assert.Equal(testEvent, e.event)
	assert.Equal([]byte(testData), e.data)
}

func TestEvent_Getters(t *testing.T) {
	assert := assert.New(t)

	e := newEvent(testPeerID, testEvent, []byte(testData))
	assert.Equal(testPeerID, e.PeerID())
	assert.Equal(testEvent, e.Event())
	assert.Equal([]byte(testData), e.Data())
}
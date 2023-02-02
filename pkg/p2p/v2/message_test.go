package p2p

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage_NewRequestMessage(t *testing.T) {
	assert := assert.New(t)

	msg := newRequestMessage("testPeerID", "knownPeers", []byte("test request data"))
	assert.NotNil(msg)
	assert.NotEmpty(msg.ID)
	assert.NotEmpty(msg.Timestamp)
	assert.Equal("7YHPeiMNWetQV9", msg.PeerID)
	assert.Equal("knownPeers", msg.Procedure)
	assert.Equal([]byte("test request data"), msg.Data)
}

func TestMessage_NewResponseMessage(t *testing.T) {
	assert := assert.New(t)

	msg := newResponseMessage("123456789", []byte("test response data"), errors.New("test error"))
	assert.NotNil(msg)
	assert.NotEmpty(msg.ID)
	assert.NotEmpty(msg.Timestamp)
	assert.Equal([]byte("test response data"), msg.Data)
	assert.Equal("test error", msg.Error)
}

func TestMessage_NewMessage(t *testing.T) {
	assert := assert.New(t)

	msg := newMessage([]byte("test data"))
	assert.NotNil(msg)
	assert.NotEmpty(msg.Timestamp)
	assert.Equal([]byte("test data"), msg.Data)
}

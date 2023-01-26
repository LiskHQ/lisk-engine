package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage_NewRequestMessage(t *testing.T) {
	assert := assert.New(t)

	msg := newRequestMessage("testPeerID", MessageRequestTypeKnownPeers, []byte("test request data"))
	assert.NotNil(msg)
	assert.NotEmpty(msg.ID)
	assert.NotEmpty(msg.Timestamp)
	assert.Equal("7YHPeiMNWetQV9", msg.PeerID)
	assert.Equal(string(MessageRequestTypeKnownPeers), msg.Procedure)
	assert.Equal([]byte("test request data"), msg.Data)
}

func TestMessage_NewResponseMessage(t *testing.T) {
	assert := assert.New(t)

	msg := newResponseMessage("testPeerID", "123456789", []byte("test response data"))
	assert.NotNil(msg)
	assert.NotEmpty(msg.ID)
	assert.NotEmpty(msg.Timestamp)
	assert.Equal("7YHPeiMNWetQV9", msg.PeerID)
	assert.Equal([]byte("test response data"), msg.Data)
}

func TestMessage_NewMessage(t *testing.T) {
	assert := assert.New(t)

	msg := newMessage([]byte("test data"))
	assert.NotNil(msg)
	assert.NotEmpty(msg.Timestamp)
	assert.Equal([]byte("test data"), msg.Data)
}

package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRequestMessage(t *testing.T) {
	msg := newRequestMessage("testPeerID", MessageRequestTypeKnownPeers, []byte("test request data"))
	assert.NotNil(t, msg)
	assert.NotEmpty(t, msg.ID)
	assert.NotEmpty(t, msg.Timestamp)
	assert.Equal(t, "7YHPeiMNWetQV9", msg.PeerID)
	assert.Equal(t, MessageRequestTypeKnownPeers, msg.Procedure)
	assert.Equal(t, []byte("test request data"), msg.Data)
}

func TestNewResponseMessage(t *testing.T) {
	msg := newResponseMessage("testPeerID", "123456789", []byte("test response data"))
	assert.NotNil(t, msg)
	assert.NotEmpty(t, msg.ID)
	assert.NotEmpty(t, msg.Timestamp)
	assert.Equal(t, "7YHPeiMNWetQV9", msg.PeerID)
	assert.Equal(t, []byte("test response data"), msg.Data)
}

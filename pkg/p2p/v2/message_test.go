package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRequestMessage(t *testing.T) {
	msg := newRequestMessage("testPeerID", MessageRequestTypeKnownPeers, []byte("test request data"))
	assert.NotNil(t, msg)
	assert.Equal(t, "7YHPeiMNWetQV9", msg.MsgData.PeerID)
	assert.Equal(t, uint32(MessageRequestTypeKnownPeers), msg.Procedure)
	assert.Equal(t, []byte("test request data"), msg.Data)
}

func TestNewResponseMessage(t *testing.T) {
	msg := newResponseMessage("testPeerID", "123456789", []byte("test response data"))
	assert.NotNil(t, msg)
	assert.Equal(t, "7YHPeiMNWetQV9", msg.MsgData.PeerID)
	assert.Equal(t, "123456789", msg.ReqMsgID)
	assert.Equal(t, []byte("test response data"), msg.Data)
}

func TestNewMessageData(t *testing.T) {
	msg := newMessageData("testRemotePeerID")
	assert.NotNil(t, msg)
	assert.NotEmpty(t, msg.Id)
	assert.NotEmpty(t, msg.Timestamp)
	assert.Equal(t, "FNe9LDdyGdfSxvugSunT9d", msg.PeerID)
}

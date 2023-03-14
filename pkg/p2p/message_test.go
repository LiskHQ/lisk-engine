package p2p

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage_NewRequestMessage(t *testing.T) {
	assert := assert.New(t)

	msg := newRequestMessage(testPeerID, testProcedure, []byte(testRequestData))
	assert.NotNil(msg)
	assert.NotEmpty(msg.ID)
	assert.NotEmpty(msg.Timestamp)
	assert.Equal(testPeerID, msg.PeerID)
	assert.Equal(testProcedure, msg.Procedure)
	assert.Equal([]byte(testRequestData), msg.Data)
}

func TestMessage_NewResponseMessage(t *testing.T) {
	assert := assert.New(t)

	msg := newResponseMessage(testReqMsgID, testProcedure, []byte(testResponseData), errors.New(testError))
	assert.NotNil(msg)
	assert.NotEmpty(msg.ID)
	assert.Equal(testProcedure, msg.Procedure)
	assert.NotEmpty(msg.Timestamp)
	assert.Equal([]byte(testResponseData), msg.Data)
	assert.Equal(testError, msg.Error)
}

func TestMessage_NewMessage(t *testing.T) {
	assert := assert.New(t)

	msg := NewMessage([]byte(testData))
	assert.NotNil(msg)
	assert.NotEmpty(msg.Timestamp)
	assert.Equal([]byte(testData), msg.Data)
}

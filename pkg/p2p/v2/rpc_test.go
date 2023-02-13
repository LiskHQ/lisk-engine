package p2p

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRPC_NewResponse(t *testing.T) {
	assert := assert.New(t)

	e := newResponse(testTimestamp, testPeerID, []byte(testData), errors.New(testError))
	assert.Equal(testTimestamp, e.timestamp)
	assert.Equal(testPeerID, e.peerID)
	assert.Equal([]byte(testData), e.data)
	assert.Equal(errors.New(testError), e.err)
}

func TestRPC_Getters(t *testing.T) {
	assert := assert.New(t)

	e := newResponse(testTimestamp, testPeerID, []byte(testData), errors.New(testError))
	assert.Equal(testTimestamp, e.Timestamp())
	assert.Equal(testPeerID, e.PeerID())
	assert.Equal([]byte(testData), e.Data())
	assert.Equal(errors.New(testError), e.Error())
}

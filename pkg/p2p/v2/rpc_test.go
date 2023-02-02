package p2p

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRPC_NewResponse(t *testing.T) {
	assert := assert.New(t)

	timestamp := int64(123456789)
	peerID := "testPeerID"
	data := []byte("testData")
	err := errors.New("testError")

	e := newResponse(timestamp, peerID, data, err)
	assert.Equal(int64(123456789), e.timestamp)
	assert.Equal("testPeerID", e.peerID)
	assert.Equal([]byte("testData"), e.data)
	assert.Equal(errors.New("testError"), e.err)
}

func TestRPC_Getters(t *testing.T) {
	assert := assert.New(t)

	timestamp := int64(123456789)
	peerID := "testPeerID"
	data := []byte("testData")
	err := errors.New("testError")

	e := newResponse(timestamp, peerID, data, err)
	assert.Equal(int64(123456789), e.Timestamp())
	assert.Equal("testPeerID", e.PeerID())
	assert.Equal([]byte("testData"), e.Data())
	assert.Equal(errors.New("testError"), e.Error())
}

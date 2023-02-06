package p2p

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

const timestamp = int64(123456789)
const peerID = "testPeerID"
const data = "testData"
const err = "testError"

func TestRPC_NewResponse(t *testing.T) {
	assert := assert.New(t)

	e := newResponse(timestamp, peerID, []byte(data), errors.New(err))
	assert.Equal(timestamp, e.timestamp)
	assert.Equal(peerID, e.peerID)
	assert.Equal([]byte(data), e.data)
	assert.Equal(errors.New(err), e.err)
}

func TestRPC_Getters(t *testing.T) {
	assert := assert.New(t)

	e := newResponse(timestamp, peerID, []byte(data), errors.New(err))
	assert.Equal(timestamp, e.Timestamp())
	assert.Equal(peerID, e.PeerID())
	assert.Equal([]byte(data), e.Data())
	assert.Equal(errors.New(err), e.Error())
}

package p2p

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	msg        = NewMessage([]byte("testMessageData"))
	invalidMsg = NewMessage([]byte("testMessageInvalid"))
)

func TestValidator(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	assert.Equal(testMV(ctx, invalidMsg), ValidationReject)
	assert.Equal(testMV(ctx, msg), ValidationAccept)
}

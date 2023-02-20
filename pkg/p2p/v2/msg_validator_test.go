package p2p

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidator(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	assert.Equal(testMV(ctx, invalidMsg), ValidationReject)
	assert.Equal(testMV(ctx, msg), ValidationAccept)
}

package math

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeAdd(t *testing.T) {
	_, ok := SafeAdd(math.MaxUint64, math.MaxUint64)
	assert.False(t, ok)
	res, ok := SafeAdd(1000000000000, 2000000000000)
	assert.True(t, ok)
	assert.Equal(t, uint64(3000000000000), res)
}

func TestSafeSub(t *testing.T) {
	_, ok := SafeSub(1000000000000, 2000000000000)
	assert.False(t, ok)

	res, ok := SafeSub(2000000000000, 1000000000000)
	assert.True(t, ok)
	assert.Equal(t, uint64(1000000000000), res)
}

func TestSafeMul(t *testing.T) {
	_, ok := SafeMul(1000000000000, math.MaxUint64)
	assert.False(t, ok)

	res, ok := SafeMul(2000000000000, 3)
	assert.True(t, ok)
	assert.Equal(t, uint64(6000000000000), res)
}

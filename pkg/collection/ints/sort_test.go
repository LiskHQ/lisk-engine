package ints

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMax(t *testing.T) {
	assert.Equal(t, 4, Max(-3, 2, 0, 4, 2, 1, 4))
	assert.Equal(t, 5, Max(-3, 2, 0, 4, 5, 2, 1, 4))

	assert.Equal(t, 4, Max(2, 0, 4, 2, 1, 4))
	assert.Equal(t, 5, Max(2, 0, 4, 5, 2, 1, 4))
}

func TestMin(t *testing.T) {
	assert.Equal(t, -3, Min(-3, 2, 0, 4, 2, 1, -3))
	assert.Equal(t, -5, Min(-3, 2, 0, 4, -5, 2, 1, 4))

	assert.Equal(t, 0, Min(2, 0, 4, 2, 0, 4))
	assert.Equal(t, 1, Min(2, 4, 5, 2, 1, 4))
}

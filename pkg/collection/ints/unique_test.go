package ints

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnique(t *testing.T) {
	original := []uint32{91, 3, 2, 3, 5, 8, 13, 91, 34}

	assert.Equal(t, false, IsUnique(original))

	filtered := Unique(original)
	assert.Equal(t, true, IsUnique(filtered))
	assert.Len(t, filtered, 7)
}

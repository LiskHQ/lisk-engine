package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLastHeight(t *testing.T) {
	heights := getLastHeights(200, 10)
	assert.Equal(t, 9, len(heights))
	assert.Equal(t, uint32(200), heights[0])
	assert.Equal(t, uint32(199), heights[1])
}

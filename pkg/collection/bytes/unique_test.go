package bytes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnique(t *testing.T) {
	original := [][]byte{
		{1, 2, 3, 4, 5, 6, 7, 7},
		{91, 3, 2, 3, 5, 8, 13, 21, 34},
		{1, 2, 3, 5, 8, 13, 21, 34},
		{1, 2, 3, 4, 5, 6, 7, 7},
	}

	assert.Equal(t, false, IsUnique(original))

	filtered := Unique(original)
	assert.Equal(t, true, IsUnique(filtered))
	assert.Len(t, filtered, 3)
}

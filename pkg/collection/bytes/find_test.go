package bytes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindIndex(t *testing.T) {
	original := [][]byte{
		{1, 2, 3, 5, 8, 13, 21, 34},
		{3, 5, 8, 13, 21, 34},
		{3, 5, 8, 13, 21, 34, 0, 99, 255},
	}
	assert.Equal(t, FindIndex(original, []byte{3, 5, 8, 13, 21, 34}), 1)
	assert.Equal(t, FindIndex(original, []byte{3}), -1)
}

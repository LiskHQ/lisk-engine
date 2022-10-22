package bytes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSort(t *testing.T) {
	original := [][]byte{
		{1, 2, 3, 4, 5, 6, 7, 7},
		{91, 3, 2, 3, 5, 8, 13, 21, 34},
		{1, 2, 3, 5, 8, 13, 21, 34},
	}

	assert.Equal(t, false, IsSorted(original))

	Sort(original)
	assert.Equal(t, true, IsSorted(original))
	assert.Equal(t, []byte{1, 2, 3, 4}, original[0][:4])
	assert.Equal(t, []byte{1, 2, 3, 5}, original[1][:4])
	assert.Equal(t, []byte{91, 3, 2, 3}, original[2][:4])
}

func TestReverse(t *testing.T) {
	original := []byte{91, 3, 2, 3, 5, 8, 13, 21, 34}
	assert.Equal(t, []byte{34, 21, 13, 8, 5, 3, 2, 3, 91}, Reverse(original))
}

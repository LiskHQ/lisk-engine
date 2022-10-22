package bytes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopy(t *testing.T) {
	original := []byte{1, 2, 3, 5, 8, 13, 21, 34}
	copied := Copy(original)
	assert.Equal(t, original, copied)
	original[0], original[1] = original[1], original[0]
}

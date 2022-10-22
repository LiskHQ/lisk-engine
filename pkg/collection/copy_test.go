package collection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopy(t *testing.T) {
	original := []int{1, 2, 3, 4, 5}
	copied := Copy(original)
	copied[2] = 6

	assert.NotEqual(t, original, copied)
}

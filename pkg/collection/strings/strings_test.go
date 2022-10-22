package strings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContain(t *testing.T) {
	assert.Equal(t, true, Contain([]string{"a", "b", "d", "c"}, "d"))
	assert.Equal(t, false, Contain([]string{"a", "b", "d", "c"}, "f"))
}

func TestUnique(t *testing.T) {
	orignal := []string{"a", "b", "d", "c", "ran", "go", "b", "ap", "a"}

	assert.Equal(t, false, IsUnique(orignal))
	filtered := Unique(orignal)
	assert.Equal(t, true, IsUnique(filtered))
	assert.Len(t, filtered, 7)
}

func TestGenerateRandomString(t *testing.T) {
	assert.Len(t, GenerateRandom(10), 10)
	assert.NotEqual(t, GenerateRandom(10), GenerateRandom(10))
}

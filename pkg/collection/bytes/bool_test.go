package bytes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBoolsToBytes(t *testing.T) {
	assert.Equal(t, FromBools([]bool{true, false, false, true, false, false, false, false}), []byte{byte(0b10010000)})
	assert.Equal(t, FromBools([]bool{false, false, false, false, false, false, false, false}), []byte{byte(0b00000000)})
	assert.Equal(t, FromBools([]bool{true, true, true}), []byte{byte(0b00000111)})
}

func TestBytesToBools(t *testing.T) {
	assert.Equal(t, ToBools([]byte{byte(0b10010000)}), []bool{true, false, false, true, false, false, false, false})
	assert.Equal(t, ToBools([]byte{byte(0b00000000)}), []bool{false, false, false, false, false, false, false, false})
}

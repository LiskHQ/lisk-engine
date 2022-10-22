package bytes

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUint32ToBytes(t *testing.T) {
	assert.Equal(t, FromUint32(math.MaxUint32), []byte{255, 255, 255, 255})
	assert.Equal(t, FromUint32(0), []byte{0, 0, 0, 0})
	assert.Equal(t, FromUint32(25), []byte{0, 0, 0, 25})
}

func TestUint64ToBytes(t *testing.T) {
	assert.Equal(t, FromUint64(math.MaxUint64), []byte{255, 255, 255, 255, 255, 255, 255, 255})
	assert.Equal(t, FromUint64(0), []byte{0, 0, 0, 0, 0, 0, 0, 0})
	assert.Equal(t, FromUint64(25), []byte{0, 0, 0, 0, 0, 0, 0, 25})
}

func TestUint16ToBytes(t *testing.T) {
	assert.Equal(t, FromUint16(math.MaxUint16), []byte{255, 255})
	assert.Equal(t, FromUint16(0), []byte{0, 0})
	assert.Equal(t, FromUint16(25), []byte{0, 25})
}

func TestBytesToUint32(t *testing.T) {
	assert.Equal(t, ToUint32([]byte{255, 255, 255, 255}), uint32(math.MaxUint32))
	assert.Equal(t, ToUint32([]byte{0, 0, 0, 0}), uint32(0))
	assert.Equal(t, ToUint32([]byte{0, 0, 0, 25}), uint32(25))
	assert.Equal(t, ToUint32([]byte{0, 0, 0, 25, 99}), uint32(25))
}

func TestBytesToUint64(t *testing.T) {
	assert.Equal(t, ToUint64([]byte{255, 255, 255, 255, 255, 255, 255, 255}), uint64(math.MaxUint64))
	assert.Equal(t, ToUint64([]byte{0, 0, 0, 0, 0, 0, 0, 0}), uint64(0))
	assert.Equal(t, ToUint64([]byte{0, 0, 0, 0, 0, 0, 0, 25}), uint64(25))
}

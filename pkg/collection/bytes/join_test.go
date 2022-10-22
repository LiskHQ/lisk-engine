package bytes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	emptyHash = []byte{1, 2, 3, 4, 5, 6, 7, 7}
	prefix    = []byte{3}
)

func TestJoin(t *testing.T) {
	assert.Equal(t, Join(emptyHash, prefix), []byte{1, 2, 3, 4, 5, 6, 7, 7, 3})
}

func TestJoinSize(t *testing.T) {
	assert.Equal(t, JoinSize(9, emptyHash, prefix), []byte{1, 2, 3, 4, 5, 6, 7, 7, 3})
}

func TestJoinSlice(t *testing.T) {
	a := [][]byte{{1, 2, 3}, {3, 2, 1}}
	b := [][]byte{{3, 4, 5}}

	result := JoinSlice(a, b)

	// mutate original data
	b[0] = []byte{7, 7, 7}

	assert.Equal(t, [][]byte{{1, 2, 3}, {3, 2, 1}, {3, 4, 5}}, result)
}

func BenchmarkJoin(b *testing.B) {
	for n := 0; n < b.N; n++ {
		Join(prefix, emptyHash)
	}
	b.StopTimer()
}

func BenchmarkJoinSize(b *testing.B) {
	for n := 0; n < b.N; n++ {
		JoinSize(len(emptyHash)+len(prefix), prefix, emptyHash)
	}
	b.StopTimer()
}

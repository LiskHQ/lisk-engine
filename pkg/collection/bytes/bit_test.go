package bytes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBitSet(t *testing.T) {
	cases := []struct {
		input    []byte
		index    int
		expected bool
	}{
		{
			input:    []byte{0, 0, 0, 1},
			index:    31,
			expected: true,
		},
		{
			input:    []byte{255},
			index:    0,
			expected: true,
		},
		{
			input:    FromBools([]bool{false, false, true, false, true, false, false, false}),
			index:    2,
			expected: true,
		},
		{
			input:    FromBools([]bool{false, false, true, false, true, false, false, false}),
			index:    3,
			expected: false,
		},
		{
			input:    FromBools([]bool{false, false, true, false, true, false, false, false}),
			index:    4,
			expected: true,
		},
	}
	for _, testCase := range cases {
		t.Log(testCase.input)
		assert.Equal(t, testCase.expected, IsBitSet(testCase.input, testCase.index))
	}
}

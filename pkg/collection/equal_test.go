package collection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEqual(t *testing.T) {
	testCases := []struct {
		left   []bool
		right  []bool
		result bool
	}{
		{
			left:   []bool{false, true},
			right:  []bool{true, true},
			result: false,
		},
		{
			left:   []bool{false, true},
			right:  []bool{false},
			result: false,
		},
		{
			left:   []bool{false, true, false},
			right:  []bool{false, true},
			result: false,
		},
		{
			left:   []bool{false, true},
			right:  []bool{false, true},
			result: true,
		},
	}

	for _, testCase := range testCases {
		assert.Equal(t, Equal(testCase.left, testCase.right), testCase.result)
		assert.Equal(t, Equal(testCase.right, testCase.left), testCase.result)
	}
}

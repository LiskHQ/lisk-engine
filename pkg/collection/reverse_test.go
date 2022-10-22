package collection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReverse(t *testing.T) {
	testCases := []struct {
		list   []int
		result []int
	}{
		{
			list:   []int{0, 1, 2, 3},
			result: []int{3, 2, 1, 0},
		},
		{
			list:   []int{3},
			result: []int{3},
		},
		{
			list:   []int{1, 2, 3},
			result: []int{3, 2, 1},
		},
	}
	for _, testCase := range testCases {
		assert.Equal(t, Reverse(testCase.list), testCase.result)
	}
}

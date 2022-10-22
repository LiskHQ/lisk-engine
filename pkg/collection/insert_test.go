package collection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInsert(t *testing.T) {
	testCases := []struct {
		list   []int
		index  int
		value  int
		result []int
	}{
		{
			list:   []int{0, 1, 2, 3},
			index:  0,
			value:  99,
			result: []int{99, 0, 1, 2, 3},
		},
		{
			list:   []int{0, 1, 2, 3},
			index:  4,
			value:  99,
			result: []int{0, 1, 2, 3, 99},
		},
		{
			list:   []int{0, 1, 2, 3},
			index:  2,
			value:  99,
			result: []int{0, 1, 99, 2, 3},
		},
	}
	for _, testCase := range testCases {
		assert.Equal(t, Insert(testCase.list, testCase.index, testCase.value), testCase.result)
	}
}

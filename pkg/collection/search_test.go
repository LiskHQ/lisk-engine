package collection

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func ExampleBinarySearch() {
	res1 := BinarySearch([]int{3, 4, 5}, func(v int) bool { return v > 4 })
	res2 := BinarySearch([]int{3, 4, 5}, func(v int) bool { return v > 6 })
	res3 := BinarySearch([]int{3, 4, 5}, func(v int) bool { return v > 2 })
	fmt.Println(res1, res2, res3)
	// Output: 2 3 0
}

func TestSearch(t *testing.T) {
	testCases := []struct {
		list   []int
		target int
		result int
	}{
		{
			list:   []int{0, 1, 2, 3},
			target: -1,
			result: 0,
		},
		{
			list:   []int{3},
			target: 5,
			result: 1,
		},
		{
			list:   []int{1, 2, 3},
			target: 2,
			result: 2,
		},
	}
	for _, testCase := range testCases {
		assert.Equal(t, BinarySearch(testCase.list, func(v int) bool {
			return v > testCase.target
		}), testCase.result)
	}
}

package collection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveDuplicateInt(t *testing.T) {
	assert := assert.New(t)

	list := []int{1, 2, 3, 4, 5, 1, 2, 3, 4, 5}
	result := RemoveDuplicate(list)
	assert.Equal([]int{1, 2, 3, 4, 5}, result)

	testCases := []struct {
		list []int
		want []int
	}{
		{
			list: []int{1, 2, 3, 4, 5, 1, 2, 3, 4, 5},
			want: []int{1, 2, 3, 4, 5},
		},
		{
			list: []int{1, 2, 3, 4, 5},
			want: []int{1, 2, 3, 4, 5},
		},
		{
			list: []int{1, 5, 1, 2, 3, 4, 5, 1, 2, 3, 2, 4, 5},
			want: []int{1, 5, 2, 3, 4},
		},
		{
			list: []int{1, 1},
			want: []int{1},
		},
	}

	for _, testCase := range testCases {
		result := RemoveDuplicate(testCase.list)
		assert.Equal(testCase.want, result)
	}
}

func TestRemoveDuplicateString(t *testing.T) {
	assert := assert.New(t)

	list := []string{"1", "2", "3", "4", "5", "1", "2", "3", "4", "5"}
	result := RemoveDuplicate(list)
	assert.Equal([]string{"1", "2", "3", "4", "5"}, result)

	testCases := []struct {
		list []string
		want []string
	}{
		{
			list: []string{"1", "2", "3", "4", "5", "1", "2", "3", "4", "5"},
			want: []string{"1", "2", "3", "4", "5"},
		},
		{
			list: []string{"1", "2", "3", "4", "5"},
			want: []string{"1", "2", "3", "4", "5"},
		},
		{
			list: []string{"1", "5", "1", "2", "3", "4", "5", "1", "2", "3", "4", "5"},
			want: []string{"1", "5", "2", "3", "4"},
		},
		{
			list: []string{"1", "1"},
			want: []string{"1"},
		},
	}

	for _, testCase := range testCases {
		result := RemoveDuplicate(testCase.list)
		assert.Equal(testCase.want, result)
	}
}

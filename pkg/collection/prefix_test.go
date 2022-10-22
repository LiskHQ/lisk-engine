package collection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommonPrefix(t *testing.T) {
	testCases := []struct {
		left   []bool
		right  []bool
		result []bool
	}{
		{
			left:   []bool{false, true},
			right:  []bool{true, true},
			result: []bool{},
		},
		{
			left:   []bool{false, true},
			right:  []bool{false},
			result: []bool{false},
		},
		{
			left:   []bool{false, true, false},
			right:  []bool{false, true},
			result: []bool{false, true},
		},
		{
			left:   []bool{false, true},
			right:  []bool{false, true},
			result: []bool{false, true},
		},
	}

	for i, testCase := range testCases {
		t.Logf("case %d", i)
		assert.Equal(t, CommonPrefix(testCase.left, testCase.right), testCase.result)
		assert.Equal(t, CommonPrefix(testCase.right, testCase.left), testCase.result)
	}
}

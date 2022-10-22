package collection

import (
	"testing"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
	"github.com/stretchr/testify/assert"
)

func TestFind(t *testing.T) {
	cases := []struct {
		input     []int
		predicate func(val int) bool
		expected  int
	}{
		{
			input: []int{1, 2, 3},
			predicate: func(val int) bool {
				return val == 7
			},
			expected: 0,
		},
		{
			input: []int{1, 7, 3},
			predicate: func(val int) bool {
				return val == 7
			},
			expected: 7,
		},
	}

	for _, testCase := range cases {
		result := Find(testCase.input, testCase.predicate)
		assert.Equal(t, testCase.expected, result)
	}

	casesObj := []struct {
		input     []codec.Hex
		predicate func(val codec.Hex) bool
		expected  codec.Hex
	}{
		{
			input: []codec.Hex{
				crypto.RandomBytes(32),
			},
			predicate: func(val codec.Hex) bool {
				return bytes.Equal(val, []byte{7, 7, 7})
			},
			expected: nil,
		},
		{
			input: []codec.Hex{
				[]byte{7, 7, 7},
			},
			predicate: func(val codec.Hex) bool {
				return bytes.Equal(val, []byte{7, 7, 7})
			},
			expected: []byte{7, 7, 7},
		},
	}

	for _, testCase := range casesObj {
		result := Find(testCase.input, testCase.predicate)
		assert.Equal(t, testCase.expected, result)
	}
}

func TestFindIndex(t *testing.T) {
	cases := []struct {
		input     []int
		predicate func(val int) bool
		expected  int
	}{
		{
			input: []int{1, 2, 3},
			predicate: func(val int) bool {
				return val == 7
			},
			expected: -1,
		},
		{
			input: []int{1, 7, 3},
			predicate: func(val int) bool {
				return val == 7
			},
			expected: 1,
		},
	}

	for _, testCase := range cases {
		result := FindIndex(testCase.input, testCase.predicate)
		assert.Equal(t, testCase.expected, result)
	}

	casesObj := []struct {
		input     []codec.Hex
		predicate func(val codec.Hex) bool
		expected  int
	}{
		{
			input: []codec.Hex{
				crypto.RandomBytes(32),
			},
			predicate: func(val codec.Hex) bool {
				return bytes.Equal(val, []byte{7, 7, 7})
			},
			expected: -1,
		},
		{
			input: []codec.Hex{
				[]byte{0, 7, 7},
				[]byte{7, 7, 7},
				[]byte{7, 8, 7},
			},
			predicate: func(val codec.Hex) bool {
				return bytes.Equal(val, []byte{7, 7, 7})
			},
			expected: 1,
		},
	}

	for _, testCase := range casesObj {
		result := FindIndex(testCase.input, testCase.predicate)
		assert.Equal(t, testCase.expected, result)
	}
}

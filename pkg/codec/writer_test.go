package codec

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteBool(t *testing.T) {
	cases := []struct {
		input       bool
		fieldNumber int
		result      string
	}{
		{
			input:       true,
			fieldNumber: 1,
			result:      "0801",
		},
		{
			input:       false,
			fieldNumber: 1,
			result:      "0800",
		},
	}
	for _, c := range cases {
		writer := NewWriter()
		writer.WriteBool(1, c.input)
		expected, err := hex.DecodeString(c.result)
		assert.Nil(t, err)
		assert.Equal(t, expected, writer.Result())
	}
}

func TestWriteBytes(t *testing.T) {
	cases := []struct {
		input       []byte
		fieldNumber int
		result      string
	}{
		{
			input:       mustDecodeHex("e11a11364738225813f86ea85214400e5db08d6e"),
			fieldNumber: 1,
			result:      "0a14e11a11364738225813f86ea85214400e5db08d6e",
		},
	}
	for _, c := range cases {
		writer := NewWriter()
		writer.WriteBytes(1, c.input)
		expected, err := hex.DecodeString(c.result)
		assert.Nil(t, err)
		assert.Equal(t, expected, writer.Result())
	}
}

func TestWriteBytesArray(t *testing.T) {
	cases := []struct {
		input       [][]byte
		fieldNumber int
		result      string
	}{
		{
			input: [][]byte{
				mustDecodeHex("e11a11364738225813f86ea85214400e5db08d6e"),
				mustDecodeHex(""),
				mustDecodeHex("676f676f676f67"),
			},
			fieldNumber: 1,
			result:      "0a14e11a11364738225813f86ea85214400e5db08d6e0a000a07676f676f676f67",
		},
	}
	for _, c := range cases {
		writer := NewWriter()
		writer.WriteBytesArray(1, c.input)
		expected, err := hex.DecodeString(c.result)
		assert.Nil(t, err)
		assert.Equal(t, expected, writer.Result())
	}
}

func TestWriteUInt(t *testing.T) {
	cases := []struct {
		input       uint64
		fieldNumber int
		result      string
	}{
		{
			input:       10,
			fieldNumber: 1,
			result:      "080a",
		},
		{
			input:       372036854775807,
			fieldNumber: 1,
			result:      "08ffffc9a4d9cb54",
		},
	}
	for _, c := range cases {
		writer := NewWriter()
		writer.WriteUInt(c.fieldNumber, c.input)
		expected, err := hex.DecodeString(c.result)
		assert.Nil(t, err)
		assert.Equal(t, expected, writer.Result())
	}
}

func TestWriteUInts(t *testing.T) {
	cases := []struct {
		input       []uint64
		fieldNumber int
		result      string
	}{
		{
			input:       []uint64{3, 1, 4, 1, 5, 9, 2, 6, 5},
			fieldNumber: 1,
			result:      "0a09030104010509020605",
		},
	}
	for _, c := range cases {
		{
			writer := NewWriter()
			writer.WriteUInts(c.fieldNumber, c.input)
			expected, err := hex.DecodeString(c.result)
			assert.Nil(t, err)
			assert.Equal(t, expected, writer.Result())
		}
		{
			writer := NewWriter()
			input := make([]uint32, len(c.input))
			for i, val := range c.input {
				input[i] = uint32(val)
			}
			writer.WriteUInt32s(c.fieldNumber, input)
			expected, err := hex.DecodeString(c.result)
			assert.Nil(t, err)
			assert.Equal(t, expected, writer.Result())
		}
	}
}

func TestWriteInts(t *testing.T) {
	cases := []struct {
		input       []int64
		fieldNumber int
		result      string
	}{
		{
			input:       []int64{-2, -1, 2147483647, -1, -3, -5, 2, 6, -3},
			fieldNumber: 1,
			result:      "0a0d0301feffffff0f010509040c05",
		},
	}
	for _, c := range cases {
		{
			writer := NewWriter()
			writer.WriteInts(c.fieldNumber, c.input)
			expected, err := hex.DecodeString(c.result)
			assert.Nil(t, err)
			assert.Equal(t, expected, writer.Result())
		}
		{
			writer := NewWriter()
			input := make([]int32, len(c.input))
			for i, val := range c.input {
				input[i] = int32(val)
			}
			writer.WriteInt32s(c.fieldNumber, input)
			expected, err := hex.DecodeString(c.result)
			assert.Nil(t, err)
			assert.Equal(t, expected, writer.Result())
		}
	}
}

func TestWriteInt(t *testing.T) {
	cases := []struct {
		input       int64
		fieldNumber int
		result      string
	}{
		{
			input:       8758644044142934452,
			result:      "08e8e6d7d3ca96fa8cf301",
			fieldNumber: 1,
		},
		{
			input:       -10,
			fieldNumber: 1,
			result:      "0813",
		},
		{
			input:       -9007199254740991,
			fieldNumber: 1,
			result:      "08fdffffffffffff1f",
		},
	}
	for _, c := range cases {
		writer := NewWriter()
		writer.WriteInt(1, c.input)
		expected, err := hex.DecodeString(c.result)
		assert.Nil(t, err)
		assert.Equal(t, expected, writer.Result())
	}
}

func TestWriteBools(t *testing.T) {
	cases := []struct {
		input       []bool
		fieldNumber int
		result      string
	}{
		{
			input:       []bool{true, true, false, true, false, false},
			fieldNumber: 1,
			result:      "0a06010100010000",
		},
	}
	for _, c := range cases {
		writer := NewWriter()
		writer.WriteBools(c.fieldNumber, c.input)
		expected, err := hex.DecodeString(c.result)
		assert.Nil(t, err)
		assert.Equal(t, expected, writer.Result())
	}
}

func TestWriteStrings(t *testing.T) {
	cases := []struct {
		input       []string
		fieldNumber int
		result      string
	}{
		{
			input:       []string{"lisk", "", "gogogog"},
			fieldNumber: 1,
			result:      "0a046c69736b0a000a07676f676f676f67",
		},
		{
			input:       []string{},
			fieldNumber: 1,
			result:      "",
		},
	}
	for _, c := range cases {
		writer := NewWriter()
		writer.WriteStrings(c.fieldNumber, c.input)
		expected, err := hex.DecodeString(c.result)
		assert.Nil(t, err)
		assert.Equal(t, expected, writer.Result())
	}
}

func (t *testDecodableAccount) Encode() []byte {
	writer := NewWriter()
	writer.WriteString(1, t.Address)
	writer.WriteUInt(2, t.Amount)
	return writer.Result()
}

func TestWriteEncodables(t *testing.T) {
	expected, err := hex.DecodeString("0a2e0a286531316131313336343733383232353831336638366561383532313434303065356462303864366510a08d060a2e0a286161326131313336343733383232353831336638366561383532313434303065356462303866666610e0a712")
	assert.Nil(t, err)
	vals := []*testDecodableAccount{
		{
			Address: "e11a11364738225813f86ea85214400e5db08d6e",
			Amount:  100000,
		},
		{
			Address: "aa2a11364738225813f86ea85214400e5db08fff",
			Amount:  300000,
		},
	}

	writer := NewWriter()

	for _, val := range vals {
		writer.WriteEncodable(1, val)
	}
	assert.Equal(t, expected, writer.Result())
}

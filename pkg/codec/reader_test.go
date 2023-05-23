package codec

import (
	"encoding/binary"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

type BoolStruct struct {
	Prop bool
}

func TestReadBool(t *testing.T) {
	cases := []struct {
		input       string
		fieldNumber int
		strict      bool
		result      bool
		err         string
	}{
		{
			input:       "0801",
			fieldNumber: 1,
			strict:      false,
			result:      true,
			err:         "",
		},
		{
			input:       "0800",
			fieldNumber: 1,
			strict:      false,
			result:      false,
			err:         "",
		},
		{
			input:       "0803",
			fieldNumber: 1,
			strict:      false,
			result:      false,
			err:         ErrInvalidData.Error(),
		},
		{
			input:       "0a01",
			fieldNumber: 1,
			strict:      false,
			result:      false,
			err:         ErrInvalidData.Error(),
		},
		{
			input:       "0801",
			fieldNumber: 2,
			strict:      true,
			result:      false,
			err:         ErrUnexpectedFieldNumber.Error(),
		},
	}
	for _, testCase := range cases {
		input, err := hex.DecodeString(testCase.input)
		assert.Nil(t, err)

		reader := NewReader(input)
		result, err := reader.ReadBool(testCase.fieldNumber, testCase.strict)
		if testCase.err == "" {
			assert.Nil(t, err)
			assert.Equal(t, result, testCase.result)
		} else {
			assert.Contains(t, err.Error(), testCase.err)
		}
	}
}

func TestReadBytes(t *testing.T) {
	cases := []struct {
		input       string
		fieldNumber int
		strict      bool
		result      []byte
		err         string
	}{
		{
			input:       "0a14e11a11364738225813f86ea85214400e5db08d6e",
			fieldNumber: 1,
			strict:      false,
			result:      []byte{225, 26, 17, 54, 71, 56, 34, 88, 19, 248, 110, 168, 82, 20, 64, 14, 93, 176, 141, 110},
			err:         "",
		},
		{
			input:       "0a14e11a11364738225813f86ea85214400e5db08d6e",
			fieldNumber: 1,
			strict:      true,
			result:      []byte{225, 26, 17, 54, 71, 56, 34, 88, 19, 248, 110, 168, 82, 20, 64, 14, 93, 176, 141, 110},
			err:         "",
		},
		{
			input:       "0814e11a11364738225813f86ea85214400e5db08d6e",
			fieldNumber: 1,
			strict:      false,
			result:      []byte{225, 26, 17, 54, 71, 56, 34, 88, 19, 248, 110, 168, 82, 20, 64, 14, 93, 176, 141, 110},
			err:         ErrInvalidData.Error(),
		},
		{
			input:       "1214e11a11364738225813f86ea85214400e5db08d6e",
			fieldNumber: 1,
			strict:      true,
			result:      nil,
			err:         ErrUnexpectedFieldNumber.Error(),
		},
		{
			input:       "0a14e11a11364738225813f86ea85214400e5db0",
			fieldNumber: 1,
			strict:      true,
			result:      nil,
			err:         "invalid byte size",
		},
	}
	for _, testCase := range cases {
		input, err := hex.DecodeString(testCase.input)
		assert.Nil(t, err)

		reader := NewReader(input)
		result, err := reader.ReadBytes(testCase.fieldNumber, testCase.strict)
		if testCase.err == "" {
			assert.Nil(t, err)
			assert.Equal(t, result, testCase.result)
		} else {
			assert.Contains(t, err.Error(), testCase.err)
		}
	}
}

func TestReadUInt(t *testing.T) {
	cases := []struct {
		input       string
		fieldNumber int
		result      uint64
		strict      bool
		err         string
	}{
		{
			input:       "080a",
			fieldNumber: 1,
			strict:      false,
			result:      10,
		},
		{
			input:       "08ffffc9a4d9cb54",
			fieldNumber: 1,
			strict:      false,
			result:      372036854775807,
		},
		{
			input:       "0affffc9a4d9cb54",
			fieldNumber: 1,
			strict:      false,
			result:      0,
			err:         ErrInvalidData.Error(),
		},
		{
			input:       "08ffffffffffffffffff02", // invalid range
			fieldNumber: 1,
			strict:      false,
			result:      0,
			err:         "out of range",
		},
		{
			input:       "08ffffc9a4d9cb54",
			fieldNumber: 2,
			strict:      true,
			result:      0,
			err:         ErrUnexpectedFieldNumber.Error(),
		},
		{
			input:       "08ffffc9a4d9cb",
			fieldNumber: 1,
			strict:      true,
			result:      0,
			err:         "invalid data",
		},
	}
	for _, c := range cases {
		expected, err := hex.DecodeString(c.input)
		assert.Nil(t, err)
		{
			reader := NewReader(expected)
			result, err := reader.ReadUInt(c.fieldNumber, c.strict)
			if c.err == "" {
				assert.Nil(t, err)
				assert.Equal(t, result, c.result)
			} else {
				assert.Contains(t, err.Error(), c.err)
			}
		}
		{
			reader := NewReader(expected)
			result, err := reader.ReadUInt32(c.fieldNumber, c.strict)
			if c.err == "" {
				assert.Nil(t, err)
				assert.Equal(t, result, uint32(c.result))
			} else {
				assert.Contains(t, err.Error(), c.err)
			}
		}
	}
}

func TestReadInt(t *testing.T) {
	cases := []struct {
		input       string
		fieldNumber int
		result      int64
		strict      bool
		err         string
	}{
		{
			input:       "0813",
			fieldNumber: 1,
			result:      -10,
			strict:      false,
			err:         "",
		},
		{
			input:       "08fdffffffffffff1f",
			fieldNumber: 1,
			result:      -9007199254740991,
			strict:      false,
			err:         "",
		},
		{
			input:       "01fdffffffffffff1f",
			fieldNumber: 1,
			result:      0,
			strict:      false,
			err:         ErrInvalidData.Error(),
		},
		{
			input:       "08ffffffffffffffffff02", // invalid range
			fieldNumber: 1,
			strict:      false,
			result:      0,
			err:         "out of range",
		},
		{
			input:       "08fdffffffffffff1f",
			fieldNumber: 2,
			result:      -9007199254740991,
			strict:      true,
			err:         ErrUnexpectedFieldNumber.Error(),
		},
	}
	for _, c := range cases {
		expected, err := hex.DecodeString(c.input)
		assert.Nil(t, err)

		reader := NewReader(expected)
		result, err := reader.ReadInt(c.fieldNumber, c.strict)
		if c.err == "" {
			assert.Nil(t, err)
			assert.Equal(t, result, c.result)
		} else {
			assert.Contains(t, err.Error(), c.err)
		}
	}
}

func TestReadInt32(t *testing.T) {
	cases := []struct {
		input       string
		fieldNumber int
		result      int64
		strict      bool
		err         string
	}{
		{
			input:       "0813",
			fieldNumber: 1,
			result:      -10,
			strict:      false,
			err:         "",
		},
		{
			input:       "08fdffffffffffff1f",
			fieldNumber: 1,
			result:      -9007199254740991,
			strict:      false,
			err:         "",
		},
		{
			input:       "0afdffffffffffff1f",
			fieldNumber: 1,
			result:      -0,
			strict:      false,
			err:         ErrInvalidData.Error(),
		},
		{
			input:       "08ffffffffffffffffff02", // invalid range
			fieldNumber: 1,
			strict:      false,
			result:      0,
			err:         "out of range",
		},
		{
			input:       "08fdffffffffffff1f",
			fieldNumber: 2,
			result:      -9007199254740991,
			strict:      true,
			err:         ErrUnexpectedFieldNumber.Error(),
		},
	}
	for _, c := range cases {
		expected, err := hex.DecodeString(c.input)
		assert.Nil(t, err)

		reader := NewReader(expected)
		result, err := reader.ReadInt32(c.fieldNumber, c.strict)
		if c.err == "" {
			assert.Nil(t, err)
			assert.Equal(t, result, int32(c.result))
		} else {
			assert.Contains(t, err.Error(), c.err)
		}
	}
}

func TestReadUInts(t *testing.T) {
	cases := []struct {
		input       string
		fieldNumber int
		result      []uint64
	}{
		{
			input:       "0a09030104010509020605",
			fieldNumber: 1,
			result:      []uint64{3, 1, 4, 1, 5, 9, 2, 6, 5},
		},
		{
			input:       "0a09030104010509020605",
			fieldNumber: 2,
			result:      []uint64{},
		},
	}
	for _, c := range cases {
		expected, err := hex.DecodeString(c.input)
		assert.Nil(t, err)

		{
			reader := NewReader(expected)
			result, err := reader.ReadUInts(c.fieldNumber)
			assert.Nil(t, err)
			assert.Equal(t, result, c.result)
		}
		{
			reader := NewReader(expected)
			result, err := reader.ReadUInt32s(c.fieldNumber)
			assert.Nil(t, err)
			conv := make([]uint32, len(c.result))
			for i, v := range c.result {
				conv[i] = uint32(v)
			}
			assert.Equal(t, result, conv)
		}
	}
}

func TestReadInts(t *testing.T) {
	cases := []struct {
		input       string
		fieldNumber int
		result      []int64
	}{
		{
			input:       "0a09030108010509040c05",
			fieldNumber: 1,
			result:      []int64{-2, -1, 4, -1, -3, -5, 2, 6, -3},
		},
		{
			input:       "0a09030108010509040c05",
			fieldNumber: 2,
			result:      []int64{},
		},
	}
	for _, c := range cases {
		expected, err := hex.DecodeString(c.input)
		assert.Nil(t, err)

		{
			reader := NewReader(expected)
			result, err := reader.ReadInts(c.fieldNumber)
			assert.Nil(t, err)
			assert.Equal(t, c.result, result)
		}
	}
}

func TestReadBools(t *testing.T) {
	cases := []struct {
		input       string
		fieldNumber int
		result      []bool
	}{
		{
			input:       "0a06010100010000",
			fieldNumber: 1,
			result:      []bool{true, true, false, true, false, false},
		},
		{
			input:       "0a06010100010000",
			fieldNumber: 2,
			result:      []bool{},
		},
	}
	for _, c := range cases {
		expected, err := hex.DecodeString(c.input)
		assert.Nil(t, err)

		reader := NewReader(expected)
		result, err := reader.ReadBools(c.fieldNumber)
		assert.Nil(t, err)
		assert.Equal(t, result, c.result)
	}
}

func TestReadString(t *testing.T) {
	cases := []struct {
		input       string
		fieldNumber int
		result      string
		strict      bool
		err         string
	}{
		{
			input:       "0a046c69736b",
			fieldNumber: 1,
			result:      "lisk",
			strict:      false,
			err:         "",
		},
		{
			input:       "08046c69736b",
			fieldNumber: 1,
			result:      "lisk",
			strict:      false,
			err:         ErrInvalidData.Error(),
		},
		{
			input:       "0a046c69736b",
			fieldNumber: 2,
			result:      "",
			strict:      true,
			err:         ErrUnexpectedFieldNumber.Error(),
		},
	}
	for _, c := range cases {
		expected, err := hex.DecodeString(c.input)
		assert.Nil(t, err)

		reader := NewReader(expected)
		result, err := reader.ReadString(c.fieldNumber, c.strict)
		if c.err == "" {
			assert.Nil(t, err)
			assert.Equal(t, result, c.result)
		} else {
			assert.Contains(t, err.Error(), c.err)
		}
	}
}

func TestReadStrings(t *testing.T) {
	cases := []struct {
		input       string
		fieldNumber int
		result      []string
	}{
		{
			input:       "0a046c69736b0a000a07676f676f676f67",
			fieldNumber: 1,
			result:      []string{"lisk", "", "gogogog"},
		},
		{
			input:       "",
			fieldNumber: 1,
			result:      []string{},
		},
		{
			input:       "0a046c69736b0a000a07676f676f676f67",
			fieldNumber: 2,
			result:      []string{},
		},
	}
	for _, c := range cases {
		expected, err := hex.DecodeString(c.input)
		assert.Nil(t, err)

		reader := NewReader(expected)
		result, err := reader.ReadStrings(c.fieldNumber)
		assert.Nil(t, err)
		assert.Equal(t, result, c.result)
	}
}

func TestReadBytesArray(t *testing.T) {
	cases := []struct {
		input       string
		fieldNumber int
		result      [][]byte
	}{
		{
			input:       "0a046c69736b0a000a07676f676f676f67",
			fieldNumber: 1,
			result:      [][]byte{mustDecodeHex("6c69736b"), mustDecodeHex(""), mustDecodeHex("676f676f676f67")},
		},
		{
			input:       "",
			fieldNumber: 1,
			result:      [][]byte{},
		},
	}
	for _, c := range cases {
		expected, err := hex.DecodeString(c.input)
		assert.Nil(t, err)

		reader := NewReader(expected)
		result, err := reader.ReadBytesArray(c.fieldNumber)
		assert.Nil(t, err)
		assert.Equal(t, result, c.result)
	}
}

type testDecodable struct {
	vals []*testDecodableAccount
}

type testDecodableAccount struct {
	Address string
	Amount  uint64
}

func (t *testDecodableAccount) DecodeFromReader(reader *Reader) error {
	val1, err := reader.ReadString(1, false)
	if err != nil {
		return err
	}
	t.Address = val1
	val2, err := reader.ReadUInt(2, false)
	if err != nil {
		return err
	}
	t.Amount = val2
	return nil
}

func TestReadDecodable(t *testing.T) {
	{
		expected, err := hex.DecodeString("0a2e0a286531316131313336343733383232353831336638366561383532313434303065356462303864366510a08d06")
		assert.Nil(t, err)

		reader := NewReader(expected)

		vals, err := reader.ReadDecodable(1, func() DecodableReader { return new(testDecodableAccount) }, false)
		assert.Nil(t, err)
		assert.Equal(t, &testDecodableAccount{
			Address: "e11a11364738225813f86ea85214400e5db08d6e",
			Amount:  100000,
		}, vals)
	}
	{
		expected, err := hex.DecodeString("0a2e0a286531316131313336343733383232353831336638366561383532313434303065356462303864366510a08d06")
		assert.Nil(t, err)

		reader := NewReader(expected)

		_, err = reader.ReadDecodable(2, func() DecodableReader { return new(testDecodableAccount) }, true)
		assert.Contains(t, err.Error(), ErrUnexpectedFieldNumber.Error())
	}
	{
		expected, err := hex.DecodeString("0a2e0a286531316131313336343733383232353831336638366561383532313434303065356462303864366510a08d")
		assert.Nil(t, err)

		reader := NewReader(expected)

		_, err = reader.ReadDecodable(1, func() DecodableReader { return new(testDecodableAccount) }, true)
		assert.Contains(t, err.Error(), "invalid data")
	}
}

func TestReadDecodables(t *testing.T) {
	expected, err := hex.DecodeString("0a2e0a286531316131313336343733383232353831336638366561383532313434303065356462303864366510a08d060a2e0a286161326131313336343733383232353831336638366561383532313434303065356462303866666610e0a712")
	assert.Nil(t, err)

	reader := NewReader(expected)

	result := testDecodable{}
	vals, err := reader.ReadDecodables(1, func() DecodableReader { return new(testDecodableAccount) })
	assert.Nil(t, err)
	r := make([]*testDecodableAccount, len(vals))
	for i, v := range vals {
		r[i] = v.(*testDecodableAccount)
	}
	result.vals = r
	assert.Equal(t, []*testDecodableAccount{
		{
			Address: "e11a11364738225813f86ea85214400e5db08d6e",
			Amount:  100000,
		},
		{
			Address: "aa2a11364738225813f86ea85214400e5db08fff",
			Amount:  300000,
		},
	}, result.vals)
}

func TestVarintShortestForm(t *testing.T) {
	cases := []struct {
		data uint64
		size int
	}{
		{
			data: 0,
			size: 1,
		},
		{
			data: 127,
			size: 1,
		},
		{
			data: 128,
			size: 2,
		},
		{
			data: 127 * 127,
			size: 2,
		},
	}

	for _, c := range cases {
		assert.True(t, isVarintShortestForm(c.data, c.size), "Testing %d with size %d", c.data, c.size)
	}
}

func FuzzIsVarintShortestForm(f *testing.F) {
	f.Add(uint64(0))
	f.Fuzz(func(t *testing.T, val uint64) {
		vint := make([]byte, binary.MaxVarintLen64)
		size := binary.PutUvarint(vint, val)
		assert.True(t, isVarintShortestForm(val, size))
	})
}

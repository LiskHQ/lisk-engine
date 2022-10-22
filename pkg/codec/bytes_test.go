package codec

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testHexStruct struct {
	Data Hex `json:"data"`
}

func mustDecodeHex(v string) []byte {
	decoded, err := hex.DecodeString(v)
	if err != nil {
		panic(err)
	}
	return decoded
}

func TestBytesArrayConvert(t *testing.T) {
	bytesArr := [][]byte{
		mustDecodeHex("fc9738370f44dce0bf5877df66ac17afe8346d9f"),
		mustDecodeHex("ab9738370f44dce0bf5877df66ac17afe8346d9f"),
		mustDecodeHex("cd9738370f44dce0bf5877df66ac17afe8346d9f"),
	}

	hexArr := BytesArrayToHexArray(bytesArr)
	assert.Equal(t, []byte(hexArr[0]), bytesArr[0])
	assert.Equal(t, []byte(hexArr[2]), bytesArr[2])

	lisk32Arr := BytesArrayToLisk32Array(bytesArr)
	assert.Equal(t, []byte(lisk32Arr[0]), bytesArr[0])
	assert.Equal(t, []byte(lisk32Arr[2]), bytesArr[2])

	assert.Equal(t, bytesArr, HexArrayToBytesArray(hexArr))
	assert.Equal(t, bytesArr, Lisk32ArrayToBytesArray(lisk32Arr))
}

func TestHexUnmarshal(t *testing.T) {
	expected := []byte(`{"data":"ff1709"}`)
	result := &testHexStruct{}
	err := json.Unmarshal(expected, result)
	assert.Nil(t, err)
	assert.Equal(t, []byte(result.Data), []byte{255, 23, 9})
	assert.Equal(t, "ff1709", result.Data.String())
}

func TestHexMarshal(t *testing.T) {
	expected := &testHexStruct{
		Data: []byte{255, 23, 9},
	}
	result, err := json.Marshal(expected)
	assert.Nil(t, err)
	assert.Equal(t, string(result), `{"data":"ff1709"}`)
}

type testLisk32Struct struct {
	Data Lisk32 `json:"data"`
}

func TestLisk32Unmarshal(t *testing.T) {
	expected, _ := hex.DecodeString("fc9738370f44dce0bf5877df66ac17afe8346d9f")
	input := []byte(`{"data":"lskgr5tu9283t77x8d27g8e95zwqgkc3sogx4zazd"}`)
	result := &testLisk32Struct{}
	err := json.Unmarshal(input, result)
	assert.Nil(t, err)
	assert.Equal(t, expected, []byte(result.Data))
	assert.Equal(t, "lskgr5tu9283t77x8d27g8e95zwqgkc3sogx4zazd", result.Data.String())
}

func TestLisk32Marshal(t *testing.T) {
	hexVal, _ := hex.DecodeString("fc9738370f44dce0bf5877df66ac17afe8346d9f")
	input := &testLisk32Struct{
		Data: hexVal,
	}
	result, err := json.Marshal(input)
	assert.Nil(t, err)
	assert.Equal(t, `{"data":"lskgr5tu9283t77x8d27g8e95zwqgkc3sogx4zazd"}`, string(result))
}

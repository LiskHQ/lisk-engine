package codec

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	lisk32Charset = "zxvcpmbn3465o978uyrtkqew2adsjhfg"
)

var (
	generator = [...]int{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}
)

type Hex []byte

func Lisk32ArrayToBytesArray(val []Lisk32) [][]byte {
	converted := make([][]byte, len(val))
	for i, v := range val {
		converted[i] = v
	}
	return converted
}

func HexArrayToBytesArray(val []Hex) [][]byte {
	converted := make([][]byte, len(val))
	for i, v := range val {
		converted[i] = v
	}
	return converted
}

func BytesArrayToHexArray(val [][]byte) []Hex {
	converted := make([]Hex, len(val))
	for i, v := range val {
		converted[i] = v
	}
	return converted
}

func BytesArrayToLisk32Array(val [][]byte) []Lisk32 {
	converted := make([]Lisk32, len(val))
	for i, v := range val {
		converted[i] = v
	}
	return converted
}

func (h *Hex) UnmarshalJSON(b []byte) error {
	str := ""
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	res, err := hex.DecodeString(str)
	if err != nil {
		return err
	}
	*h = res
	return nil
}

func (h Hex) String() string {
	return hex.EncodeToString(h)
}

func (h Hex) MarshalJSON() ([]byte, error) {
	str := hex.EncodeToString(h)
	return json.Marshal(str)
}

type Lisk32 []byte

func (h *Lisk32) UnmarshalJSON(b []byte) error {
	str := ""
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	res, err := Lisk32ToBytes(str)
	if err != nil {
		return err
	}
	*h = res
	return nil
}

func (h Lisk32) MarshalJSON() ([]byte, error) {
	str, err := BytesToLisk32(h)
	if err != nil {
		return nil, err
	}
	return json.Marshal(str)
}

func (h Lisk32) String() string {
	str, _ := BytesToLisk32(h)
	return str
}

func BytesToLisk32[T ~byte](val []T) (string, error) {
	if len(val) == 0 {
		return "", nil
	}
	if len(val) != 20 {
		return "", fmt.Errorf("lisk32 bytes must be size of 20 but received %d", len(val))
	}
	target := make([]int, len(val))
	for i, v := range val {
		target[i] = int(v)
	}
	uint5Arr := convertUIntArray(target, 8, 5)
	checksum := createChecksum(uint5Arr)
	result := uint5ToLisk32(append(uint5Arr, checksum...))
	return "lsk" + result, nil
}

func Lisk32ToBytes(val string) ([]byte, error) {
	if val == "" {
		return []byte{}, nil
	}
	if err := ValidateLisk32(val); err != nil {
		return nil, err
	}
	target := val[3:35]
	uint5Arr := []int{}
	for _, c := range target {
		index := strings.Index(lisk32Charset, string(c))
		uint5Arr = append(uint5Arr, index)
	}
	uint8Arr := convertUIntArray(uint5Arr, 5, 8)
	result := make([]byte, len(uint8Arr))
	for i, v := range uint8Arr {
		result[i] = byte(v)
	}
	return result, nil
}

func ValidateLisk32(val string) error {
	if len(val) != 41 {
		return fmt.Errorf("lisk32 string must be size of 41 but received %d", len(val))
	}
	withoutChecksum := val[3:]
	uint5Arr := []int{}
	for _, c := range withoutChecksum {
		index := strings.Index(lisk32Charset, string(c))
		if index < 0 {
			return fmt.Errorf("lisk32 string includes invalid character %s", string(c))
		}
		uint5Arr = append(uint5Arr, index)
	}
	if polymod(uint5Arr) != 1 {
		return errors.New("invalid checksum")
	}
	return nil
}

func convertUIntArray(
	uintArray []int,
	fromBits int,
	toBits int,
) []int {
	maxValue := (1 << toBits) - 1
	accumulator := 0
	bits := 0
	result := []int{}
	for p := 0; p < len(uintArray); p++ {
		value := uintArray[p]
		// check that the entry is a value between 0 and 2^frombits-1
		if value < 0 || value>>fromBits != 0 {
			return []int{}
		}

		accumulator = (accumulator << fromBits) | value
		bits += fromBits
		for bits >= toBits {
			bits -= toBits
			result = append(result, (accumulator>>bits)&maxValue)
		}
	}
	return result
}

func createChecksum(uint5Array []int) []int {
	values := append(uint5Array, []int{0, 0, 0, 0, 0, 0}...) //nolint:gocritic // prefer if/else
	mod := polymod(values) ^ 1
	result := []int{}
	for p := 0; p < 6; p++ {
		result = append(result, ((mod >> (5 * (5 - p))) & 31))
	}
	return result
}

// See for details: https://github.com/LiskHQ/lips/blob/master/proposals/lip-0018.md#creating-checksum
func polymod(uint5Array []int) int {
	chk := 1
	for _, value := range uint5Array {
		top := chk >> 25
		chk = ((chk & 0x1ffffff) << 5) ^ value
		for i := 0; i < 5; i += 1 {
			if (top>>i)&1 != 0 {
				chk ^= generator[i]
			}
		}
	}
	return chk
}

func uint5ToLisk32(values []int) string {
	result := ""
	for _, v := range values {
		c := lisk32Charset[v]
		result += string(c)
	}
	return result
}

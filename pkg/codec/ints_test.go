package codec

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testUint64StrData struct {
	Val []UInt64Str `json:"val"`
}

func TestUInt64Str(t *testing.T) {
	obj := &testUint64StrData{
		Val: []UInt64Str{10000, 0, 1234},
	}

	marshaled, err := json.Marshal(obj)
	assert.NoError(t, err)
	assert.Equal(t, "{\"val\":[\"10000\",\"0\",\"1234\"]}", string(marshaled))

	data := &testUint64StrData{}
	err = json.Unmarshal(marshaled, data)
	assert.NoError(t, err)
	assert.Equal(t, obj.Val, data.Val)

	assert.Contains(t, json.Unmarshal([]byte("{\"val\":[\"10000\",\"0\",\"ab\"]}"), data).Error(), "strconv.ParseInt")
}

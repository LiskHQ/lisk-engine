package codec_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTxDecodeStrict(t *testing.T) {
	cases := []struct {
		desc  string
		input string
		err   string
	}{
		{
			desc:  "Valid transaction",
			input: "0a0673616d706c65120465786563180220b9602a20e30b27ee6981f10bfdcedac758f7f14c2c9bf9513a6f4028c994f4b546315381320ca91a76218197730e06d1eed9",
			err:   "",
		},
		{
			desc:  "Valid transaction with extra bytes",
			input: "0a0673616d706c65120465786563180220b9602a20e30b27ee6981f10bfdcedac758f7f14c2c9bf9513a6f4028c994f4b546315381320ca91a76218197730e06d1eed9121314",
			err:   "unread bytes exist",
		},
		{
			desc:  "Valid transaction with missing key",
			input: "0a0673616d706c65120465786563180220b96a20e30b27ee6981f10bfdcedac758f7f14c2c9bf9513a6f4028c994f4b546315381320ca91a76218197730e06d1eed9",
			err:   "unexpected field number found",
		},
	}
	for _, c := range cases {
		t.Log(c.desc)
		val, err := hex.DecodeString(c.input)
		assert.NoError(t, err)
		tx := &Transaction{}
		err = tx.DecodeStrict(val)
		if c.err == "" {
			assert.NoError(t, err)
		} else {
			assert.Contains(t, err.Error(), c.err)
		}
	}
}

package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDerivationPath(t *testing.T) {
	cases := []struct {
		input    string
		expected []int
		errStr   string
	}{
		{
			input:    "am/32",
			expected: []int{},
			errStr:   "derivation path must start from `m`",
		},
		{
			input:    "m/32/'31",
			expected: []int{},
			errStr:   "invalid segment format",
		},
		{
			input:    "m/2'/1x",
			expected: []int{},
			errStr:   "invalid segment format for 1x",
		},
		{
			input:    "m/4294967295'/1x",
			expected: []int{},
			errStr:   "segment 4294967295' exceeds max uint32 / 2",
		},
		{
			input:    "m/332'/a",
			expected: []int{},
			errStr:   "invalid segment format",
		},
		{
			input:    "m/13343",
			expected: []int{13343},
			errStr:   "",
		},
		{
			input:    "m/44'/134'/0'",
			expected: []int{44 + hardendOffset, 134 + hardendOffset, hardendOffset},
			errStr:   "",
		},
	}

	for _, testCase := range cases {
		t.Logf("Testing input %s", testCase.input)
		result, err := parseDerivationPath(testCase.input)
		if testCase.errStr != "" {
			assert.Contains(t, err.Error(), testCase.errStr)
			continue
		}
		assert.NoError(t, err)
		assert.Equal(t, result, testCase.expected)
	}
}

func TestDeriveEd25519Key(t *testing.T) {
	cases := []struct {
		recoveryPhrase string
		derivationPath string
		privateKey     []byte
		errStr         string
	}{
		{
			recoveryPhrase: "target cancel solution recipe vague faint bomb convince pink vendor fresh patrol",
			derivationPath: "m/44'/134'/0'",
			privateKey:     strToHex("0xc465dfb15018d3aef0d94d411df048e240e87a3ec9cd6d422cea903bfc101f61c6bae83af23540096ac58d5121b00f33be6f02f05df785766725acdd5d48be9d"),
		},
		{
			recoveryPhrase: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon art",
			derivationPath: "m/44'/134'/0'",
			privateKey:     strToHex("0x111b6146ec9fbfd7631c75bf42de7c020837d905323a1c161352efed680e86a94815aaeb2da9e7485bfd4f43a5a57431d78fd9e2a3545f9aa6f131ff35ee57b0"),
		},
		{
			recoveryPhrase: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon art",
			derivationPath: "m/44'/134'/1'",
			privateKey:     strToHex("0x544a796e02833f9b6fe90512a8fe48360924a9a5462a5e263a3a40092dae99f50ad5733ff582886700791aed326ff226e1c04ab5b683facb082b36594b7eddb1"),
		},
	}

	for i, testCase := range cases {
		t.Logf("Testing case %d", i+1)
		pk, err := DeriveEd25519Key(testCase.recoveryPhrase, testCase.derivationPath)
		assert.NoError(t, err)
		assert.Equal(t, testCase.privateKey, pk)
	}
}

func TestDeriveBLSKey(t *testing.T) {
	cases := []struct {
		recoveryPhrase string
		derivationPath string
		privateKey     []byte
		errStr         string
	}{
		{
			recoveryPhrase: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
			derivationPath: "m/12381",
			privateKey:     strToHex("0x3cde49b9640cd34170877e3df098d2d5d2260951403b263d180fdfa80e7d4bb4"),
		},
	}

	for i, testCase := range cases {
		t.Logf("Testing case %d", i+1)
		pk, err := DeriveBLSKey(testCase.recoveryPhrase, testCase.derivationPath)
		assert.NoError(t, err)
		assert.Equal(t, testCase.privateKey, pk)
	}
}

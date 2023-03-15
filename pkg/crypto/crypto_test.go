package crypto

import (
	"encoding/hex"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseEncryptedPassphrase(t *testing.T) {
	encryptedPassphrase := "salt=f3b475d796e5f08d69a1290a4f701692&cipherText=a4e670&iv=92f8c3778306cf2f4d03b05c&tag=06e473e10ffc28b402c907ee682e3a36&version=1"
	res, err := ParseEncryptedPassphrase(encryptedPassphrase)
	if err != nil {
		t.Error(err)
	}
	if res.CipherText != "a4e670" {
		t.Errorf("expected cipher text to be %s but got %s", "a4e670", res.CipherText)
	}
}

func TestDecryptPassphraseWithPassword(t *testing.T) {
	encryptedPassphrase := "salt=bfe37e566970ed1b82fbf1e758f29e4d&cipherText=0dab01226e5b7534&iv=f4feff3e18a52700233a6754&tag=9b5870a9052736904e4b2251bda03486&version=1"
	password := "123"
	res, err := ParseEncryptedPassphrase(encryptedPassphrase)
	if err != nil {
		t.Error(err)
		return
	}
	passphrase, err := DecryptPassphraseWithPassword(res, password)
	if err != nil {
		t.Error(err)
		return
	}
	assert.Equal(t, "11101986", passphrase)
}

func TestEncryptPassphraseWithPassword(t *testing.T) {
	encryptedPass, err := EncryptPassphraseWithPassword("11101986", "123", 1)
	assert.NoError(t, err)

	decrypted, err := DecryptPassphraseWithPassword(encryptedPass, "123")
	assert.NoError(t, err)
	assert.Equal(t, "11101986", decrypted)

	queryMap, err := url.ParseQuery(encryptedPass.String())
	assert.NoError(t, err)
	assert.Equal(t, "1", queryMap.Get("iterations"))
}

func TestKeyGeneration(t *testing.T) {
	passphrase := "endless focus guilt bronze hold economy bulk parent soon tower cement venue"
	publicKey, privateKey, err := GetKeys(passphrase)
	assert.Nil(t, err)
	assert.Equal(t, "508a965871253595b36e2f8dc27bff6e67b39bdd466531be9c6f8c401253979c", hex.EncodeToString(publicKey))
	assert.Equal(t, "a30c9e2b10599702b985d18fee55721b56691877cd2c70bbdc1911818dabc9b9508a965871253595b36e2f8dc27bff6e67b39bdd466531be9c6f8c401253979c", hex.EncodeToString(privateKey))
}

func TestHashOnion(t *testing.T) {
	seed := GetHashOnionSeed()

	hashOnions := HashOnion(seed, 100, 1)
	assert.Len(t, hashOnions, 101)
	for i := range hashOnions {
		if len(hashOnions) > i+1 {
			assert.Equal(t, hashOnions[i], Hash(hashOnions[i+1])[:16])
		}
	}
}

func TestGetAddress(t *testing.T) {
	pk, _ := hexToBytes("af52aaba65b2e71b01f8eadd748e56d877b7e555376ae389922e3a0ab4f5bee0")
	address, _ := hexToBytes("24dad3958695e86b1ad5b4ef4c7de8108abf1ba6")
	assert.Equal(t, address, GetAddress(pk))
}

func TestSignature(t *testing.T) {
	pk, _ := hexToBytes("af52aaba65b2e71b01f8eadd748e56d877b7e555376ae389922e3a0ab4f5bee0")
	sk, _ := hexToBytes("81076308b1be76842f0bbcdc8659647400a14c193c333582a649bfda130856e3af52aaba65b2e71b01f8eadd748e56d877b7e555376ae389922e3a0ab4f5bee0")
	signature := Sign(sk, []byte("message"))

	err := VerifySignature(pk, signature, []byte("message"))
	assert.NoError(t, err)
}

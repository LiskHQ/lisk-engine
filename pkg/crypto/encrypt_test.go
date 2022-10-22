package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncryptDecryptMessage(t *testing.T) {
	message := []byte("secret")
	encryptedMsg, err := EncryptMessageWithPassword(message, "passwd", nil)
	assert.NoError(t, err)
	assert.NoError(t, encryptedMsg.Validate())

	decrypted, err := DecryptMessageWithPassword(encryptedMsg, "passwd")
	assert.NoError(t, err)
	assert.Equal(t, message, decrypted)
}

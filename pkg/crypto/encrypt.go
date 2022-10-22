package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

const (
	CipherAES256GCM = "aes-256-gcm"
	KDFArgon2ID     = "argon2id"
)

type KDFParams struct {
	Parallelism uint32    `json:"parallelism" fieldNumber:"1"`
	Iterations  uint32    `json:"iterations" fieldNumber:"2"`
	MemorySize  uint32    `json:"memorySize" fieldNumber:"3"`
	Salt        codec.Hex `json:"salt" fieldNumber:"4"`
}

func (e KDFParams) String() string {
	val := `
	Parallelism: %d
	Iterations: %d
	MemorySize: %d
	Salt: %s
	`
	return fmt.Sprintf(
		val,
		e.Parallelism,
		e.Iterations,
		e.MemorySize,
		e.Salt,
	)
}

func (e KDFParams) Validate() error {
	if len(e.Salt) == 0 {
		return errors.New("salt cannot be empty")
	}
	return nil
}

type CipherParams struct {
	IV  codec.Hex `json:"iv" fieldNumber:"1"`
	Tag codec.Hex `json:"tag" fieldNumber:"2"`
}

func (e CipherParams) String() string {
	val := `
	iv: %s
	tag: %s
	`
	return fmt.Sprintf(
		val,
		e.IV,
		e.Tag,
	)
}

func (e CipherParams) Validate() error {
	if len(e.IV) == 0 {
		return errors.New("iv cannot be empty")
	}
	if len(e.Tag) == 0 {
		return errors.New("tag cannot be empty")
	}
	return nil
}

type EncryptedMessage struct {
	Version      string        `json:"version" fieldNumber:"1"`
	CipherText   codec.Hex     `json:"cipherText" fieldNumber:"2"`
	Mac          codec.Hex     `json:"mac" fieldNumber:"3"`
	KDF          string        `json:"kdf" fieldNumber:"4"`
	KDFParams    *KDFParams    `json:"kdfparams" fieldNumber:"5"`
	Cipher       string        `json:"cipher" fieldNumber:"6"`
	CipherParams *CipherParams `json:"cipherparams" fieldNumber:"7"`
}

func (e EncryptedMessage) String() string {
	val := `
	Version: %s
	CipherText: %s
	Mac: %s
	KDF: %s
	KDFParams: %s
	Cipher: %s
	CipherParams: %s
	`
	return fmt.Sprintf(
		val,
		e.Version,
		e.CipherText,
		e.Mac,
		e.KDF,
		e.KDFParams,
		e.Cipher,
		e.CipherParams,
	)
}

func (e EncryptedMessage) Validate() error {
	if e.Version != "1" {
		return fmt.Errorf("version must be `1` but received %s", e.Version)
	}
	if e.KDF != KDFArgon2ID {
		return fmt.Errorf("only %s is supported for KDF", KDFArgon2ID)
	}
	if e.Cipher != CipherAES256GCM {
		return fmt.Errorf("only %s is supported for Cipher", CipherAES256GCM)
	}
	if e.KDFParams == nil {
		return errors.New("kdfparams cannot be empty")
	}
	if err := e.KDFParams.Validate(); err != nil {
		return err
	}
	if e.CipherParams == nil {
		return errors.New("cipherparams cannot be empty")
	}
	if err := e.CipherParams.Validate(); err != nil {
		return err
	}
	return nil
}

type EncryptOptions struct {
	KDF         string
	Parallelism uint8
	Iterations  uint32
	MemorySize  uint32
}

func DefaultEncryptOptions() *EncryptOptions {
	return &EncryptOptions{
		KDF:         KDFArgon2ID,
		Parallelism: 4,
		Iterations:  1,
		MemorySize:  2024,
	}
}

func EncryptMessageWithPassword(message []byte, password string, options *EncryptOptions) (*EncryptedMessage, error) {
	if options == nil {
		options = DefaultEncryptOptions()
	}
	iv := make([]byte, 12)
	_, err := rand.Read(iv)
	if err != nil {
		return nil, err
	}
	salt := make([]byte, 16)
	if err != nil {
		return nil, err
	}
	_, err = rand.Read(salt)
	if err != nil {
		return nil, err
	}
	key := argon2.IDKey([]byte(password), salt, options.Iterations, options.MemorySize, options.Parallelism, 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aecgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	cipherTextWithTag := aecgcm.Seal(nil, iv, message, nil)
	cipherText := cipherTextWithTag[0 : len(cipherTextWithTag)-16]
	tag := cipherTextWithTag[len(cipherTextWithTag)-16:]
	encrypted := &EncryptedMessage{
		Mac: calculateMac(key, cipherText),
		KDF: options.KDF,
		KDFParams: &KDFParams{
			Parallelism: uint32(options.Parallelism),
			Iterations:  options.Iterations,
			MemorySize:  options.MemorySize,
			Salt:        salt,
		},
		Cipher:     CipherAES256GCM,
		CipherText: codec.Hex(cipherText),
		CipherParams: &CipherParams{
			IV:  iv,
			Tag: tag,
		},
		Version: "1",
	}
	return encrypted, nil
}

func DecryptMessageWithPassword(encryptedMessage *EncryptedMessage, password string) ([]byte, error) {
	if encryptedMessage == nil {
		return nil, errors.New("encryptedPassphrase cannot be nil")
	}
	if err := encryptedMessage.Validate(); err != nil {
		return nil, err
	}
	key := argon2.IDKey([]byte(password), encryptedMessage.KDFParams.Salt, encryptedMessage.KDFParams.Iterations, encryptedMessage.KDFParams.MemorySize, uint8(encryptedMessage.KDFParams.Parallelism), 32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aecgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ciphertext := bytes.Join(encryptedMessage.CipherText, encryptedMessage.CipherParams.Tag)
	res, err := aecgcm.Open(nil, encryptedMessage.CipherParams.IV, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func calculateMac(key, cipherText []byte) []byte {
	hasher := sha256.New()
	hasher.Write(key[16:32])
	hasher.Write(cipherText)
	return hasher.Sum(nil)
}

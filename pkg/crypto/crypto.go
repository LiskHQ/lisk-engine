// Package crypto provides crypto related utility functions.
//
// It supports ed25519, BLS for signature scheme, sha256 for hash, argon2id and pbkdf2 for encryption.
package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	ed "golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/pbkdf2"
)

const (
	// PBKDF2Iterations is default iteration for KDF.
	PBKDF2Iterations = 1000000
	// PBKDF2Iterations is default key length.
	PBKDF2KeyLen       = 32
	HashLengh          = 32
	EdPublicKeyLength  = 32
	EdPrivateKeyLength = 64
	EdSignatureLength  = 64
)

func RandomBytes(size int) []byte {
	r := make([]byte, size)
	if _, err := rand.Read(r); err != nil {
		panic(err)
	}
	return r
}

func GetKeys(passphrase string) ([]byte, []byte, error) {
	passphraseHash := Hash([]byte(passphrase))
	randReader := bytes.NewReader(passphraseHash)
	senderPublicKey, senderPrivateKey, err := ed.GenerateKey(randReader)
	if err != nil {
		return nil, nil, err
	}
	return senderPublicKey[:], senderPrivateKey[:], nil
}

func GetEdPublicKey(privateKey []byte) []byte {
	if len(privateKey) == 64 {
		return privateKey[32:]
	}
	pk := ed.NewKeyFromSeed(privateKey)
	return pk.Public().([]byte)
}

func GetAddress(publicKey []byte) []byte {
	publicKeyHash := Hash(publicKey)
	return publicKeyHash[:20]
}

func Hash(key []byte) []byte {
	hasher := sha256.New()
	hasher.Write(key)
	return hasher.Sum(nil)
}

func hexString(bytes []byte) string {
	return hex.EncodeToString(bytes)
}

func hexToBytes(key string) ([]byte, error) {
	return hex.DecodeString(key)
}

func Sign(privateKey []byte, message []byte) ([]byte, error) {
	signature := ed.Sign(privateKey, message)
	return signature, nil
}

func VerifySignature(publicKey, signature []byte, message []byte) error {
	if valid := ed.Verify(publicKey, message, signature); !valid {
		return fmt.Errorf("invalid signature %s by %s", signature, publicKey)
	}
	return nil
}

type EncryptedPassphrase struct {
	Iterations int
	Salt       string
	CipherText string
	IV         string
	Tag        string
	Version    string
}

func (ep EncryptedPassphrase) String() string {
	query := url.Values{}
	query.Add("iterations", strconv.Itoa(ep.Iterations))
	query.Add("salt", ep.Salt)
	query.Add("cipherText", ep.CipherText)
	query.Add("iv", ep.IV)
	query.Add("tag", ep.Tag)
	query.Add("version", ep.Version)
	return query.Encode()
}

func ParseEncryptedPassphrase(encryptedPassphrase string) (*EncryptedPassphrase, error) {
	queryMap, err := url.ParseQuery(encryptedPassphrase)
	if err != nil {
		return nil, err
	}
	ep := &EncryptedPassphrase{}
	if val, exist := queryMap["iterations"]; exist && len(val) > 0 {
		iterations, err := strconv.Atoi(val[0])
		if err != nil {
			return nil, err
		}
		ep.Iterations = iterations
	}
	if val, err := getURLValue("salt", queryMap); err != nil {
		return nil, err
	} else {
		ep.Salt = val
	}
	if val, err := getURLValue("cipherText", queryMap); err != nil {
		return nil, err
	} else {
		ep.CipherText = val
	}
	if val, err := getURLValue("iv", queryMap); err != nil {
		return nil, err
	} else {
		ep.IV = val
	}
	if val, err := getURLValue("tag", queryMap); err != nil {
		return nil, err
	} else {
		ep.Tag = val
	}
	if val, err := getURLValue("version", queryMap); err != nil {
		return nil, err
	} else {
		ep.Version = val
	}
	return ep, nil
}

func EncryptPassphraseWithPassword(passphrase, password string, iteration int) (*EncryptedPassphrase, error) {
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
	key := pbkdf2.Key([]byte(password), salt, iteration, PBKDF2KeyLen, sha256.New)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aecgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	cipherTextWithTag := aecgcm.Seal(nil, iv, []byte(passphrase), nil)
	cipherText := cipherTextWithTag[0 : len(cipherTextWithTag)-16]
	tag := cipherTextWithTag[len(cipherTextWithTag)-16:]
	encrypted := &EncryptedPassphrase{
		Iterations: iteration,
		Salt:       hexString(salt),
		CipherText: hexString(cipherText),
		IV:         hexString(iv),
		Tag:        hexString(tag),
		Version:    "1",
	}
	return encrypted, nil
}

func DecryptPassphraseWithPassword(encryptedPassphrase *EncryptedPassphrase, password string) (string, error) {
	if encryptedPassphrase == nil {
		return "", errors.New("encryptedPassphrase cannot be nil")
	}
	if encryptedPassphrase.Iterations == 0 {
		encryptedPassphrase.Iterations = PBKDF2Iterations
	}
	saltBuffer, err := hexToBytes(encryptedPassphrase.Salt)
	if err != nil {
		return "", err
	}
	key := pbkdf2.Key([]byte(password), saltBuffer, encryptedPassphrase.Iterations, PBKDF2KeyLen, sha256.New)
	tagBuffer, err := hexToBytes(encryptedPassphrase.Tag)
	if err != nil {
		return "", err
	}
	ivBuffer, err := hexToBytes(encryptedPassphrase.IV)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aecgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ciphertext, err := hexToBytes(encryptedPassphrase.CipherText)
	if err != nil {
		return "", err
	}
	ciphertext = append(ciphertext, tagBuffer...)
	res, err := aecgcm.Open(nil, ivBuffer, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func getURLValue(key string, queryMap url.Values) (string, error) {
	if val, exist := queryMap[key]; exist && len(val) > 0 {
		return val[0], nil
	}
	return "", fmt.Errorf("cannot find key %s from query", key)
}

func GetHashOnionSeed() []byte {
	return Hash(RandomBytes(64))[:16]
}

func HashOnion(seed []byte, count int, distance int) [][]byte {
	if count < distance {
		panic("Invalid count or distance. Count must be greater than distance")
	}
	if count%distance != 0 {
		panic("Invalid count. Count must be multiple of distance")
	}
	previousHash := seed
	hashes := [][]byte{seed}
	for i := 1; i <= count; i++ {
		next := Hash(previousHash)[:16]
		if i%distance == 0 {
			hashes = append(hashes, next)
		}
		previousHash = next
	}
	// reverse the hash array
	for i := len(hashes)/2 - 1; i >= 0; i-- {
		opp := len(hashes) - 1 - i
		hashes[i], hashes[opp] = hashes[opp], hashes[i]
	}
	return hashes
}

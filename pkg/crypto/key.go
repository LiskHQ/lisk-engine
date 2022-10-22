package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash"
	"math"
	"regexp"
	"strconv"
	"strings"

	blst "github.com/supranational/blst/bindings/go"
	"github.com/tyler-smith/go-bip39"
	ed "golang.org/x/crypto/ed25519"

	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
)

const (
	hardendOffset = 0x80000000
)

func parseDerivationPath(path string) ([]int, error) {
	if string(path[0]) != "m" {
		return nil, errors.New("derivation path must start from `m`")
	}
	segments := strings.Split(path, "/")
	result := make([]int, len(segments)-1)
	keyPathRegex := regexp.MustCompile("^[0-9]+'?$")
	for i, segment := range segments {
		// first segment is m
		if i == 0 {
			continue
		}
		if segment == "" {
			return nil, errors.New("each segment cannot be empty")
		}
		if !keyPathRegex.MatchString(segment) {
			return nil, fmt.Errorf("invalid segment format for %s", segment)
		}
		if string(segment[len(segment)-1]) == "'" {
			val, err := strconv.Atoi(segment[:len(segment)-1])
			if err != nil {
				return nil, err
			}
			if val > math.MaxUint32/2 {
				return nil, fmt.Errorf("segment %s exceeds max uint32 / 2", segment)
			}
			result[i-1] = val + hardendOffset
			continue
		}
		val, err := strconv.Atoi(segment)
		if err != nil {
			return nil, err
		}
		result[i-1] = val
	}
	return result, nil
}

func DeriveEd25519Key(recoveryPhrase, path string) ([]byte, error) {
	derivationPath, err := parseDerivationPath(path)
	if err != nil {
		return nil, err
	}
	seed := bip39.NewSeed(recoveryPhrase, "")
	key, chainCode, err := getEd25519MasterKey(seed)
	if err != nil {
		return nil, err
	}
	for _, segment := range derivationPath {
		var err error
		key, chainCode, err = getEd25519ChildKey(key, chainCode, segment)
		if err != nil {
			return nil, err
		}
	}

	_, sk, err := ed.GenerateKey(bytes.NewReader(key))
	if err != nil {
		return nil, err
	}
	return sk, nil
}

func hmacHash(hasher func() hash.Hash, key, message []byte) ([]byte, error) {
	hmacer := hmac.New(hasher, key)
	if _, err := hmacer.Write(message); err != nil {
		return nil, err
	}
	result := hmacer.Sum(nil)
	return result, nil
}

func getEd25519MasterKey(seed []byte) ([]byte, []byte, error) {
	result, err := hmacHash(sha512.New, []byte("ed25519 seed"), seed)
	if err != nil {
		return nil, nil, err
	}
	return result[:32], result[32:], nil
}

func getEd25519ChildKey(key, chainCode []byte, index int) ([]byte, []byte, error) {
	indexBytes := bytes.FromUint32(uint32(index))

	hmacer := hmac.New(sha512.New, chainCode)
	if _, err := hmacer.Write(bytes.Join([]byte{0}, key, indexBytes)); err != nil {
		return nil, nil, err
	}
	result := hmacer.Sum(nil)
	return result[:32], result[32:], nil
}

func DeriveBLSKey(recoveryPhrase, path string) ([]byte, error) {
	derivationPath, err := parseDerivationPath(path)
	if err != nil {
		return nil, err
	}
	seed := bip39.NewSeed(recoveryPhrase, "")
	key := getBLSMasterKey(seed)
	if err != nil {
		return nil, err
	}
	for _, segment := range derivationPath {
		var err error
		key, err = getBLSChildKey(key, segment)
		if err != nil {
			return nil, err
		}
	}
	return key, nil
}

func getBLSMasterKey(seed []byte) []byte {
	sk := blst.KeyGen(seed)
	return sk.Serialize()
}

func getBLSChildKey(ikm []byte, index int) ([]byte, error) {
	salt := bytes.FromUint32(uint32(index))
	lamport0, err := ikmToLamport(salt, ikm)
	if err != nil {
		return nil, err
	}
	hashedLamport0 := hashAll(lamport0)
	lamport1, err := ikmToLamport(salt, flipBits(ikm))
	if err != nil {
		return nil, err
	}
	hashedLamport1 := hashAll(lamport1)
	combined := append(hashedLamport0, hashedLamport1...) //nolint:gocritic // prefer if/else
	lamportPK := Hash(bytes.Join(combined...))
	sk := blst.KeyGen(lamportPK)

	return sk.Serialize(), nil
}

func hkdfExtract(salt, ikm []byte) ([]byte, error) {
	return hmacHash(sha256.New, salt, ikm)
}

func hkdfExpand(prk, info []byte, l int) ([]byte, error) {
	t := []byte{}
	okm := []byte{}
	for i := 0; i < int(math.Ceil(float64(l)/32)); i += 1 {
		var err error
		t, err = hmacHash(sha256.New, prk, bytes.Join(t, info, []byte{uint8(1 + i)}))
		if err != nil {
			return nil, err
		}
		okm = bytes.Join(okm, t)
	}
	return okm[:l], nil
}

func ikmToLamport(salt, ikm []byte) ([][]byte, error) {
	info := []byte{}
	prk, err := hkdfExtract(salt, ikm)
	if err != nil {
		return nil, err
	}
	okm, err := hkdfExpand(prk, info, 32*255)
	if err != nil {
		return nil, err
	}
	result := [][]byte{}
	for i := 0; i < 255; i += 1 {
		next := okm[i*32 : (i+1)*32]
		result = append(result, next)
	}
	return result, nil
}

func hashAll(input [][]byte) [][]byte {
	result := make([][]byte, len(input))
	for i, in := range input {
		result[i] = Hash(in)
	}
	return result
}

func flipBits(input []byte) []byte {
	result := make([]byte, len(input))
	for i, in := range input {
		result[i] = in ^ 0xff
	}
	return result
}

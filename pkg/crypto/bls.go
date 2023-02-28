package crypto

import (
	"math"

	blst "github.com/supranational/blst/bindings/go"

	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
)

const (
	BLSPublicKeyLength = 48
	BLSSignatureLength = 96
)

type BLSPublicKey = blst.P1Affine
type BLSSignature = blst.P2Affine
type BLSSecretKey = blst.SecretKey
type BLSAggregateSignature = blst.P2Aggregate
type BLSAggregatePublicKey = blst.P1Aggregate

// BLSKeyPair is a container for a BLS Keypair.
type BLSKeyPair struct {
	PublicKey  []byte
	PrivateKey []byte
}

type BLSPublicKeySignaturePair struct {
	PublicKey []byte
	Signature []byte
}

var dst = []byte("BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_")
var dstPop = []byte("BLS_POP_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_")

// BLSRandomKeyGen returns a random BLSKeyPair.
func BLSRandomKeyGen() *BLSKeyPair {
	randomBytes := RandomBytes(32)
	sk := blst.KeyGen(randomBytes)
	pk := new(BLSPublicKey).From(sk)
	return &BLSKeyPair{
		PublicKey:  pk.Compress(),
		PrivateKey: sk.Serialize(),
	}
}

// BLSKeyGen generates a BLSKeyPair from the passphrase provided.
func BLSKeyGen(passphrase []byte) *BLSKeyPair {
	if len(passphrase) < 32 {
		panic("passphrase must be longer than 32")
	}
	sk := blst.KeyGen(passphrase)
	pk := new(BLSPublicKey).From(sk)
	return &BLSKeyPair{
		PublicKey:  pk.Compress(),
		PrivateKey: sk.Serialize(),
	}
}

func BLSSign(msg []byte, privateKey []byte) []byte {
	sk := new(BLSSecretKey).Deserialize(privateKey)
	if sk == nil {
		return []byte{}
	}
	signature := new(BLSSignature).Sign(sk, msg, dst)
	return signature.Compress()
}

func BLSVerify(msg, signature, publicKey []byte) bool {
	pk := new(BLSPublicKey).Uncompress(publicKey)
	sign := new(BLSSignature).Uncompress(signature)
	return sign.Verify(true, pk, true, msg, dst)
}

func BLSSKToPK(privateKey []byte) []byte {
	sk := new(BLSSecretKey).Deserialize(privateKey)
	pk := new(BLSPublicKey).From(sk).Compress()
	return pk
}

func BLSPopProve(privateKey []byte) []byte {
	sk := new(BLSSecretKey).Deserialize(privateKey)
	msg := BLSSKToPK(privateKey)
	signature := new(BLSSignature).Sign(sk, msg, dstPop)
	return signature.Compress()
}

func BLSPopVerify(publicKey, proof []byte) bool {
	pk := new(BLSPublicKey).Uncompress(publicKey)
	if pk == nil {
		return false
	}
	sign := new(BLSSignature).Uncompress(proof)
	return sign.Verify(true, pk, true, publicKey, dstPop)
}

type Bits []byte

func (b *Bits) write(i int, val bool) {
	byteIndex := int(math.Floor(float64(i) / 8))
	bitIndex := i % 8

	original := *b
	if val {
		original[byteIndex] |= 1 << bitIndex
	} else {
		original[byteIndex] &= ^(1 << bitIndex)
	}
	*b = original
}
func (b Bits) read(i int) bool {
	byteIndex := int(math.Floor(float64(i) / 8))
	bitIndex := i % 8
	return (b[byteIndex]>>bitIndex)%2 == 1
}

func BLSCreateAggSig(keysList [][]byte, pubKeySignaturePairs []*BLSPublicKeySignaturePair) ([]byte, []byte) {
	aggregationBits := make(Bits, int(math.Ceil(float64(len(keysList))/8)))
	signatures := make([]*BLSSignature, len(pubKeySignaturePairs))
	for i, pair := range pubKeySignaturePairs {
		sign := new(BLSSignature).Uncompress(pair.Signature)
		signatures[i] = sign
		keyIndex := bytes.FindIndex(keysList, pair.PublicKey)
		if keyIndex >= 0 {
			aggregationBits.write(keyIndex, true)
		}
	}
	signature := new(BLSAggregateSignature)
	signature.Aggregate(signatures, false)
	return aggregationBits, signature.ToAffine().Compress()
}

func BLSVerifyAggSig(keysList [][]byte, aggregationBits []byte, signature []byte, message []byte) bool {
	keys := []*BLSPublicKey{}
	for i := 0; i < len(keysList); i++ {
		if Bits(aggregationBits).read(i) {
			keys = append(keys, new(BLSPublicKey).Uncompress(keysList[i]))
		}
	}
	aggSig := new(BLSSignature).Uncompress(signature)
	return aggSig.FastAggregateVerify(true, keys, message, dst)
}

func BLSVerifyWeightedAggSig(keysList [][]byte, aggregationBits []byte, signature []byte, weights []uint64, threshold uint64, message []byte) bool {
	keys := []*BLSPublicKey{}
	weightSum := uint64(0)
	for i := 0; i < len(keysList); i++ {
		if Bits(aggregationBits).read(i) {
			keys = append(keys, new(BLSPublicKey).Uncompress(keysList[i]))
			weightSum += weights[i]
		}
	}
	if weightSum < threshold {
		return false
	}
	aggSig := new(BLSSignature).Uncompress(signature)
	return aggSig.FastAggregateVerify(true, keys, message, dst)
}

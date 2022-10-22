package generator

import (
	"errors"
	"fmt"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

const (
	KeyTypePlain     = "plain"
	KeyTypeEncrypted = "encrypted"
)

type Keypair struct {
	address              []byte `fieldNumber:"1"`
	generationPublicKey  []byte `fieldNumber:"2"`
	generationPrivateKey []byte `fieldNumber:"3"`
	blsPrivateKey        []byte `fieldNumber:"4"`
}

func NewKeypair(
	address []byte,
	generationPublicKey []byte,
	generationPrivateKey []byte,
	blsPrivateKey []byte,
) *Keypair {
	return &Keypair{
		address:              address,
		generationPublicKey:  generationPublicKey,
		generationPrivateKey: generationPrivateKey,
		blsPrivateKey:        blsPrivateKey,
	}
}

type GeneratorInfo struct {
	Height             uint32 `json:"height" fieldNumber:"1"`
	MaxHeightPrevoted  uint32 `json:"maxHeightPrevoted" fieldNumber:"2"`
	MaxHeightGenerated uint32 `json:"maxHeightGenerated" fieldNumber:"3"`
}

func (f GeneratorInfo) IsZero() bool {
	return f.Height == 0 && f.MaxHeightPrevoted == 0 && f.MaxHeightGenerated == 0
}
func (f GeneratorInfo) Equal(val *GeneratorInfo) bool {
	return f.Height == val.Height && f.MaxHeightPrevoted == val.MaxHeightPrevoted && f.MaxHeightGenerated == val.MaxHeightGenerated
}

type Keys struct {
	Address codec.Lisk32 `json:"address" fieldNumber:"1"`
	Type    string       `json:"type" fieldNumber:"2"`
	Data    []byte       `json:"data" fieldNumber:"3"`
}

func (k Keys) Validate() error {
	if k.Type != KeyTypePlain && k.Type != KeyTypeEncrypted {
		return fmt.Errorf("invalid keys type %s", k.Type)
	}
	if len(k.Data) == 0 {
		return errors.New("keys data is empty")
	}
	return nil
}
func (k Keys) IsPlain() bool {
	return k.Type == KeyTypePlain
}

func (k Keys) IsEncrypted() bool {
	return k.Type == KeyTypeEncrypted
}

func (k *Keys) PlainKeys() (*PlainKeys, error) {
	if k.Type != KeyTypePlain {
		return nil, errors.New("keys type is not plain")
	}
	plainKeys := &PlainKeys{}
	if err := plainKeys.Decode(k.Data); err != nil {
		return nil, err
	}
	return plainKeys, nil
}

func (k *Keys) EncryptedKeys() (*crypto.EncryptedMessage, error) {
	if k.Type != KeyTypeEncrypted {
		return nil, errors.New("keys type is not encrypted")
	}
	encryptedKeys := &crypto.EncryptedMessage{}
	if err := encryptedKeys.Decode(k.Data); err != nil {
		return nil, err
	}
	return encryptedKeys, nil
}

type PlainKeys struct {
	GeneratorKey        codec.Hex `json:"generatorKey" fieldNumber:"1"`
	GeneratorPrivateKey codec.Hex `json:"generatorPrivateKey" fieldNumber:"2"`
	BLSKey              codec.Hex `json:"blsKey" fieldNumber:"3"`
	BLSPrivateKey       codec.Hex `json:"blsPrivateKey" fieldNumber:"4"`
}

func (k PlainKeys) Validate() error {
	if len(k.GeneratorKey) != 32 {
		return fmt.Errorf("invalid generator key %s", k.GeneratorKey)
	}
	if len(k.GeneratorPrivateKey) != 64 {
		return fmt.Errorf("invalid generator private key %s", k.GeneratorPrivateKey)
	}
	if len(k.BLSKey) != 48 {
		return fmt.Errorf("invalid bls key %s", k.BLSKey)
	}
	if len(k.BLSPrivateKey) != 32 {
		return fmt.Errorf("invalid bls private key %s", k.BLSPrivateKey)
	}
	return nil
}

type KeysFile struct {
	Keys []*KeysFileItem `json:"keys"`
}

type KeysFileItem struct {
	Address   codec.Lisk32             `json:"address"`
	Plain     *PlainKeys               `json:"plain"`
	Encrypted *crypto.EncryptedMessage `json:"encrypted"`
}

package blockchain

import (
	"fmt"
	"regexp"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

const (
	MaxTransactionParamsSize = 14 * 1024
)

var alphanumericRegex = regexp.MustCompile("^[a-zA-Z0-9]*$")

// Transaction holds general transaction.
type Transaction struct {
	ID              codec.Hex   `json:"id"`
	size            int         `json:"-"`
	Module          string      `json:"module" fieldNumber:"1"`
	Command         string      `json:"commandID" fieldNumber:"2"`
	Nonce           uint64      `json:"nonce,string" fieldNumber:"3"`
	Fee             uint64      `json:"fee,string" fieldNumber:"4"`
	SenderPublicKey codec.Hex   `json:"senderPublicKey" fieldNumber:"5"`
	Params          codec.Hex   `json:"params" fieldNumber:"6"`
	Signatures      []codec.Hex `json:"signatures" fieldNumber:"7"`
}

// NewTransaction creates instance from decoded value.
func NewTransaction(value []byte) (*Transaction, error) {
	tx := &Transaction{}
	if err := tx.DecodeStrict(value); err != nil {
		return nil, err
	}
	txBytes, err := tx.Encode()
	if err != nil {
		return nil, err
	}
	id := crypto.Hash(txBytes)
	size := len(txBytes)
	tx.ID = id
	tx.size = size
	return tx, nil
}

func (t *Transaction) Init() error {
	txBytes, err := t.Encode()
	if err != nil {
		return err
	}
	id := crypto.Hash(txBytes)
	size := len(txBytes)
	t.ID = id
	t.size = size
	return nil
}

// Size returns the transaction size.
func (t *Transaction) Size() int                   { return t.size }
func (t *Transaction) SenderAddress() codec.Lisk32 { return crypto.GetAddress(t.SenderPublicKey) }

// Bytes return transaction in bytes.
func (t *Transaction) Bytes() ([]byte, error) {
	return t.Encode()
}

func (t *Transaction) Copy() *Transaction {
	tx := &Transaction{
		ID:              t.ID,
		size:            t.size,
		Module:          t.Module,
		Command:         t.Command,
		Nonce:           t.Nonce,
		Fee:             t.Fee,
		SenderPublicKey: bytes.Copy(t.SenderPublicKey),
		Params:          bytes.Copy(t.Params),
		Signatures:      make([]codec.Hex, len(t.Signatures)),
	}
	for i, sign := range t.Signatures {
		tx.Signatures[i] = bytes.Copy(sign)
	}
	return tx
}
func (t *Transaction) GetSignature(chainID, privateKey []byte) ([]byte, error) {
	signingBytes, err := t.SigningBytes()
	if err != nil {
		return nil, err
	}
	return crypto.Sign(privateKey, crypto.Hash(bytes.Join(TagTransaction, chainID, signingBytes)))
}

// Bytes return transaction in bytes.
func (t *Transaction) Freeze() FrozenTransaction {
	tx := &frozenTransaction{
		id:              t.ID,
		size:            t.size,
		module:          t.Module,
		command:         t.Command,
		nonce:           t.Nonce,
		fee:             t.Fee,
		senderPublicKey: bytes.Copy(t.SenderPublicKey),
		params:          bytes.Copy(t.Params),
	}
	return tx
}

// SigningBytes return signed byte so the transaction.
func (t *Transaction) SigningBytes() ([]byte, error) {
	stx := &SigningTransaction{
		Module:          t.Module,
		Command:         t.Command,
		Nonce:           t.Nonce,
		Fee:             t.Fee,
		SenderPublicKey: t.SenderPublicKey,
		Params:          t.Params,
	}
	return stx.Encode()
}

// Validate transaction statically.
func (t *Transaction) Validate() error {
	if !alphanumericRegex.MatchString(t.Module) {
		return fmt.Errorf("module %s does not satisfy alphanumeric", t.Module)
	}
	if !alphanumericRegex.MatchString(t.Command) {
		return fmt.Errorf("command %s does not satisfy alphanumeric", t.Command)
	}
	if len(t.Params) > MaxTransactionParamsSize {
		return fmt.Errorf("params size %d exceeds maximum %d", len(t.Params), MaxTransactionParamsSize)
	}

	if len(t.SenderPublicKey) != crypto.EdPublicKeyLength {
		return fmt.Errorf("senderPublicKey must have length of 32 but received %d", len(t.SenderPublicKey))
	}
	if len(t.Signatures) == 0 {
		return fmt.Errorf("signatures must have length at least 1 but received %d", len(t.Signatures))
	}
	for _, sig := range t.Signatures {
		if len(sig) != crypto.EdSignatureLength {
			return fmt.Errorf("signatures must have length of 64 but received %d", len(sig))
		}
	}
	return nil
}

// SigningTransaction holds signed part of the transaction.
type SigningTransaction struct {
	Module          string `fieldNumber:"1"`
	Command         string `fieldNumber:"2"`
	Nonce           uint64 `fieldNumber:"3"`
	Fee             uint64 `fieldNumber:"4"`
	SenderPublicKey []byte `fieldNumber:"5"`
	Params          []byte `fieldNumber:"6"`
}

type FrozenTransaction interface {
	Params() []byte
	Module() string
	Command() string
	Fee() uint64
	Nonce() uint64
	SenderAddress() codec.Lisk32
	SenderPublicKey() codec.Hex
	Signatures() [][]byte
	Size() int
	SigningBytes() ([]byte, error)
}

type frozenTransaction struct {
	id              []byte
	size            int
	module          string
	command         string
	nonce           uint64
	fee             uint64
	senderPublicKey []byte
	params          []byte
	signatures      [][]byte
}

func (t *frozenTransaction) ID() []byte                  { return t.id }
func (t *frozenTransaction) Module() string              { return t.module }
func (t *frozenTransaction) Command() string             { return t.command }
func (t *frozenTransaction) Nonce() uint64               { return t.nonce }
func (t *frozenTransaction) Fee() uint64                 { return t.fee }
func (t *frozenTransaction) SenderPublicKey() codec.Hex  { return t.senderPublicKey }
func (t *frozenTransaction) SenderAddress() codec.Lisk32 { return crypto.GetAddress(t.senderPublicKey) }
func (t *frozenTransaction) Params() []byte              { return t.params }
func (t *frozenTransaction) Signatures() [][]byte        { return t.signatures }
func (t *frozenTransaction) Size() int                   { return t.size }
func (t *frozenTransaction) SigningBytes() ([]byte, error) {
	stx := &SigningTransaction{
		Module:          t.module,
		Command:         t.command,
		Nonce:           t.nonce,
		Fee:             t.fee,
		SenderPublicKey: t.senderPublicKey,
		Params:          t.params,
	}
	return stx.Encode()
}

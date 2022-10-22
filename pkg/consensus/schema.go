package consensus

import (
	"bytes"
	"sort"

	"github.com/LiskHQ/lisk-engine/pkg/consensus/certificate"
	"github.com/LiskHQ/lisk-engine/pkg/labi"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen
type EventPostBlock struct {
	Block []byte `fieldNumber:"1"`
}

type EventPostSingleCommits struct {
	SingleCommits []*certificate.SingleCommit `fieldNumber:"1"`
}

type ValidatorWithBLSKey struct {
	Address   []byte
	BFTWeight uint64
	BLSKey    []byte
}

type ValidatorsWithBLSKey []*ValidatorWithBLSKey

func (v *ValidatorsWithBLSKey) sort() {
	original := *v
	sort.Slice(original, func(i, j int) bool {
		return bytes.Compare(original[i].BLSKey, original[j].BLSKey) < 0
	})
	*v = original
}

type ValidatorsHash struct {
	Keys                 [][]byte `fieldNumber:"1"`
	Weights              []uint64 `fieldNumber:"2"`
	CertificateThreshold uint64   `fieldNumber:"3"`
}

type NextValidatorParams struct {
	NextValidators       []*labi.Validator `fieldNumber:"1"`
	PrecommitThreshold   uint64            `fieldNumber:"2"`
	CertificateThreshold uint64            `fieldNumber:"3"`
}

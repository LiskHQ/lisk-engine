package rmt

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

// Proof holds the proof for lisk tree.
type Proof struct {
	Size          uint64   `fieldNumber:"1"`
	Idxs          []uint64 `fieldNumber:"2"`
	SiblingHashes [][]byte `fieldNumber:"3"`
}

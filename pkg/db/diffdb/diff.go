package diffdb

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

// KV is a key value pair.
type KV struct {
	Key   []byte `fieldNumber:"1"`
	Value []byte `fieldNumber:"2"`
}

// Diff holds a change in the state.
type Diff struct {
	// Newly added key
	Added [][]byte `fieldNumber:"1"`
	// Updated key-value pair where value is original value
	Updated []*KV `fieldNumber:"2"`
	// Deleted key-value pair
	Deleted []*KV `fieldNumber:"3"`
}

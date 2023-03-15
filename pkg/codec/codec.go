// Package codec implements serialization and deserialization defined in [LIP-0027] and [LIP-0064] and lisk32 representation of address defined in [LIP-0018].
//
// [LIP-0018]: https://github.com/LiskHQ/lips/blob/main/proposals/lip-0018.md
// [LIP-0027]: https://github.com/LiskHQ/lips/blob/main/proposals/lip-0027.md
// [LIP-0064]: https://github.com/LiskHQ/lips/blob/main/proposals/lip-0064.md
package codec

// Encodable is interface for struct which is encodable
// All generated struct code should have this method.
type Encodable interface {
	Encode() []byte
}

// Decodable is interface for struct which is decodable
// All generated struct code should have this method.
type Decodable interface {
	Decode([]byte) error
}

// DecodableReader is interface for struct which is decodable
// All generated struct code should have this method.
type DecodableReader interface {
	DecodeFromReader(*Reader) error
}

// EncodeDecodable can encode and decode.
type EncodeDecodable interface {
	Encodable
	Decodable
	DecodableReader
}

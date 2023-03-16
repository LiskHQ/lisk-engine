package statemachine

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/db"
	"github.com/LiskHQ/lisk-engine/pkg/db/diffdb"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen
type testObj struct {
	data codec.Hex `fieldNumber:"1"`
}

func TestUtilDecode(t *testing.T) {
	database, _ := db.NewInMemoryDB()
	diffStore := diffdb.New(database, []byte{1})

	diffStore.Set([]byte("invalid"), []byte{3, 2, 1})
	diffStore.Set([]byte("key"), (&testObj{data: codec.Hex("123")}).Encode())

	err := GetDecodable(diffStore, []byte("invalid"), &testObj{})
	assert.EqualError(t, err, "invalid data")
	err = GetDecodableOrDefault(diffStore, []byte("invalid"), &testObj{})
	assert.EqualError(t, err, "invalid data")

	err = GetDecodable(diffStore, []byte("no data"), &testObj{})
	assert.EqualError(t, err, "data was not found")
	err = GetDecodableOrDefault(diffStore, []byte("no data"), &testObj{})
	assert.NoError(t, err)

	result := &testObj{}
	err = GetDecodable(diffStore, []byte("key"), result)
	assert.NoError(t, err)
	assert.Equal(t, codec.Hex("123"), result.data)

	result = &testObj{}
	err = GetDecodableOrDefault(diffStore, []byte("key"), result)
	assert.NoError(t, err)
	assert.Equal(t, codec.Hex("123"), result.data)
}

func TestUtilEncode(t *testing.T) {
	database, _ := db.NewInMemoryDB()
	diffStore := diffdb.New(database, []byte{1})

	err := SetEncodable(diffStore, []byte("key"), &testObj{})
	assert.NoError(t, err)

	exist := diffStore.Has([]byte("key"))
	assert.True(t, exist)
}

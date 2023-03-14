package statemachine

import (
	"errors"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/db"
)

func GetDecodable(diffStore ImmutableStore, key []byte, value codec.Decodable) error {
	resultBytes, exist := diffStore.Get(key)
	if !exist {
		return db.ErrDataNotFound
	}
	if err := value.Decode(resultBytes); err != nil {
		return err
	}
	return nil
}

func GetDecodableOrDefault(diffStore ImmutableStore, key []byte, value codec.Decodable) error {
	err := GetDecodable(diffStore, key, value)
	if err != nil {
		if errors.Is(err, db.ErrDataNotFound) {
			return nil
		}
		return err
	}
	return nil
}

func SetEncodable(diffStore Store, key []byte, value codec.Encodable) error {
	encodedSender, err := value.Encode()
	if err != nil {
		return err
	}
	diffStore.Set(key, encodedSender)
	return nil
}

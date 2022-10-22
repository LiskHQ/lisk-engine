package diffdb

import (
	"errors"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

type getter interface {
	Get([]byte) ([]byte, error)
}

type setter interface {
	Set([]byte, []byte) error
}

func GetDecodable(diffDB getter, key []byte, value codec.Decodable) error {
	resultBytes, err := diffDB.Get(key)
	if err != nil {
		return err
	}
	if err := value.Decode(resultBytes); err != nil {
		return err
	}
	return nil
}

func GetDecodableOrDefault(diffDB getter, key []byte, value codec.Decodable) error {
	err := GetDecodable(diffDB, key, value)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	return nil
}

func SetEncodable(diffDB setter, key []byte, value codec.Encodable) error {
	encodedSender, err := value.Encode()
	if err != nil {
		return err
	}
	if err := diffDB.Set(key, encodedSender); err != nil {
		return err
	}
	return nil
}

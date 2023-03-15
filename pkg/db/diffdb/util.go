package diffdb

import (
	"errors"
	"fmt"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

type getter interface {
	Get([]byte) ([]byte, bool)
}

type setter interface {
	Set([]byte, []byte)
}

func GetDecodable(diffDB getter, key codec.Hex, value codec.Decodable) error {
	resultBytes, exist := diffDB.Get(key)
	if !exist {
		return fmt.Errorf("requested key %s does not exist", key)
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

func SetEncodable(diffDB setter, key []byte, value codec.Encodable) {
	encodedSender := value.Encode()
	diffDB.Set(key, encodedSender)
}
